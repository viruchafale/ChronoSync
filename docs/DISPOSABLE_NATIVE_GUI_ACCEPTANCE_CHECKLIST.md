# Disposable native GUI acceptance checklist

## Candidate and safety prerequisites

- Candidate: `/private/tmp/chronosync-native-gui-acceptance-20261108/release/bundle/macos/ChronoSync.app`
- Expected host SHA-256: `49075b0a8c28ec6824bc9cd9e98d24137406d67b74af401045635f143df1380b`
- Expected bundle identifier/version: `dev.chronosync.desktop` / `0.0.0`
- Classification: local ad-hoc development bundle; not production-signed or Gatekeeper-accepted.
- Perform the session in a dedicated disposable macOS standard-user account with a new login Keychain and no personal data. The stable identifier uses the normal application-data namespace.
- Use only a secret-free synthetic workspace and reviewed inert fixture/package files. Do not invoke a live provider or real Codex executable.
- Keep the submitted DMG unmounted and unchanged. Before and after testing require SHA-256 `e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`.
- Record operator, timestamp, macOS version, candidate hash, account name, expected result, actual result and PASS/FAIL/BLOCKED for every item. Screenshots must exclude usernames, home paths, secrets, seed material and unrelated files.
- Do not approve deletion, trust mutation, key migration or credential-sensitive actions without separate explicit authorization. Cancellation testing is sufficient where specified.

## A. Launch and initial window

**Steps**

1. Verify the candidate executable hash and protected DMG hash.
2. From the disposable account, open `ChronoSync.app` locally. Do not bypass a Gatekeeper security interstitial; this same-host local build should open directly.
3. Wait for the bottom status to change from `Connecting to engine…` to `Engine responded`.
4. Inspect the 1280×800 initial window, sidebar, Overview empty state, title, traffic-light controls and status bar.

**Expected:** one ChronoSync window opens, the bundled engine responds through private stdio, no terminal window or network listener appears, and no existing non-disposable records are shown.

**Pass/fail:** PASS only if launch and engine connection succeed without a security bypass and the current workbench is usable. FAIL on crash, blank window, persistent engine error, unexpected data or an unsafe bypass requirement.

**Evidence:** full-window screenshot after `Engine responded`, plus candidate hash text captured separately.

**Cleanup:** none; leave the app open for the next test.

## B. Native export confirmation presentation

**Precondition:** a reviewed inert disposable regression fixture exists in the disposable account. If none exists, record BLOCKED; do not use a real run merely to create one.

**Steps**

1. Open Replay & regression and refresh fixtures.
2. Select Export for the inert fixture.
3. Stop at the native `Export regression fixture` confirmation. Do not approve yet.
4. Inspect title, warning text, button labels, default focus, window attachment, clipping, contrast and VoiceOver names.

**Expected:** a native modal confirmation appears above ChronoSync, clearly identifies export/review-before-sharing intent, keeps the underlying action blocked and exposes affirmative and Cancel controls.

**Pass/fail:** PASS if the dialog is native, readable, correctly scoped and keyboard accessible. FAIL if missing, clipped, ambiguous, background-interactive or incorrectly focused.

**Evidence:** sanitized full-window screenshot showing the confirmation; do not include private fixture contents.

**Cleanup:** continue directly to test C without approving export.

## C. Export confirmation cancellation

**Steps**

1. With the export confirmation still open, press Escape. If Escape is not supported by the platform dialog, activate Cancel with the keyboard.
2. Verify focus returns to the originating Export control or logical fixture row.
3. Confirm no save picker opens, no export-success message appears and no file is created in the intended disposable destination.
4. Repeat once using pointer-based Cancel to compare behavior.

**Expected:** cancellation returns `cancelled` without exporting bytes or opening a save picker.

**Pass/fail:** PASS only if both cancellation paths cause no write and focus returns predictably. FAIL on export, save picker continuation, lost focus or stale busy state.

**Evidence:** screenshot after cancellation with the fixture still present and no success message; optional Finder screenshot of the empty disposable export directory.

**Cleanup:** remove the empty disposable export directory only with operator approval, or leave it for final disposable-account removal.

## D. Native import file picker presentation

**Steps**

1. Open Replay & regression.
2. Activate `Import fixture…`.
3. Inspect the native Open panel without selecting a file.
4. Verify the ChronoSync fixture filter, filename area, Cancel/Open controls, initial focus, keyboard traversal and that unrelated hidden files are not unexpectedly exposed.

**Expected:** a native file picker appears with a fixture/JSON-oriented filter and a disabled or safe Open action until a valid selection is made.

**Pass/fail:** PASS if the panel is native, correctly filtered, readable and keyboard accessible. FAIL on missing filter, wrong modality, clipped controls or unexpected automatic selection/import.

**Evidence:** sanitized picker screenshot showing only the dedicated secret-free test directory.

**Cleanup:** continue to test E; do not select a file.

## E. File-picker cancellation

**Steps**

1. Press Escape in the open picker.
2. Reopen it and activate Cancel with the keyboard or pointer.
3. Verify the app returns to Replay & regression, shows no imported/success state, adds no fixture and restores usable focus.

**Expected:** both cancellation methods are side-effect free.

**Pass/fail:** PASS if no fixture or file operation occurs and focus returns. FAIL on import, error loop, frozen modal state or lost keyboard focus.

