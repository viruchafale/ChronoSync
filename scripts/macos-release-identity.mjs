import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath, URL } from "node:url";

export const APP_IDENTIFIER = "dev.chronosync.desktop";
export const ENGINE_IDENTIFIER = "dev.chronosync.engine";
export const TEAM_TOKEN = "__CHRONOSYNC_TEAM_IDENTIFIER__";
const ROOT = resolve(fileURLToPath(new URL("..", import.meta.url)));
export const TEMPLATE_PATH = resolve(
  ROOT,
  "apps/desktop/src-tauri/Entitlements.release.plist.template",
);
export const RELEASE_CONFIG_PATH = resolve(
  ROOT,
  "apps/desktop/src-tauri/tauri.release.conf.json",
);

export function validateTeamIdentifier(value) {
  if (!/^[A-Z0-9]{10}$/.test(value ?? "")) {
    throw new Error(
      "TeamIdentifier must be exactly 10 uppercase ASCII letters/digits",
    );
  }
  return value;
}

export function renderEntitlements(template, teamIdentifier) {
  const team = validateTeamIdentifier(teamIdentifier);
  const occurrences = template.split(TEAM_TOKEN).length - 1;
  if (occurrences !== 3) {
    throw new Error(
      "Release entitlement template has an unexpected TeamIdentifier shape",
    );
  }
  const rendered = template.replaceAll(TEAM_TOKEN, team);
  if (rendered.includes(TEAM_TOKEN)) {
    throw new Error(
      "Release entitlement template still contains a placeholder",
    );
  }
  return rendered;
}

export function checkReleaseStructure() {
  const template = readFileSync(TEMPLATE_PATH, "utf8");
  renderEntitlements(template, "ABCDEFGHIJ");
  const config = JSON.parse(readFileSync(RELEASE_CONFIG_PATH, "utf8"));
  const mac = config?.bundle?.macOS;
  if (
    mac?.hardenedRuntime !== true ||
    mac?.minimumSystemVersion !== "10.15" ||
    mac?.entitlements !== "Entitlements.release.plist" ||
    mac?.files?.["embedded.provisionprofile"] !== "embedded.provisionprofile" ||
    "signingIdentity" in mac
  ) {
    throw new Error(
      "Release config must require hardened runtime, macOS 10.15, generated entitlements/profile, and an external signing identity",
    );
  }
  if (!template.includes(`${TEAM_TOKEN}.${APP_IDENTIFIER}`)) {
    throw new Error(
      "Release entitlement template does not bind the ChronoSync app identifier",
    );
  }
  return { appIdentifier: APP_IDENTIFIER, minimumSystemVersion: "10.15" };
}

export function parseCodeSignature(output) {
  const value = (name) =>
    output.match(new RegExp(`^${name}=(.+)$`, "m"))?.[1]?.trim() ?? "";
  return {
    identifier: value("Identifier"),
    teamIdentifier: value("TeamIdentifier"),
    signature: value("Signature"),
    authorities: [...output.matchAll(/^Authority=(.+)$/gm)].map((match) =>
      match[1].trim(),
    ),
    flags:
      output.match(/^CodeDirectory .+ flags=([^\n]+)$/m)?.[1]?.trim() ?? "",
    timestamp: value("Timestamp") || value("Signed Time"),
  };
}

export function parseIdentityInventory(output) {
  const identities = [
    ...output.matchAll(/^\s*\d+\)\s+[0-9A-F]+\s+"([^"]+)"/gm),
  ].map((match) => match[1]);
  const developerIdApplications = identities
    .filter((identity) => identity.startsWith("Developer ID Application:"))
    .map((identity) => ({
      identity,
      teamIdentifier: identity.match(/\(([A-Z0-9]{10})\)$/)?.[1] ?? "",
    }));
  const reportedCount = Number(
    output.match(/^\s*(\d+) valid identities found\s*$/m)?.[1] ?? 0,
  );
  return {
    validIdentityCount: reportedCount,
    developerIdApplicationCount: developerIdApplications.length,
    developerIdApplications,
  };
}

export function acceptanceReadiness({
  identityInventory,
  teamIdentifier,
  signingIdentity,
  renderedEntitlementsPresent,
  provisioningProfilePresent,
}) {
  const blockers = [];
  if (identityInventory.developerIdApplicationCount < 1) {
    blockers.push("authorized_developer_id_application_identity_missing");
  }
  if (!teamIdentifier) blockers.push("explicit_team_identifier_missing");
  else validateTeamIdentifier(teamIdentifier);
  if (!signingIdentity || signingIdentity === "-") {
    blockers.push("explicit_signing_identity_missing");
  } else {
    const selected = identityInventory.developerIdApplications.find(
      (entry) => entry.identity === signingIdentity,
    );
    if (!selected) {
      blockers.push("explicit_signing_identity_not_found");
    } else if (teamIdentifier && selected.teamIdentifier !== teamIdentifier) {
      blockers.push("team_identifier_does_not_match_signing_identity");
    }
  }
  if (!renderedEntitlementsPresent) {
    blockers.push("rendered_entitlements_missing");
  }
  if (!provisioningProfilePresent) {
    blockers.push("authorized_provisioning_profile_missing");
  }
  return {
    schemaVersion: 1,
    readyForSignedCandidate: blockers.length === 0,
    validCodeSigningIdentityCount: identityInventory.validIdentityCount,
    developerIdApplicationIdentityCount:
      identityInventory.developerIdApplicationCount,
    blockers,
  };
}

