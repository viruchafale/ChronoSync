import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { URL } from "node:url";
import {
  APP_IDENTIFIER,
  ENGINE_IDENTIFIER,
  TEAM_TOKEN,
  acceptanceReadiness,
  assertCandidateContinuity,
  assertProvisioningProfile,
  assertSignedEntitlements,
  checkReleaseStructure,
  parseCodeSignature,
  parseIdentityInventory,
  renderEntitlements,
  validateTeamIdentifier,
} from "./macos-release-identity.mjs";

assert.throws(() => validateTeamIdentifier(""));
assert.throws(() => validateTeamIdentifier("invented-team"));
assert.equal(validateTeamIdentifier("A1B2C3D4E5"), "A1B2C3D4E5");

const template = `${TEAM_TOKEN}\n${TEAM_TOKEN}.${APP_IDENTIFIER}\n${TEAM_TOKEN}.${APP_IDENTIFIER}`;
const rendered = renderEntitlements(template, "A1B2C3D4E5");
assert.equal(rendered.includes(TEAM_TOKEN), false);
assert.equal(rendered.match(/A1B2C3D4E5/g)?.length, 3);

const parsed = parseCodeSignature(`Executable=/tmp/ChronoSync
Identifier=dev.chronosync.desktop
CodeDirectory v=20500 size=1 flags=0x10000(runtime) hashes=1+0 location=embedded
Signature size=9000
Timestamp=Nov 4, 2026 at 10:00:00 AM
Authority=Developer ID Application: ChronoSync Example (A1B2C3D4E5)
Authority=Developer ID Certification Authority
TeamIdentifier=A1B2C3D4E5
`);
assert.equal(parsed.identifier, APP_IDENTIFIER);
assert.equal(parsed.teamIdentifier, "A1B2C3D4E5");
assert.match(parsed.flags, /runtime/);
assert.match(parsed.timestamp, /Nov 4, 2026/);
assert.match(parsed.authorities[0], /^Developer ID Application:/);

assert.deepEqual(parseIdentityInventory("     0 valid identities found\n"), {
  validIdentityCount: 0,
  developerIdApplicationCount: 0,
  developerIdApplications: [],
});
const inventory =
  parseIdentityInventory(`  1) ABCDEF0123 "Developer ID Application: Example (A1B2C3D4E5)"
  2) 012345ABCD "Apple Development: Example (A1B2C3D4E5)"
     2 valid identities found
`);
assert.deepEqual(inventory, {
  validIdentityCount: 2,
  developerIdApplicationCount: 1,
  developerIdApplications: [
    {
      identity: "Developer ID Application: Example (A1B2C3D4E5)",
      teamIdentifier: "A1B2C3D4E5",
    },
  ],
});
assert.deepEqual(
  acceptanceReadiness({
    identityInventory: inventory,
    teamIdentifier: "A1B2C3D4E5",
    signingIdentity: "Developer ID Application: Example (A1B2C3D4E5)",
    renderedEntitlementsPresent: true,
    provisioningProfilePresent: true,
  }),
  {
    schemaVersion: 1,
    readyForSignedCandidate: true,
    validCodeSigningIdentityCount: 2,
    developerIdApplicationIdentityCount: 1,
    blockers: [],
  },
);
const blocked = acceptanceReadiness({
  identityInventory: parseIdentityInventory("0 valid identities found"),
  renderedEntitlementsPresent: false,
  provisioningProfilePresent: false,
});
assert.equal(blocked.readyForSignedCandidate, false);
assert.deepEqual(blocked.blockers, [
  "authorized_developer_id_application_identity_missing",
  "explicit_team_identifier_missing",
  "explicit_signing_identity_missing",
  "rendered_entitlements_missing",
  "authorized_provisioning_profile_missing",
]);
const wrongSelection = acceptanceReadiness({
  identityInventory: inventory,
  teamIdentifier: "A1B2C3D4E5",
  signingIdentity: "Developer ID Application: Other (Z9Y8X7W6V5)",
  renderedEntitlementsPresent: true,
  provisioningProfilePresent: true,
});
assert.deepEqual(wrongSelection.blockers, [
  "explicit_signing_identity_not_found",
]);
const wrongTeam = acceptanceReadiness({
  identityInventory: inventory,
  teamIdentifier: "Z9Y8X7W6V5",
  signingIdentity: "Developer ID Application: Example (A1B2C3D4E5)",
  renderedEntitlementsPresent: true,
  provisioningProfilePresent: true,
});
assert.deepEqual(wrongTeam.blockers, [
  "team_identifier_does_not_match_signing_identity",
]);