**Evidence:** screenshot of Replay after cancellation and the unchanged fixture count.

**Cleanup:** none.

## F. Keyboard focus and accessibility

**Steps**

1. Navigate the sidebar using Tab and Shift-Tab; activate Overview, Replay and Security using Space/Return.
2. Open the command palette with Command-K or the displayed shortcut, type a filter, move through results, dismiss with Escape and verify focus restoration.
3. Traverse a native confirmation and file picker using Tab, Shift-Tab, Escape and the default action without approving a write.
4. With VoiceOver enabled by the operator, verify names for navigation, dialog buttons, inputs, status/error messages and the selected destination. Do not change global accessibility settings solely for the test.

**Expected:** focus is always visible, ordering is logical, Escape dismisses transient UI, focus returns to the invoker, and controls have meaningful names/states.

**Pass/fail:** PASS if every tested flow is keyboard-completable with no focus trap or silent status. FAIL on invisible focus, unreachable controls, unexpected default activation or missing accessible names.

**Evidence:** screenshots of visible focus on sidebar, command palette and one native Cancel control; record VoiceOver observations in text.

**Cleanup:** disable VoiceOver only if the operator enabled it for this session.

## G. Keychain and user-presence ceremonies

This section requires separate explicit operator authorization immediately before any Keychain mutation. The candidate is ad-hoc and cannot positively verify protected per-sign user presence.

**Login-Keychain steps, only if authorized**

1. Confirm the account is disposable and its Keychain contains no personal items.
2. In Exchange & trust, enter a unique label such as `ChronoSync GUI acceptance <timestamp>`.
3. First open `Create signing identity`, inspect the native warning, and cancel. Verify no identity appears.
4. If approval to create disposable Keychain data is explicit, repeat and approve. Observe any macOS Keychain prompt without exposing credentials in screenshots.
5. Refresh identities, run `Check Keychain access`, and record available/denied/cancelled behavior separately.
6. Do not migrate a legacy key, revoke trust or sign/export a package unless separately authorized.

**Expected:** cancellation creates nothing. An authorized creation stores one random Ed25519 seed as a generic-password item under service `local.chronosync.signing.v1`; the UI must continue to state that it is not Secure Enclave, hardware-backed or per-sign user-presence protected.

**Protected user-presence expected result:** BLOCKED for positive success. The unsigned/ad-hoc candidate must not claim protected user presence. Any fail-closed unavailable/refusal result is evidence only of negative behavior, not positive acceptance.

**Pass/fail:** PASS the cancellation path only if no item/identity is created. PASS an authorized generic-Keychain path only if the UI record and exact Keychain item agree and no seed is displayed. FAIL on silent creation, misleading protection claims, secret exposure or fallback described as protected presence.

**Evidence:** sanitized confirmation and post-operation status screenshots. Never capture passwords, seed bytes, authentication material or the contents of Keychain items.

**Cleanup:** record the created key ID/account. Remove only that exact `local.chronosync.signing.v1` generic-password item through Keychain Access after explicit cleanup approval, then remove the disposable macOS account when authorized. Never issue a broad Keychain deletion command. If cleanup is not authorized, record it as pending and stop.

## H. Window close and reopen

**Steps**

1. With no operation active, close the main window using the red window control and record whether the app remains active by design.
2. Use the app menu or Dock icon to reopen the main window if available.
3. Quit with Command-Q, confirm the app exits, and reopen `ChronoSync.app`.
4. Verify the engine reconnects and only disposable state persists. Confirm no operation resumes automatically.

**Expected:** close/reopen and full quit/relaunch are predictable; the app never duplicates the engine unexpectedly, loses the window permanently or resumes an interrupted operation without disclosure.

**Pass/fail:** PASS if window lifecycle and persisted disposable state are coherent. FAIL on crash, duplicate windows/processes, stale modal overlay, lost state that should persist or unsafe automatic resume.

**Evidence:** one screenshot after window reopen and one after full relaunch with `Engine responded`.

**Cleanup:** quit the app before error/cleanup work.

## I. Error states and recovery

**Steps**

1. Reopen the app and select a small reviewed invalid UTF-8 or malformed fixture from the disposable test directory. Do not use an oversized, secret-containing or adversarial file outside the documented bounds.
2. Verify the import is rejected with a bounded, understandable error and no fixture is stored.
3. Dismiss the error, refresh fixtures and navigate to another destination.
4. Cancel a second import and export confirmation to confirm recovery after the error.
5. Do not alter the app bundle, sidecar, database or Keychain to manufacture failures.

**Expected:** invalid input is rejected, the app remains responsive, existing disposable evidence is unchanged, and subsequent cancellation/navigation works.

**Pass/fail:** PASS if the error is clear, bounded and recoverable with no partial artifact. FAIL on crash, raw secret/path leakage, partial import, permanent busy state or required restart without explanation.

**Evidence:** sanitized error screenshot and a following healthy destination/status screenshot.

**Cleanup:** remove only the dedicated malformed test file and disposable test directory after explicit approval. At session end, quit ChronoSync, verify the protected DMG hash again, and either retain the disposable account for review or remove it through the approved macOS account-cleanup process.

## Completion rule

Native acceptance closes only when the operator records actual results and evidence for A–I. Build success, launch smoke, browser screenshots and component tests are supporting evidence only. Positive protected user-presence, Developer ID, notarization and Gatekeeper acceptance remain separate external gates.
