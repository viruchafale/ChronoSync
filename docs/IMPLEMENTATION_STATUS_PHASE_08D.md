# Phase 8D implementation status — portable exchange format

Implemented deterministic store-only ZIP packages with an 8 MiB/100-entry bound, canonical manifest, signer-independent content ID, exact entry digests/lengths, one benchmark, optional canonical reports, explicit included/omitted provenance, privacy declarations and a signed identity envelope. Packages are parsed in memory and never extracted.

Readers refuse compression, encryption, traversal, absolute/backslash/non-normalized paths, duplicate/unknown entries, directories, symlinks/special files, excessive counts/sizes, malformed/noncanonical/duplicate-key/unknown-field JSON and mismatched content/provenance. See the [format specification](BENCHMARK_EXCHANGE_FORMAT.md).