export function assertCandidateContinuity(candidateA, candidateB) {
  for (const field of ["identifier", "teamIdentifier"]) {
    if (
      !candidateA.app?.[field] ||
      candidateA.app[field] !== candidateB.app?.[field]
    ) {
      throw new Error(`Candidate app ${field} continuity failed`);
    }
    if (
      !candidateA.engine?.[field] ||
      candidateA.engine[field] !== candidateB.engine?.[field]
    ) {
      throw new Error(`Candidate engine ${field} continuity failed`);
    }
  }
  return {
    appIdentifier: candidateA.app.identifier,
    engineIdentifier: candidateA.engine.identifier,
    teamIdentifier: candidateA.app.teamIdentifier,
  };
}

function exactStringArray(value, expected) {
  return (
    Array.isArray(value) &&
    value.length === expected.length &&
    value.every((item, index) => item === expected[index])
  );
}

export function assertSignedEntitlements(entitlements, expectedTeam) {
  const team = validateTeamIdentifier(expectedTeam);
  const applicationIdentifier = `${team}.${APP_IDENTIFIER}`;
  if (
    entitlements?.["com.apple.application-identifier"] !==
      applicationIdentifier ||
    entitlements?.["com.apple.developer.team-identifier"] !== team ||
    !exactStringArray(entitlements?.["keychain-access-groups"], [
      applicationIdentifier,
    ])
  ) {
    throw new Error(
      "Signed app entitlements do not exactly match the reviewed team, application, and single access group",
    );
  }
  return { applicationIdentifier, accessGroup: applicationIdentifier };
}

export function assertProvisioningProfile(
  profile,
  expectedTeam,
  now = new Date(),
) {
  const team = validateTeamIdentifier(expectedTeam);
  const expected = `${team}.${APP_IDENTIFIER}`;
  if (
    !exactStringArray(profile?.TeamIdentifier, [team]) ||
    !exactStringArray(profile?.ApplicationIdentifierPrefix, [team])
  ) {
    throw new Error("Provisioning profile team/prefix does not exactly match");
  }
  assertSignedEntitlements(profile?.Entitlements, team);
  const expiration = new Date(profile?.ExpirationDate ?? "");
  if (!Number.isFinite(expiration.getTime()) || expiration <= now) {
    throw new Error("Provisioning profile is missing, malformed, or expired");
  }
  return {
    applicationIdentifier: expected,
    expiresAt: expiration.toISOString(),
  };
}

function run(command, args, input) {
  const result = spawnSync(command, args, { encoding: "utf8", input });
  const output = `${result.stdout ?? ""}${result.stderr ?? ""}`;
  if (result.status !== 0) {
    throw new Error(`${command} ${args[0] ?? ""} failed: ${output.trim()}`);
  }
  return output;
}

function parsePlist(xml, label) {
  let value;
  try {
    value = JSON.parse(
      run("plutil", ["-convert", "json", "-o", "-", "--", "-"], xml),
    );
  } catch (error) {
    throw new Error(`${label} is not a valid plist`, { cause: error });
  }
  if (!value || Array.isArray(value) || typeof value !== "object") {
    throw new Error(`${label} must be a plist dictionary`);
  }
  return value;
}

function inspectSignature(path) {
  return parseCodeSignature(run("codesign", ["-dvvv", path]));
}

