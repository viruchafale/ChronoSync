# Environment audit — 2026-10-21

Initial root `/Users/viru/Desktop/ChronoSync` was empty. `git status --short --branch` reported not a Git repository. There was no existing work, convention, lockfile, or CI to preserve. No commits/pushes are authorized or performed; Git initialization is not needed to inspect the foundation.

| Tool            | Observed                                 |
| --------------- | ---------------------------------------- |
| OS              | macOS 26.6.2, arm64                      |
| Go              | 1.26.2                                   |
| Node            | 22.14.0                                  |
| npm             | 10.9.2                                   |
| Git             | 2.47.0                                   |
| Codex           | 0.163.0; login status reports ChatGPT    |
| Apple clang     | 21.0.0                                   |
| Developer tools | /Library/Developer/CommandLineTools      |
| Rust/Cargo      | not on PATH and absent from ~/.cargo/bin |

Go has built-in gofmt, vet, testing and race support. Frontend formatting/testing/linting are repository dependencies. Native Rust tooling is missing; WebView/runtime testing is not done. No remote CI run exists.

Compatibility sources inspected: [Tauri prerequisites](https://v2.tauri.app/start/prerequisites/), [Vite guide](https://vite.dev/guide/), and official Codex pages linked in CODEX_INTEGRATION.md. Dependency package engines and the lockfile govern the selected Vite 7 line, not assumptions about the latest major. Rust installation is a system prerequisite, not silently performed by bootstrap.

Initial npm registry queries and installation failed with sandbox DNS ENOTFOUND; an escalated installation was requested. Final results are recorded in IMPLEMENTATION_STATUS.md. The initial generic tool probe used `go --version`, which Go rejects; `go version` subsequently established the actual version. Codex help/login probes warned they could not create PATH aliases; their requested read-only output still succeeded.
