# Launch-candidate profile isolation

The integrated launch candidate supports KENNEL_ELECTRON_DATA_DIR as an absolute Electron userData directory override. Without it, packaged and development defaults are unchanged. A relative override fails at startup rather than creating state in an arbitrary working directory.

For disposable packaged acceptance, set KENNEL_ELECTRON_DATA_DIR, KENNEL_DATA_DIR and KENNEL_RUN_FILE to separate paths beneath the audit directory, and choose a free KENNEL_PORT. Do not override HOME or reuse the production profile. This isolates Chromium state as well as daemon data. It does not bypass application permissions or provider authentication.
