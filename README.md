# Kennel

Kennel is an independently maintained, AO-derived desktop foundation for supervising coding-agent work. This repository currently ships the orchestration chassis: a Go loopback daemon and `kennel` CLI, an Electron/React supervisor, provider adapters, durable session and terminal infrastructure, worktree/SCM coordination, and a prototype Outcome-oriented UI overlay.

This is a foundation, not a completed Waldo personal agent. Mission, durable personal memory, Waldo authority and verification semantics, and Xirp/Medley/Paxel integrations are not implemented here. The existing Outcome screens are an inherited/prototype product surface; they are not evidence that the broader product architecture has shipped.

## Foundation contract

Kennel is isolated from Agent Orchestrator at every installed-product boundary:

| Boundary | Kennel value |
| --- | --- |
| Product / executable / CLI | `Kennel` / `kennel` |
| macOS bundle and application ID | `in.heywaldo.kennel` |
| Deep-link protocol | `kennel-app` |
| Global state root | `~/.kennel` |
| Updater cache | `kennel-updater` |
| Release repository | `Pin4sf/Waldo-Kennel` |
| Environment namespace | `KENNEL_*` |
| Loopback / development / LAN ports | `3031` / `3032` / `3041` |
| Generated branch namespace | `kennel/` |

Kennel does not read or migrate `~/.ao` or the older `~/.agent-orchestrator` layout. Existing AO installations remain separate, and users add local repositories explicitly through Kennel's supported project flow. Project-local `.ao/attachments` and `.ao/launch.json` names remain temporary upstream compatibility artifacts inside a project; they are not Kennel global state.

The source entrypoint remains `backend/cmd/ao`, and the Go module remains `github.com/aoagents/agent-orchestrator/backend`, as deliberate upstream synchronization seams. Packaged users receive the `kennel` executable and Kennel identifiers. See [identity and state](docs/identity-and-state.md) and [upstream provenance](docs/upstream-provenance.md).

## What is present today

- A local Go daemon with HTTP/SSE/WebSocket APIs, SQLite persistence, lifecycle handling, and change-data capture.
- An Electron desktop supervisor connected through the generated API client.
- Claude Code, Codex, and other inherited coding-harness adapters, terminals, worktrees, browser preview, PR/review facts, and recovery machinery.
- Desktop packaging with asserted Kennel bundle, executable, protocol, updater, release-target, and state identities.
- An Outcome clarification and planning overlay which remains a prototype, not the next product architecture.
- Additive migration compatibility for databases that traversed the colliding AO/Kennel migration 0098 histories.
- Reproducible bootstrap, foundation, dependency, package-identity, and security checks.

Some AO-derived surfaces remain in the tree as donors or compatibility seams and are not current Kennel product promises: `frontend/src/landing`, `packages/mobile`, `packages/cloud-client`, the frozen `packages/ao*` npm packages, and release/pod helper scripts. They stay buildable or inspectable while their future is decided explicitly; they are not published by this foundation work.

## Development

Use Node 22.23.2, npm 10.9.8, and Go 1.25.7. From a clean checkout:

```sh
npm run bootstrap
npm run test:foundation
npm run audit:production
```

Narrow checks are documented in [the development guide](docs/development.md). API changes require `npm run api`; SQLite query/schema changes require `npm run sqlc`.

Do not launch an older packaged build to validate this branch: it can use the pre-isolation identity. Package without launching, then run:

```sh
npm --prefix frontend run package
npm --prefix frontend run package:identity
```

## Upstream and next step

[`upstream.json`](upstream.json) pins the AO source tree, and [`scripts/compare-upstream.sh`](scripts/compare-upstream.sh) supports reviewed, non-destructive synchronization. The repositories do not share Git ancestry; repairing published ancestry would rewrite `main` and therefore requires a separate approved migration.

Once this foundation PR is accepted, the next architecture entrypoint is to define Mission and personal-agent semantics—ownership, authority, memory, verification, and release/closure contracts—before implementing them. That work is deliberately outside this gate.

See the dated [foundation acceptance record](docs/foundation-acceptance-2026-08-18.md) for exact evidence and exclusions.

## License

Kennel is licensed under Apache-2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
