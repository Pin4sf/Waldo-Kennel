# Waldo Kennel

Waldo Kennel is a local-first desktop control plane for delegated coding-agent work. It keeps the durable responsibility above provider sessions: the user manages **Outcomes**, while Kennel validates, schedules, records, recovers, and explains the execution required to make those Outcomes true.

The repository is an independently maintained, AO-derived foundation that has evolved into a standalone Kennel codebase. The current `beta` already contains a Go daemon, SQLite/change-log persistence, Electron/React supervisor, worktrees, provider sessions, recovery, terminal/chat/browser/PR supervision, and the canonical Outcome lifecycle through explicit user Acceptance. The next kernel program adds the WorkUnit DAG, truthful scheduler/workspace leases, structured receipts, and the final Outcome-first Work projections.

## Start here

Coding agents and contributors should **not** recursively ingest every historical document in this repository. Use the authority chain:

1. [`AGENTS.md`](AGENTS.md)
2. [`docs/product/kennel-v1-product-architecture.md`](docs/product/kennel-v1-product-architecture.md)
3. [`docs/adr/0008-responsibility-composition-and-workunit-execution-dag.md`](docs/adr/0008-responsibility-composition-and-workunit-execution-dag.md)
4. [`docs/adr/0009-workunit-scheduling-workspace-leases-and-effect-fencing.md`](docs/adr/0009-workunit-scheduling-workspace-leases-and-effect-fencing.md)
5. [`docs/STATUS.md`](docs/STATUS.md)
6. [`docs/product/kennel-build-program.md`](docs/product/kennel-build-program.md)
7. [`docs/superpowers/plans/2026-09-04-kennel-builds-kennel.md`](docs/superpowers/plans/2026-09-04-kennel-builds-kennel.md)

The docs index at [`docs/README.md`](docs/README.md) explains precedence, historical material, and future product lanes.

## Product mental model

```text
Waldo proposes / interprets / recommends
              ↓
Kennel validates / schedules / records / enforces
              ↓
Providers execute
```

Canonical Work lineage:

```text
Project
├── ProjectBriefRevision*
└── Outcome*
    └── ContractRevision
        ├── DecompositionRevision → Contributing Outcomes
        └── PlanRevision → WorkUnit DAG
                              ↓
                           Attempt*
                              ↓
                       AgentSessionRef
                              ↓
                    receipts / artifacts
                              ↓
                 Evidence → Verification
                              ↓
                       user Acceptance
```

Key rule:

> **Create another Outcome when responsibility splits. Create another WorkUnit when execution splits.**

Provider Sessions are execution machinery, never the durable responsibility object.

## Provider surface

PR #92 established exactly five active first-class provider identities for new Kennel work:

- Codex
- Claude Code
- OpenCode
- Cursor
- Pi

Readiness is machine-aware. Role admission is capability-driven; the five identities do not imply identical structured-control capabilities. Explicit provider choice never silently falls back to Codex or another provider.

## Installed identity and state

| Boundary | Kennel value |
| --- | --- |
| Product / CLI | `Kennel` / `kennel` |
| macOS bundle ID | `in.heywaldo.kennel` |
| Deep-link protocol | `kennel-app` |
| Global state root | `~/.kennel` |
| Updater cache | `kennel-updater` |
| Release repository | `Pin4sf/Waldo-Kennel` |
| Environment namespace | `KENNEL_*` |
| Generated branch namespace | `kennel/` |

Kennel is standalone. Historical AO source/provenance remains documented where useful, but AO product/provider/task ontology is not current Kennel authority.

## What exists today

See [`docs/STATUS.md`](docs/STATUS.md) for the precise shipped/target boundary. Current `beta` includes:

- local Go daemon with HTTP/SSE/WebSocket surfaces;
- SQLite persistence, additive migrations, trigger-backed CDC;
- thin `kennel` CLI;
- Electron/React desktop supervisor;
- project/session lifecycle and worktrees;
- terminal/native chat/browser/diff/PR/check/review surfaces;
- restart/recovery foundations;
- first-class Codex, Claude Code, OpenCode, Cursor, Pi provider registry/readiness;
- durable Outcome/Contract/Plan/Attempt/Evidence/Verification/Acceptance foundations;
- composed Outcomes and Mission Control destination;
- bounded Project Waldo conversation.

Not yet shipped as the final kernel: `ProjectBriefRevision`, a real multi-WorkUnit DAG, WorkUnit scheduler/WorkspaceLease concurrency, canonical SessionReceipt/WorkUnitReceipt flow, truthful final Mission Graph, final external ingress, and self-hosting proof.

## Development

Use the versions pinned by the repository/toolchain. From a clean checkout, the common verification path is:

```sh
npm run bootstrap
npm run lint
npm run frontend:typecheck
cd backend && go build ./... && go test ./... && go test -race ./... && go vet ./...
cd ../frontend && npm run typecheck && npm run build
```

API changes require `npm run api`. SQLite source changes require `npm run sqlc`. See [`AGENTS.md`](AGENTS.md) for hard boundaries and [`docs/development.md`](docs/development.md) for development detail.

Product/kernel branches start from `beta` and target `beta`. A docs/spec/UX artifact is not evidence that runtime behavior has shipped; update `docs/STATUS.md` when implementation truth changes.

## Self-hosting target

The first kernel milestone is **Kennel builds Kennel**: use Kennel to implement a real repository Outcome with a real WorkUnit DAG, concurrent isolated work where safe, provider recovery, structured receipts/evidence, Ready for Review, and explicit user Acceptance—while the user primarily supervises Board/Mission Control rather than raw provider transcripts.

See [`docs/product/kennel-dogfood-acceptance-matrix.md`](docs/product/kennel-dogfood-acceptance-matrix.md).

## License

Apache-2.0. See [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).
