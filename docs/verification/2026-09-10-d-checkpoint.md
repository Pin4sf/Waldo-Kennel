# D focused review guide

Recorded 2026-09-10 at implementation head `HEAD` on
`codex/kennel-work-completion`.

## Enforced boundary

`backend/internal/governedcheck` is the deterministic check seam. It validates
the immutable AttemptExecutionPolicy, requires `worktree.exec`, pins an
absolute workspace cwd, rejects shell/interpreter argv forms, supplies a
minimal non-inherited environment, bounds combined output, supports timeout
and cancellation, and starts a process group on Unix. It does not claim to be
a provider sandbox; provider-specific enforcement and live canaries remain
open until the supported runtime proves them.

Evidence:

```text
cd backend
go test ./internal/governedcheck -v
```

## Proof boundary

The existing canonical Outcome proof service remains the only path for
Evidence, Verification and user AcceptanceDecision. The R terminal finalizer
requires exact Attempt receipt lineage/version and commits classification,
freeze, observation and fence release together. No provider prose, client
`passed` flag, or check exit code creates AcceptanceDecision.

## Open rows

The check runner currently has unit-level policy/cwd/capability evidence only;
behavioral network/write-denial canaries and provider conformance are not
claimed. An agent-scoped check submission route and full run-intent/rework
integration are still open. The document row remains open until supplied
documents run through the actual Outcome path.