export function verifySignedBundle(appPath, expectedTeam) {
  const team = validateTeamIdentifier(expectedTeam);
  const app = resolve(appPath);
  const host = resolve(app, "Contents/MacOS/chronosync-desktop");
  const engine = resolve(app, "Contents/Resources/binaries/chronosync-engine");
  const profile = resolve(app, "Contents/embedded.provisionprofile");
  for (const path of [app, host, engine, profile]) {
    if (!existsSync(path)) throw new Error(`Missing release artifact: ${path}`);
  }
  for (const path of [app, host, engine]) {
    run("codesign", ["--verify", "--strict", "--verbose=4", path]);
  }
  const appSignature = inspectSignature(app);
  const engineSignature = inspectSignature(engine);
  if (
    appSignature.identifier !== APP_IDENTIFIER ||
    appSignature.teamIdentifier !== team ||
    appSignature.signature === "adhoc" ||
    !appSignature.authorities.some((authority) =>
      authority.startsWith("Developer ID Application:"),
    ) ||
    !appSignature.flags.includes("runtime") ||
    !appSignature.timestamp
  ) {
    throw new Error(
      "Outer app does not have the expected stable hardened-runtime signature",
    );
  }
  if (
    engineSignature.identifier !== ENGINE_IDENTIFIER ||
    engineSignature.teamIdentifier !== team ||
    engineSignature.signature === "adhoc" ||
    !engineSignature.authorities.some((authority) =>
      authority.startsWith("Developer ID Application:"),
    ) ||
    !engineSignature.flags.includes("runtime") ||
    !engineSignature.timestamp
  ) {
    throw new Error(
      "Bundled engine does not have the expected signing identity",
    );
  }
  const entitlements = parsePlist(
    run("codesign", ["-d", "--entitlements", "-", app]),
    "signed app entitlements",
  );
  assertSignedEntitlements(entitlements, team);
  const provision = parsePlist(
    run("security", ["cms", "-D", "-i", profile]),
    "provisioning profile",
  );
  assertProvisioningProfile(provision, team);
  return { app: appSignature, engine: engineSignature };
}

export function verifyCandidateContinuity(
  candidateAPath,
  candidateBPath,
  expectedTeam,
) {
  return assertCandidateContinuity(
    verifySignedBundle(candidateAPath, expectedTeam),
    verifySignedBundle(candidateBPath, expectedTeam),
  );
}

export function inspectAcceptanceEnvironment() {
  const identityInventory = parseIdentityInventory(
    run("security", ["find-identity", "-v", "-p", "codesigning"]),
  );
  const readiness = acceptanceReadiness({
    identityInventory,
    teamIdentifier: process.env.APPLE_TEAM_ID,
    signingIdentity: process.env.APPLE_SIGNING_IDENTITY,
    renderedEntitlementsPresent: existsSync(
      resolve(ROOT, "apps/desktop/src-tauri/Entitlements.release.plist"),
    ),
    provisioningProfilePresent: existsSync(
      resolve(ROOT, "apps/desktop/src-tauri/embedded.provisionprofile"),
    ),
  });
  return {
    ...readiness,
    macOSVersion: run("sw_vers", ["-productVersion"]).trim(),
    architecture: run("uname", ["-m"]).trim(),
    sdkVersion: run("xcrun", ["--sdk", "macosx", "--show-sdk-version"]).trim(),
    deploymentTarget: checkReleaseStructure().minimumSystemVersion,
  };
}

function valueAfter(args, name) {
  const index = args.indexOf(name);
  return index >= 0 ? args[index + 1] : undefined;
}

export function main(args = process.argv.slice(2)) {
  const command = args[0];
  if (command === "check") {
    process.stdout.write(`${JSON.stringify(checkReleaseStructure())}\n`);
    return;
  }
  if (command === "preflight") {
    process.stdout.write(`${JSON.stringify(inspectAcceptanceEnvironment())}\n`);
    return;
  }
  if (command === "render") {
    const team = valueAfter(args, "--team-id");
    const output = valueAfter(args, "--output");
    if (!output) throw new Error("render requires --output");
    const resolvedOutput = resolve(output);
    if (resolvedOutput === TEMPLATE_PATH)
      throw new Error("refusing to overwrite entitlement template");
    const rendered = renderEntitlements(
      readFileSync(TEMPLATE_PATH, "utf8"),
      team,
    );
    writeFileSync(resolvedOutput, rendered, {
      encoding: "utf8",
      flag: "wx",
      mode: 0o600,
    });
    return;
  }
  if (command === "verify") {
    const app = valueAfter(args, "--app");
    const team = valueAfter(args, "--team-id");
    if (!app) throw new Error("verify requires --app");
    process.stdout.write(`${JSON.stringify(verifySignedBundle(app, team))}\n`);
    return;
  }
  if (command === "verify-continuity") {
    const candidateA = valueAfter(args, "--candidate-a");
    const candidateB = valueAfter(args, "--candidate-b");
    const team = valueAfter(args, "--team-id");
    if (!candidateA || !candidateB) {
      throw new Error(
        "verify-continuity requires --candidate-a and --candidate-b",
      );
    }
    process.stdout.write(
      `${JSON.stringify(verifyCandidateContinuity(candidateA, candidateB, team))}\n`,
    );
    return;
  }
  throw new Error(
    "usage: macos-release-identity.mjs check | preflight | render --team-id TEAMID --output FILE | verify --app APP --team-id TEAMID | verify-continuity --candidate-a APP --candidate-b APP --team-id TEAMID",
  );
}

if (
  process.argv[1] &&
  resolve(process.argv[1]) === fileURLToPath(import.meta.url)
) {
  try {
    main();
  } catch (error) {
    process.stderr.write(
      `${error instanceof Error ? error.message : String(error)}\n`,
    );
    process.exitCode = 1;
  }
}
