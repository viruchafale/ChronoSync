# Phase 6E implementation status — policies and report history

Policy format v1 implements explicit exact/partial event thresholds, dimension agreement, maximum missing/additional/reordered/changed counts, required evidence dimensions and `fail`/`incomparable` unsupported handling. Every rule records actual and required values. Required missing evidence cannot pass.

Canonical reports are stored append-only in private application data with exclusive creation, sync, bounds, strict decoding and SHA-256 integrity detection. Reevaluation writes a new report while retaining the original observations and earlier report. Restart, revised-policy and corruption tests pass. No SQLite migration or execution-history rewrite was introduced.
