# Admission canonicalization fixtures

Synthetic public data only. binding-v1.hex contains the exact 3,975 canonical bytes for bindingFixture in binding_test.go; binding-v1.sha256 contains SHA-256 of those bytes. The fixed vector was independently encoded with Python struct big-endian length/integer operations, not emitted by the Go canonicalizer. The field order and encoding are specified in docs/SYSTEM_DESIGN.md.

The fixture names a synthetic provider and external identity/approval references. It is not valid production capability evidence and does not authorize a process. TOKEN names a variable; no token value, key, private prompt or real credential appears here. Digests and IDs are synthetic repetitions/counters, not measurements of host data.

Update these vectors only with an explicit canonical-contract review. A serializer refactor must preserve version-1 bytes.
