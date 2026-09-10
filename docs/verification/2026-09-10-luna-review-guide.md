# Luna Work reviewer handoff

Review this branch from `origin/beta` `9396c3844ee00cf2c350df0426f4224d33ef87de`
through code head `d96fc3aff` in
`/Users/shivanshfulper/.codex/worktrees/kennel-work-completion`.

## Commit order

The relevant Luna implementation commits are intentionally reviewable in this
order:

1. `646de9f73` — R atomic classification, custody repair, receipt identity and readiness generation.
2. `888f8f568`, `e3c9c6635` — retained bytes and composition primitives.
3. `d8758a642` — governed deterministic check boundary.
4. `782e75c3d` — supplied-document and owner-gated export adapters.
5. `2b5fc669e`, `d02d77ae5`, `59aa8703a` — focused shell, overview projection, evidence update and regression assertion.
6. `0d41cb6c4`, `d96fc3aff` — retention-bound and publication-verification corrections.

The assignment handoff itself is `827efca72`; earlier graph/scheduler and A1
commits are included in the branch history and are described in the ledger.

## Review priorities

- Confirm receipt lineage, artifact-version identity, frozen SQL triggers,
  coherent reads, conditional classification and restart custody repair.
- Confirm retained content is actually retrievable, mode-preserving, bounded,
  secret/symlink-safe and fail-closed on inconsistent reads.
- Confirm composition rejects base/version/path conflicts and does not imply
  successor admission that has not been implemented.
- Confirm governed checks pin cwd, argv, environment, output, timeout and
  process-tree behavior; do not treat unit tests as provider sandbox proof.
- Confirm export requires an owner decision for the same Outcome and never
  mutates source content or overwrites a destination.
- Confirm UI changes remain Outcome projections, preserve approval/start
  separation, and do not claim packaged or daemon-backed acceptance.

## Verified local commands

From the repository root:

```sh
npm run bootstrap
npm run sqlc
npm run api
npm run lint
npm run frontend:typecheck
npx @redwoodjs/agent-ci run --all
```

From `backend/`:

```sh
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

From `frontend/`:

```sh
npm run typecheck
npm run test
npm run build
npm run package:identity
```

The package is local arm64 output at the path recorded in the completion
ledger. No package launch was performed against the owner's profile.

## Explicit open acceptance

Live reasoning/provider conformance is blocked by isolated credentials and
the missing `~/.local/bin/codex-code-mode-host`. The daemon-backed desktop
journey, packaged focused-shell behavior, canonical C-13 successor admission,
full D run intent/rework, document Outcome wiring, durable export HTTP/delivery
state, complete B1/B3 supervision, measured latency, and owner Acceptance are
open. These are not converted into passes by fixture tests, package creation,
or provider-independent adapters.
