# Implementation status index

Keep each phase or separately scoped implementation milestone in its own Markdown status file. This file is an index and brief current overview only; do not append detailed implementation reports or phase archives here.

| Phase                                     | Status                                                                                 | Detailed evidence                                                                        |
| ----------------------------------------- | -------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| 0 — Foundation                            | Historical foundation evidence; original acceptance limitations retained in plan/audit | [Phase 0 plan](phases/PHASE-00-FOUNDATION.md), [environment audit](ENVIRONMENT_AUDIT.md) |
| 1 — Codex recorder                        | Implemented; native GUI acceptance unverified/blocked                                  | [Phase 1 status](IMPLEMENTATION_STATUS_PHASE_01.md)                                      |
| 2 — Durable event timeline                | Implemented; native GUI acceptance unverified/blocked                                  | [Phase 2 status](IMPLEMENTATION_STATUS_PHASE_02.md)                                      |
| 3 — Snapshots and branching               | Implemented; native GUI acceptance unverified/blocked                                  | [Phase 3 status](IMPLEMENTATION_STATUS_PHASE_03.md)                                      |
| 4 — Architecture and policy design        | Design complete; runtime implementation not started                                    | [Phase 4 status](IMPLEMENTATION_STATUS_PHASE_04.md)                                      |
| 4A — Execution-policy core                | Core/refusal verified; no enforcement backend                                          | [Phase 4A status](IMPLEMENTATION_STATUS_PHASE_04A.md)                                    |
| 4B — Durable admission and launch binding | Design complete; implementation not started                                            | [Phase 4B status](IMPLEMENTATION_STATUS_PHASE_04B.md)                                    |
| 4C — Admission domain contracts           | Contracts/tests complete; no persistence/launch                                        | [Phase 4C status](IMPLEMENTATION_STATUS_PHASE_04C.md)                                    |
| 4D — Durable admission repository         | Complete; portable/native checks passed; no launch integration                         | [Phase 4D status](IMPLEMENTATION_STATUS_PHASE_04D.md)                                    |
| 5–7                                       | Not started                                                                            | [Phase plans](README.md#phase-plans)                                                     |

Phase 4A is explicitly authorized separately from the completed Phase 4 design. It implements policy resolution and conservative refusal only; no sandbox backend or positive policy-controlled launch is enabled. Phase 1–3 GUI acceptance remains open. Phase 4B documents the next admission/binding protocol without implementing it. Phase 4C separately authorizes pure admission contracts/tests; Phase 4D separately authorizes persistence/recovery only; positive launch still requires separately scoped approval.

## Status-file convention

Use `IMPLEMENTATION_STATUS_PHASE_NN.md` for each phase. If a phase has separately scoped implementation milestones, create a distinct file with a descriptive suffix, rather than merging its evidence into a prior milestone's report. Preserve recorded results and unresolved gates, and link every new report here. Phase 4A runtime work has its own status file, separate from the completed design report; later milestones must follow the same convention.
