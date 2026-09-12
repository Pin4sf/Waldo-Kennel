# Waldo Kennel

Waldo Kennel is a local-first desktop control plane for delegated coding-agent work. It keeps the durable responsibility above provider sessions: the user manages **Outcomes**, while Kennel validates, schedules, records, recovers, and explains the execution required to make those Outcomes true.

The repository is an independently maintained, AO-derived foundation that has evolved into a standalone Kennel codebase. The current `beta` contains a Go daemon, SQLite/change-log persistence, Electron/React supervisor, worktrees, provider sessions, recovery, terminal/chat/browser/PR supervision, and the canonical Outcome lifecycle through explicit user Acceptance. The launch milestone is one trustworthy serial Outcome loop. Parallel execution, source-linked Memory, cross-session learning, adaptive orchestration, better Outcome suggestions, and continued UI/brand work follow through explicit gates in the [`ROADMAP.md`](ROADMAP.md).

## Start here

### Try the current beta from source

1. Clone the repository and run `npm run bootstrap`.
2. Start the desktop development app with `npm --prefix frontend run dev`.
3. Register a local Project and configure a supported reasoning path in
   **Settings**: an owner-supplied OpenAI/Anthropic API credential or an
   authenticated native Codex installation.
4. Describe an Outcome, review the grounded Contract, answer planning questions
   and authorize the proposed Plan.
5. Supervise execution in Mission Control, inspect an attached Session when
   needed, then review evidence and explicitly accept or request rework.

This is an active beta. Check [`docs/STATUS.md`](docs/STATUS.md) before relying on
a provider or end-to-end path; provider identity does not imply every role has
passed conformance.

### Contributor authority

Coding agents and contributors should **not** recursively ingest every historical document in this repository. Use the authority chain:

1. [`AGENTS.md`](AGENTS.md)
2. [`docs/product/kennel-v1-product-architecture.md`](docs/product/kennel-v1-product-architecture.md)
3. [`docs/adr/0008-responsibility-composition-and-workunit-execution-dag.md`](docs/adr/0008-responsibility-composition-and-workunit-execution-dag.md)
4. [`docs/adr/0009-workunit-scheduling-workspace-leases-and-effect-fencing.md`](docs/adr/0009-workunit-scheduling-workspace-leases-and-effect-fencing.md)
5. [`docs/STATUS.md`](docs/STATUS.md)
6. [`docs/product/kennel-build-program.md`](docs/product/kennel-build-program.md)
7. [`docs/superpowers/plans/2026-09-04-kennel-builds-kennel.md`](docs/superpowers/plans/2026-09-04-kennel-builds-kennel.md)

Use [`ROADMAP.md`](ROADMAP.md) for public direction and milestone exit gates.
Use [`docs/STATUS.md`](docs/STATUS.md) for current integrated truth; a roadmap
item or open pull request is not shipped behavior.

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

Not yet launch-proven: a packaged end-to-end Outcome journey across planning,
execution, attached sessions, proof/review and owner closure. WorkUnit-scoped
parallel scheduling and workspace leases, complete receipt/materialization
flows, and the final truthful Mission Graph remain roadmap work. See the
revision-stamped inventory in [`docs/STATUS.md`](docs/STATUS.md).

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

## Launch and roadmap

The first launch proof is **Kennel builds Kennel**: use Kennel for a real
repository Outcome through grounded planning, bounded execution, attached
provider sessions, evidence, Ready for Review and explicit user closure—while
the user primarily supervises Board/Mission Control. Serial execution is an
acceptable launch boundary; concurrency is enabled only when the daemon can
truthfully enforce WorkUnit dependencies and workspace custody.

See [`docs/product/kennel-dogfood-acceptance-matrix.md`](docs/product/kennel-dogfood-acceptance-matrix.md).

The public [`ROADMAP.md`](ROADMAP.md) covers project understanding, governed
Memory, capability-driven and learned orchestration, parallel execution,
cross-harness learning, Outcome suggestions, benchmarks and ongoing UI/brand
work.

## License

Apache-2.0. See [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).
