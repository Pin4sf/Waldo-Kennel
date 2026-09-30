# K1 residue: inactive activation boundary

Base: 037018925489e07bd9a958fe7432417876eeeffb. No changes to S4 files, migrations, daemon/router or UI.

Problem and reproduction: devicebridge.PairingCoordinator's admission and recovery latch are instance-local; restarting or constructing a second coordinator can attempt another redemption. Connect is one connection, not a supervised daemon lifecycle. daemon/router currently do not wire the bridge. A lost redeem reply may mean a remote identity exists. None of these are fixed by creating another key and retrying.

Bounded fix: additive activation controller with a repository contract requiring atomic cross-instance reservation before pairing, retained blocked attempts after interruption/ambiguity, safe public status and a gated session factory. Local routes require a trusted local authorizer and are not mounted anywhere. Gate B must supply durable storage. Reconciliation is intentionally not invented. Activation requires a readiness check; no production implementation supplied. Native approvals and exactly two signed command classes remain unchanged.

Crash limitation: S3 generates custody internally before returning. An interrupted attempt stays blocked, but its key locator can be unavailable if S3 dies before returning the checkpoint. Full custody recovery requires an explicitly reviewed S3 integration hook, not a fabricated backend endpoint. This package must not be claimed live-ready.

Tests: atomic reservation under concurrent controllers, ambiguous/interrupted failure retention, owner/device result binding, no code in persisted records or status, readiness denied before session starts, lifecycle cancellation and status truth, API authorization and strict body parsing. Baseline devicebridge/secretstore tests passed on Linux, exit 0, 56 seconds (cold downloads included). No Mac proof, backend proof or Gate B proof.

Final Linux evidence:
- go test -race -count=25 -timeout=30s ./internal/bridgeactivation: exit 0, 2s wall, including reservation concurrency, restart block, safe checkpoints, owner binding, capability-widening rejection, HTTP authorization/strict JSON, dependency gate, terminal revocation and cancellation.
- go test -race -count=1 -timeout=60s ./internal/bridgeactivation ./internal/devicebridge ./internal/secretstore: exit 0, 59s wall before the final capability-equality guard/test (existing devicebridge and secretstore files unchanged). Final package-only race run above is after that guard.
- go vet ./internal/bridgeactivation: exit 0, less than 1s on final code.
- Initial race package run: exit 0, 31s including race compiler cold build.
- Earlier go test -race -count=10 across all three packages was killed by the execution time ceiling at approximately 120s. Only activation output completed. This run is INCOMPLETE, not a pass. Replaced by bounded individual regression run, recorded above.

No full go test ./... run, macOS run, deployed backend run or joint section 9 proof is claimed. Required follow-up: exact-head independent review and Linux/backend/frontend/macOS CI, actual Mac activation/status/restart/revoke checks once the inactive boundary is integrated, S4 final diff collision check, durable repository implementation/proof and reviewed custody-before-network checkpoint hook. No fixed smoke-test duration is promised.

Adversarial-review revision: aa229785 was NOT CLEAN. Reviewer reproduced Start -> revocation -> Stop -> ready factory -> Start succeeding against an unchanged paired record. The original terminal-state test checked only the first session's status. Fix adds an exact-device terminal latch checked before a new factory call, a Repository.Revoke(owner,device) durable transition contract, cancellation on rejection, and persistence-error propagation from Stop. Regression covers their exact restart path and a recreated controller with the mock revoked record, plus failed persistence remaining locally blocked. Final revised package race test (25 repeats) exit 0, 2s; vet exit 0, <1s.

Durability caveat: no production Repository implementation exists in this package. The local latch proves only same-controller safety. Recreated-controller tests demonstrate the mock storage contract, not disk/process death or cross-process proof. Production Revoke must durably fence the exact identity, preserve audit, and share its activation gate with Factory.New; otherwise stale reads/concurrent starts are unsafe. If Revoke fails, Stop reports the error and this controller remains blocked, but persistence/restart safety is NOT established. Real durable fencing/restart proof is required before activation.
