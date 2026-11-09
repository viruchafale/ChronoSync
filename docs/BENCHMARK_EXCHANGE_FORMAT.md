# Benchmark exchange format v1

ChronoSync exchange files use extension `.chronosync-exchange` and are deterministic ZIP containers. Readers must treat every byte as hostile and must never extract entries to the filesystem.

## Limits and ZIP profile

- Maximum package: 8 MiB.
- Maximum entries: 100.
- `manifest.json`: at most 128 KiB.
- `signature.json`: at most 16 KiB.
- `benchmark.benchmark`: at most the Phase 7 2 MiB dataset limit.
- Each `reports/<report-id>.json`: at most the Phase 6 3 MiB report limit.
- Only ZIP method `Store` is permitted. Compressed/encrypted entries are refused.
- Every entry must be a regular `0600` file. Directories, symlinks and special modes are refused.
- Paths must be exact normalized forward-slash names. Absolute paths, backslashes, traversal, duplicate names and unknown entries are refused.

Allowed paths are exactly:

```text
benchmark.benchmark
manifest.json
reports/<32-lowercase-hex-report-id>.json
signature.json
```

Writers sort paths and use the fixed UTC ZIP timestamp 1980-01-01. Readers do not depend on archive order or timestamps; the canonical manifest defines content identity.

## Canonical manifest

`manifest.json` is canonical UTF-8 JSON with no unknown or duplicate fields. It declares format/domain, benchmark content/family/version/compatibility, sorted content entries, sorted included and omitted report identities, and a privacy manifest. Each content entry records exact path, closed role, SHA-256 and byte length. Manifest entries cover the benchmark and reports; they do not include the manifest or signature envelope.

`package_id = SHA-256("chronosync-exchange-package-v1\\0" || canonical_manifest_bytes)`.

The content ID is intentionally independent of signer identity. Re-signing identical content and inclusion choices produces the same content ID, while the archive signature differs by identity.

## Signature envelope

`signature.json` is canonical JSON containing format 1, algorithm `ed25519`, the complete public identity record and a base64 signature. The signature payload is:

```text
"chronosync-exchange-signature-v1\0"
|| canonical_public_identity_bytes
|| 0x00
|| canonical_manifest_bytes
```

Entry integrity is transitively covered because the signed manifest contains every entry digest and length. The public identity is covered so its self-asserted label, creation time and rotation link cannot be changed without invalidating the signature.

## Provenance and privacy

Every benchmark source report must be either included or explicitly omitted. Included reports must be canonical Phase 6 report v1 bytes and must exactly satisfy each Phase 7 case's report ID, comparison index, observation IDs, comparison digest and expected status. Any omission or mismatch yields `incomplete_provenance`; such a package may be inspected and cryptographically verified but cannot import.

The privacy manifest must state false for raw commands, raw outputs, workspace paths, credentials, environment variables and private keys. These values are structurally absent from the Phase 6/7 canonical formats, but report IDs, execution/event IDs, configuration labels and equality-revealing fingerprints remain potentially sensitive. Export remains an explicit native-confirmed action.

## Verification outcomes

`valid_trusted`, `valid_untrusted`, `invalid_signature`, `revoked_identity`, `unsupported_algorithm`, `unsupported_format`, `malformed_package` and `incomplete_provenance` are distinct. A valid signature proves only that the embedded key signed the covered payload. Trust is a separate local policy, and neither validity nor trust establishes safety, quality, authorship in the civil-identity sense, deterministic AI behavior or execution authorization.
