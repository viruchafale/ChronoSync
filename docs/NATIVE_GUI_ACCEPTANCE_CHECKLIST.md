# Native GUI acceptance checklist — Phases 1–11

## Current audit result

`MANUAL_ACCEPTANCE_PENDING`

On 2026-11-05 the freshly validated unbundled current executable launched, remained running, and registered frontmost with LaunchServices; both registered paths pointed directly to the unbundled binary. It still did not appear as an accessible application surface in the available GUI automation inventory and was stopped normally. Building a new `.app`/DMG solely for acceptance is prohibited by the protected-artifact constraint. No workflow was clicked and no live provider/agent was invoked; launch smoke is not GUI acceptance.

The limitation was reproduced on 2026-11-08. The existing no-bundle arm64 Mach-O launched, but the automation inventory still omitted ChronoSync. Binding by its full executable path and by `dev.chronosync.desktop` both returned `Invalid app`. Chrome browser automation was independently restored and does not resolve this native-app discovery boundary. Native confirmation dialogs remain unverified.

A later, separately authorized 2026-11-08 task built an isolated app-only candidate without touching the protected DMG. The bundle is discoverable and launch-smoked but only ad-hoc/linker-signed; human dialog and Keychain workflows remain unverified. Use the candidate-specific [manual checklist](DISPOSABLE_NATIVE_GUI_ACCEPTANCE_CHECKLIST.md) and [identity report](DISPOSABLE_NATIVE_GUI_ACCEPTANCE_REPORT.md). The earlier no-bundle limitation remains accurate for that executable.

## Practical manual closure procedure

1. Hash the protected DMG and require `e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`; leave it unmounted and unchanged.
2. Use the already validated current no-bundle executable, or a separately authorized disposable current `.app` candidate. Record its checksum, signature class, data directory and operator. Do not treat the historical protected DMG as current-source UI evidence.
3. Point the run at disposable private application data and a secret-free synthetic workspace. Do not use a production identity or live provider.
4. Resize the content area to 1440×900, 1280×800 and 1024×768. At each size inspect Security for clipping, horizontal scrolling, readable metadata, hierarchy, spacing, panel bounds, selected navigation and status text. Save sanitized full-window screenshots.
5. With a disposable fixture, choose Export, inspect the native `Export regression fixture` confirmation, test keyboard focus and Cancel, and verify no file was written. Open Import, inspect the native file picker, then cancel it. These cancellation paths are non-destructive.
6. If separately authorized, repeat representative cancellation checks for governed-evidence and exchange confirmations. Do not approve deletion, trust mutation, migration or export merely to obtain a screenshot.
7. Run disposable Keychain/user-presence ceremonies only under the existing Phase 9–11 authorizations. Record success, cancellation and denial separately; never capture secrets or authentication material.
8. Record PASS/FAIL/BLOCKED per dialog and viewport. A browser component test, launch smoke or no-bundle build is not a substitute for this record.

## Safe test prerequisites

- Use a disposable private application-data directory and synthetic workspace containing no secrets.
- Use a reviewed inert synthetic recorder executable for automated recorder paths; a real Codex run requires separate explicit authorization and is outside this audit.
- Do not reuse production identities, trust stores, login-Keychain items or external package destinations.
- Keep the submitted DMG unmounted or read-only and never replace it.
- Record app/engine versions, platform, candidate signature state and exact data directory before testing.

## Phase 1 — recorder

- Start with explicit consent and verify created → starting → running → one terminal state.
- Observe normalized command/message/unknown-event rows, output truncation/redaction labels and workspace-change evidence.
- Cancel while running; verify acknowledged cancellation is distinct from terminal confirmation and the owned process is gone.
- Close the native host during a synthetic run; reopen and verify interruption/cleanup uncertainty is explicit and no automatic resume occurs.
- Exercise malformed executable, denied workspace, timeout and storage failure presentations without exposing raw sensitive values.

## Phase 2 — durable history

- Restart and verify run ordering, pagination, filters, event detail and warnings match the engine state.
- Corrupt only a disposable copied payload; verify history is retained/damaged, detail refuses, unrelated runs remain readable and no automatic repair occurs.
- Verify empty/loading/error/end-of-page states and keyboard navigation.

## Phase 3 — snapshots and branches

- Capture with explicit exclusions; verify pending/committed state and displayed limitations.
- Validate, restore and branch to application-owned fresh destinations; verify original workspace/history remain unchanged.
- Edit the disposable branch, recapture, and confirm ancestry plus deduplication/inventory results.
- Exercise changing-source, damaged-object, busy-recorder and interrupted materialization failures; no failed branch may appear ready.

## Phases 4–7 — refusal, replay, analysis and governance

- Confirm no UI control claims production policy authorization or sandbox enforcement.
- Create/import/inspect/evaluate inert fixtures through native dialogs; verify no command/provider execution and explicit mismatch/unsupported states.
- Compare synthetic histories; verify earliest divergence is labeled observational, causality unknown, and missing evidence cannot pass.
- Create/compare/evaluate benchmarks; verify immutable revisions/lineage, expiry preview-only behavior, reference-blocked deletion and explicit confirmation.

## Phases 8–9 — exchange, trust and login-Keychain custody

- Use disposable identities/packages only. Verify valid-untrusted confirmation, trusted import, revoked refusal, tampered refusal and restart persistence.
- Confirm validity and trust are distinct and hashes are not described as authenticity.
- Exercise disposable login-Keychain create/status/sign/delete and legacy migration only with explicit confirmation; verify no seed appears in UI/logs and no sole usable key is silently deleted.
- Windows/Linux must show unsupported custody rather than simulated success.

## Phases 10–11 — signed protected acceptance

Run only from a strictly verified authorized disposable candidate. Confirm actual user-presence success, cancellation and denial; no login-item fallback; same-key migration/restart/idempotency; Candidate A/B access continuity; and separate notarization/stapling/Gatekeeper results. Until an authorized identity/profile exists, every item in this section remains `EXTERNAL_BLOCKED`.

## Acceptance record

For each item record date, operator, app/engine version, candidate checksum/signature class, disposable data/workspace identifiers, expected/actual state, screenshots with no secrets, and PASS/FAIL/BLOCKED. Do not collapse launch smoke, component tests or native builds into manual workflow acceptance.
