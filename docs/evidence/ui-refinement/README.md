# UI refinement screenshot evidence

These PNGs were captured from the local `tests/visual` Vite harness with Chrome's native device preview. The harness renders actual ChronoSync components with bounded inert synthetic records and rejects native/write operations. The images are frontend layout evidence only.

Directories named `1440x900`, `1280x800` and `1024x768` contain target-viewport captures. `1440x542` contains early diagnostics and is excluded from acceptance. Filenames containing `before` record a defect before its fix; corresponding `after` files record the corrected layout. The 1280 governance `before-table-fix` image is retained as defect evidence; corrected governance wrapping is shown at 1440 and 1024.

Chrome produced two invalid earlier 1024 captures with incorrect zoom/cropping. They are intentionally absent. Exchange was visually inspected at 1024 but its capture failed. On 2026-11-08 the dedicated Chrome window was rebound, Security was re-inspected in the exact 1024×768 preview, and a genuine 2048×1536 Retina page capture was added as `1024x768/security.png`. The retained set now contains 35 target-viewport captures and two diagnostic-only captures. See `docs/GUI_ACCEPTANCE_CLOSURE_REPORT.md` for the final classification and interaction record.
