# Kennel status

As of the F0-F6 foundation branch on 2026-08-18, Kennel has a working AO-derived coding-agent orchestration chassis with isolated installed identity and state. It does not yet have the Mission/personal-agent architecture that will define the Waldo product.

## Shipped in the current chassis

### Backend and CLI

- A Go daemon bound to `127.0.0.1`, with health/readiness/control endpoints and an opt-in authenticated home-LAN listener.
- SQLite persistence, additive migrations, trigger-based change-data capture, SSE invalidation/replay, and durable session/chat facts.
- A thin `kennel` Cobra CLI that uses daemon HTTP rather than opening storage or adapters directly.
- Project and session lifecycle, native chat and terminal interfaces, worktree management, recovery, PR/check/review observation, terminal mux, browser preview/control, and a broad inherited provider-adapter catalog.
- Generated OpenAPI and frontend TypeScript contracts with drift checks.

### Desktop supervisor

- Electron + React 19, a generated daemon client, project/session views, terminal and native chat surfaces, notification and PR context, browser preview/DevTools, and settings.
- A prototype Outcome clarification/planning/Kanban overlay. It is present and tested, but it is not the accepted Mission or personal-agent model.
- Kennel-owned packaging: `Kennel.app`, `in.heywaldo.kennel`, `kennel`, `kennel-app`, `~/.kennel`, `kennel-updater`, and `Pin4sf/Waldo-Kennel`.

### Foundation controls

- Pinned AO provenance and a non-destructive synchronization procedure.
- Compatibility reconciliation for the colliding migration 0098 ledgers.
- Reproducible Node/npm selection and a multi-package bootstrap.
- Local foundation tests, Go lint/race checks, production dependency audit, macOS packaged-identity assertion, Dependabot, and GitHub secret scanning. Hosted CI/security workflows are intentionally deferred.
- Zero known production npm vulnerabilities across the audited package sets at the dated foundation run. Inherited development toolchain advisories remain documented in the [acceptance record](foundation-acceptance-2026-08-18.md).

## Present but not a current product promise

- `frontend/src/landing` is an AO marketing donor retained for build coverage; it is not the desktop launch surface.
- `packages/mobile` is an Expo donor, not a currently claimed Kennel mobile product.
- `packages/cloud-client` is a tested compatibility package, not proof of a deployed Kennel cloud service.
- Frozen `packages/ao*`, release, pod, and updater helpers remain for controlled compatibility/migration. This foundation does not publish them.
- The Go module, `backend/cmd/ao`, and some internal AO vocabulary remain deliberate source synchronization seams.

## Not shipped

- Mission as Waldo's governing unit of intent.
- A user-owned personal-agent identity or authority model.
- Personal memory admission, correction, provenance, conscious closure, or release semantics.
- Waldo verification/acceptance contracts beyond the inherited coding-work evidence model.
- Xirp, Medley, Paxel, or other named integrations.
- A decision that the current Outcome overlay is the final product model.
- Shared Git ancestry with AO. Repairing ancestry would rewrite published history and requires separate explicit approval.

## Verification

From a normalized checkout:

```sh
npm run bootstrap
npm run test:foundation
npm run audit:production
npm run lint
cd backend && go test -race ./...
```

Package identity is verified without launching the application:

```sh
npm --prefix frontend run package
npm --prefix frontend run package:identity
```

The next architecture entrypoint after this foundation is accepted is a written Mission/personal-agent contract covering ownership, authority, memory, verification, acceptance, and Kennel's role as a Waldo presence. No implementation should precede that decision.
