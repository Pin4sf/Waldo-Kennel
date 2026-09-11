# Launch checkpoint — 12 September 2026

This is a release-validation record for the reviewed integration branch. It is a checkpoint for beta review, not a claim of native parity, production readiness, or owner acceptance.

## Source and ancestry

- Release branch: `codex/launch-checkpoint-20260912`
- Validated source checkpoint: `68831ab03c060c2f25384ded511a8b3f2d386dd4` (a subsequent documentation-only commit adds this record).
- Integrated source before this documentation-only checkpoint: `36742c91963ae035409d23b9e44f6f79c03ec2a8`
- GitHub remote: `https://github.com/Pin4sf/Waldo-Kennel.git`
- Fetched `origin/beta`: `88f63f040e843cf9dbf3d87cd01e5943f4c70b81`
- Ancestry: the validated source checkpoint is a clean descendant of `origin/beta` (0 behind, 25 ahead at that checkpoint).

## Fresh release checks

- `npm run sqlc` passed using the local sqlc module cache and checkout-local Go cache; generated tree unchanged.
- `npm run api` passed; OpenAPI and frontend TypeScript generated trees unchanged.
- `npm run frontend:typecheck` passed.
- Full `npm --prefix frontend test` completed 230 files / 2,783 tests: 2,757 passed, 6 skipped, and 20 socket-bound tests could not run in the default sandbox (`listen EPERM`/timeouts in `supervisor-link`, `browser-runtime-link`, `agent-browser-cdp-bridge`, and real-daemon `daemon-attach`). The same four files were rerun with loopback/Unix-socket permission escalation: 4 + 4 + 5 + 33 = 46 passed, covering those 20 failures. Combined result: 2,777 passed, 6 skipped, 0 assertion failures; the default-sandbox limitation is retained for provenance.
- `npm --prefix frontend run package` passed with the verified ignored browser runtime and ACP runtime reused from the integrated checkout; daemon build, native helper checks, Vite bundles, Electron Forge arm64 packaging, and post-package hooks completed.
- `npm --prefix frontend run package:identity` passed: app ID `in.heywaldo.kennel`, display name `Kennel`, executable `kennel`, protocol `kennel-app`, release repository `Pin4sf/Waldo-Kennel`.
- Packaged artifact: `/private/tmp/kennel-launch-checkpoint-20260912/frontend/out/Kennel-darwin-arm64/Kennel.app`.
- Backend source tree is unchanged from the exact validated tree `2178aa6e0c08fd991c9a7c3c41110cbceb74d0a0`; fresh build/vet passed, and prior exact-tree Go test/race/lint evidence remains recorded at the 525ebe4 checkpoint. A redundant sandboxed all-tests attempt is not counted as fresh pass evidence because its listener/tmux boundary was denied.

## Remaining acceptance boundaries

The checkpoint intentionally does not claim signed-in native Codex conversational planning parity, a live accepted OpenAI credential (the configured credential was rejected), autonomous governed completion, native desktop terminal I/O/resize conformance, an unassisted Outcome loop, production deployment, or the user's `AcceptanceDecision`. A PR, package, provider completion, green test, or generated contract is evidence only.
