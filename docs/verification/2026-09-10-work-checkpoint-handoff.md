# Work integration checkpoint — 10 September 2026

This draft PR preserves completed work for review. It is not launch acceptance.

## Included source

- Base: beta `670a42238795b3da0c9fd61a5434132e9b622dd0`.
- Mission UI checkpoint `dcc9cab29`, canonical run/check UI integration `ac3e08fcc`.
- Backend takeover through `56f986da4`, including artifact continuity, approved checks, run intent, documents and reviewed admission corrections.
- Model Plan schema and criterion alias/check command corrections `a16cc3f14`.
- Durable owner-triggered delivery `f8b02e887`, now integrated; independent delivery review and native delivery acceptance remain open.
- Root `.env` exclusion; credentials and local runtime state are not included.

## Verification boundaries

Focused macOS arm64 package and package identity passed on `cae5795e9`, before delivery integration. Native development Electron and its isolated daemon started; health/readiness returned success. This does not prove the combined delivery package or an end-to-end journey.

The backend delivery task reported full Go tests/build/vet and affected race tests passing. Its lint run reported 15 findings outside delivery files; the combined PR does not claim clean lint. Earlier frontend/typecheck and backend evidence remain checkpoint-specific. See the final PR description for the combined regression result.

## First tasks tomorrow

1. **Project import failure:** owner screenshot shows Import project -> Choose a project folder -> `Invalid JSON body (INVALID_JSON)`. Audit daemon logs contain two POST /api/v1/projects HTTP400 responses. Exact request and root cause are not yet captured. Reproduce at the native folder-picker -> request DTO -> controller boundary; do not guess or hide the error.
2. **Development connection state:** native Work shows daemon unavailable while isolated daemon reports ready and serves reads. Notification proxy targeted default port3001 instead of the test port. Retry with KENNEL_DEV_API_TARGET matching KENNEL_PORT and trace any remaining readiness mismatch.
3. **Native testing infrastructure:** exact Electron path returns screenshot/AX, but pointer actions return noWindowsAvailable and AX actions do not report transitions. This is a tool/environment blocker, not a proven product button defect.
4. **Packaged isolation:** explicit audit profile and exact-build launch are missing; normal relocation can hand off to installed Kennel. Preserve normal install protections and user data.
5. **Plan quality:** real gpt-5-nano proposals can produce commands that exit successfully without asserting the criterion. Previous baseline printed the wrong value and still exited0. The Plan was not approved. Grounded check semantics need a meaningful falsifier, not only schema-valid commands.
6. Review durable delivery integration, generated parity, full gates, and then run repository -> clarification -> Contract -> Plan -> approval -> automatic execution -> retained evidence -> Verification -> explicit user acceptance -> authorized delivery.

## Explicitly open

Codex subscription reasoning remains fail-closed because read/tool confinement is unproven. Claude/OpenCode/Cursor subscription reasoning and the complete agent bridge/plugin are not delivered by this checkpoint. Rework, complete usage attribution, native end-to-end recovery and owner acceptance remain open. API access is available for bounded tests; it is not provider execution conformance. No tests or provider completion count as user acceptance.

The native audit used a disposable Agent Orchestrator clone. The original repository was preserved. Full local audit logs remain local; no credentials, private transcripts or screenshots are published in this PR.
