# Wednesday milestone follow-up verification

Date: 2026-09-09
Branch: `codex/wednesday-milestone-2026-09-09`
Base: `9396c3844ee00cf2c350df0426f4224d33ef87de`
Correction commit: `d617b4ee3`

## Corrected findings

- Reasoning credentials are stored and resolved by provider. A persisted Anthropic credential is not returned after switching to OpenAI; the daemon reports `MISSING_CREDENTIAL` until an OpenAI credential is supplied. Legacy unqualified secret files are not used by the provider-aware file store.
- Repository context excludes explicit environment, credential, private-key and secret-file patterns; nested dependency/build/hidden directories are pruned; cancellation, Git command failures and filesystem traversal/candidate limits stop collection rather than broadening it.
- `RepositoryContextSnapshot` is part of the Contract input digest.
- Outcome CDC events for plan, attempt, proof and correction changes are subscribed by the renderer and invalidate the relevant open schedule query. Plan, attempt, recovery and proof mutations also invalidate schedule projections locally.

## RED/GREEN evidence

Permanent regressions added in the correction commit:

- `go test ./internal/service/settings ./internal/service/intelligence` — RED before the implementation for provider switching, `.env`, cancellation and nested exclusions; GREEN after.
- `go test ./internal/secretstore` — GREEN, including provider-file separation.
- `go test ./internal/service/intelligence` — GREEN, including Git “not ignored” versus Git failure and Contract digest coverage.
- `npm --prefix frontend run test -- --run src/renderer/lib/event-transport.test.ts` — GREEN, including live schedule invalidation from outcome CDC.

Broader gates:

- `go test ./...` — exit 0.
- `go test -race ./internal/service/settings ./internal/service/intelligence ./internal/secretstore ./internal/daemon ./internal/httpd/controllers` — exit 0.
- `go vet ./...` and `go build ./...` — exit 0.
- `npm run lint` — exit 0, `0 issues`.
- `npm --prefix frontend run test` — 230 files passed, 2,776 tests passed, 6 skipped.
- `npm --prefix frontend run typecheck` — exit 0.
- `npm run sqlc`, `npm run api`, generated-contract parity and `git diff --check` — exit 0/clean.

Logs: `/tmp/kennel-followup-go-full.log`, `/tmp/kennel-followup-go-race.log`, `/tmp/kennel-followup-go-vet.log`, `/tmp/kennel-followup-go-build.log`, `/tmp/kennel-followup-lint.log`, `/tmp/kennel-followup-frontend-full.log`, `/tmp/kennel-followup-sqlc.log`, `/tmp/kennel-followup-api.log`, `/tmp/kennel-followup-live-journey.log`.

## Real-daemon journey

An isolated daemon was launched with temporary `KENNEL_DATA_DIR`, `KENNEL_RUN_FILE` and loopback port `31337`; the user's `~/.kennel` state was not used. The journey passed fresh settings readiness, saved an Anthropic canary only in temporary state, switched to OpenAI without a replacement and observed `configured:false`, `ready:false`, `keyConfigured:false`, `errorCode:"MISSING_CREDENTIAL"`. It then registered the current repository and created a direct Outcome without creating an Attempt.

Plan proposal was intentionally blocked at the daemon reasoning boundary because no usable OpenAI credential was configured. The HTTP response was `500 INTERNAL_ERROR`; the daemon request log identified the underlying `reasoning credential is not configured for openai`. No provider/model HTTP call was made, no Attempt was created, and the daemon shut down cleanly. A successful live grounded Contract/Plan and packaged Electron screen journey remain blocked until a real provider credential and the missing code-mode host are available.

The earlier Home-entry failure is not treated as inherited: the follow-up comparison reported by the reviewer passed 6/6 on both the exact base and this branch, so that failure remains unreproduced.

## Acceptance state

| Row | State | Evidence / limitation |
|---|---|---|
| Provider-bound reasoning readiness | PASS | Permanent service, file-store and isolated daemon checks |
| Secret-safe bounded grounding | PASS | Permanent exclusion, Git-error, cancellation and traversal tests |
| Contract provenance includes inspected input | PASS | Digest regression |
| Open Plan schedule updates from daemon events and mutations | PASS | Event transport regression and mutation invalidations |
| Fresh live grounded Contract/Plan | BLOCKED | No live credential/code-mode host; no provider call claimed |
| Packaged Electron Plan/settings journey | BLOCKED | Real daemon API journey passed settings/project/Outcome boundaries; packaged UI and provider-backed Plan remain unverified |

No push, merge, deployment, production data change or owner Acceptance was performed.
