# Phase 7B implementation status — benchmark dataset contracts and provenance

Implemented canonical benchmark-envelope v1 with stable family identity, positive version, content SHA-256, creation time, stable cases, resolved report/comparison/observation provenance, supported/required dimensions and explicit limitations. Strict import validates canonical JSON, bounds, identity/version uniqueness, references and comparison fingerprints. Content-addressed private publication is synced and no-replace; identical imports converge.

SHA-256 is documented as integrity/equality evidence only, not authenticity or authorship. Unknown formats, malformed/corrupt bytes, changed provenance and missing references are refused. No Phase 5 fixture or Phase 6 report format changed.