const candidate = {
  app: { identifier: APP_IDENTIFIER, teamIdentifier: "A1B2C3D4E5" },
  engine: { identifier: ENGINE_IDENTIFIER, teamIdentifier: "A1B2C3D4E5" },
};
assert.deepEqual(
  assertCandidateContinuity(candidate, {
    app: { ...candidate.app },
    engine: { ...candidate.engine },
  }),
  {
    appIdentifier: APP_IDENTIFIER,
    engineIdentifier: ENGINE_IDENTIFIER,
    teamIdentifier: "A1B2C3D4E5",
  },
);
assert.throws(() =>
  assertCandidateContinuity(candidate, {
    ...candidate,
    app: { ...candidate.app, teamIdentifier: "Z9Y8X7W6V5" },
  }),
);

const reviewedEntitlements = {
  "com.apple.application-identifier": "A1B2C3D4E5.dev.chronosync.desktop",
  "com.apple.developer.team-identifier": "A1B2C3D4E5",
  "keychain-access-groups": ["A1B2C3D4E5.dev.chronosync.desktop"],
};
assert.deepEqual(assertSignedEntitlements(reviewedEntitlements, "A1B2C3D4E5"), {
  applicationIdentifier: "A1B2C3D4E5.dev.chronosync.desktop",
  accessGroup: "A1B2C3D4E5.dev.chronosync.desktop",
});
assert.throws(() =>
  assertSignedEntitlements(
    {
      ...reviewedEntitlements,
      "keychain-access-groups": [
        ...reviewedEntitlements["keychain-access-groups"],
        "A1B2C3D4E5.unreviewed.group",
      ],
    },
    "A1B2C3D4E5",
  ),
);
const reviewedProfile = {
  TeamIdentifier: ["A1B2C3D4E5"],
  ApplicationIdentifierPrefix: ["A1B2C3D4E5"],
  Entitlements: reviewedEntitlements,
  ExpirationDate: "2030-01-01T00:00:00.000Z",
};
assert.deepEqual(
  assertProvisioningProfile(
    reviewedProfile,
    "A1B2C3D4E5",
    new Date("2029-01-01T00:00:00.000Z"),
  ),
  {
    applicationIdentifier: "A1B2C3D4E5.dev.chronosync.desktop",
    expiresAt: "2030-01-01T00:00:00.000Z",
  },
);
assert.throws(() =>
  assertProvisioningProfile(
    reviewedProfile,
    "A1B2C3D4E5",
    new Date("2031-01-01T00:00:00.000Z"),
  ),
);

assert.deepEqual(checkReleaseStructure(), {
  appIdentifier: APP_IDENTIFIER,
  minimumSystemVersion: "10.15",
});
const refusedSidecar = spawnSync(process.execPath, ["scripts/sidecar.mjs"], {
  cwd: new URL("..", import.meta.url),
  encoding: "utf8",
  env: {
    ...process.env,
    CHRONOSYNC_RELEASE_SIGN_SIDECAR: "1",
    APPLE_SIGNING_IDENTITY: "-",
  },
});
assert.notEqual(refusedSidecar.status, 0);
assert.match(
  `${refusedSidecar.stdout}${refusedSidecar.stderr}`,
  /non-ad-hoc APPLE_SIGNING_IDENTITY/,
);
process.stdout.write(
  "PASS: release identity selection, entitlements/profile, refusal, rendering, signature parsing and continuity\n",
);
