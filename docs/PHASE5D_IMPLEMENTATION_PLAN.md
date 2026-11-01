# Phase 5D implementation plan — unified replay evaluation

Phase 5D combines validated Phase 5A regression bytes with validated Phase 5B synthetic transport bytes through a new internal `replay/unified` coordinator. The caller supplies inert actual output fingerprints/exit codes and structured synthetic requests; no executable callback, provider, network or historical command crosses the boundary.

Each evaluation constructs a fresh transport matcher. Requests are processed in plan order with explicit occurrences; every recorded transport entry must be represented. Unmatched, ambiguous, invalid, unsupported or cancelled outcomes stop success. Phase 5A evaluation then compares caller-supplied actual fingerprints against recorded expectations and retains its detailed mismatch locations. Only all matched transport requests plus a completed passing Phase 5A result produce a passing unified result.

Plans and results are versioned, bounded and stable JSON structs. A caller context and bounded per-evaluation timeout apply to validation, matching and comparison. Result serialization is canonical for unchanged inputs. Tests cover pass, output/exit mismatches, repeated occurrence matching, ambiguity, unmatched/unsupported requests, malformed fixtures, cancellation, timeout validation, fresh-state repeatability and stable bytes.
