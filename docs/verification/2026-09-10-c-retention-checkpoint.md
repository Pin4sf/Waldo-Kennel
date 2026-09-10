# C retention checkpoint

Recorded 2026-09-10 on `codex/kennel-work-completion`.

## Exact state

- Worktree: `/Users/shivanshfulper/.codex/worktrees/kennel-work-completion`
- R head: `646de9f73` (`fix(work): close receipt and reasoning readiness races`)
- C head: `888f8f568` (`feat(work): retain and compose attempt artifacts`)
- Base: `c83684c11627c5851c5a5f2a23e96264b9b9498b`
- `origin/beta`: `9396c3844ee00cf2c350df0426f4224d33ef87de`

## Evidence

The new `backend/internal/artifactstore` package captures daemon-resolved
staged folders and Git worktrees into an application-state root. Git capture
unions `base..HEAD` with working-tree status, preserving committed changes,
dirty edits, untracked files, and deletions. It uses bounded traversal and
bytes, cancellation checks, `Lstat` regular-file checks, secret-like path
refusal, stable-read detection, staged publication, manifest identity, and
verified blob reads. `Compose` rejects incompatible bases and conflicting
writes/deletions; `Apply` verifies destination bytes and modes.

Automated evidence:

```text
cd backend
go test ./internal/artifactstore -v   # pass: staged bytes/mode, committed+dirty Git output, secret/symlink refusal, composition/apply/conflict
go test ./internal/storage/sqlite/store ./internal/service/outcome ./internal/daemon   # pass
```

The daemon reconciliation path now invokes retention before terminal
classification and retries an existing complete receipt by verifying its
published blobs. The SQLite receipt remains the lineage/freeze authority.

## Open acceptance

C-12 is implemented at the adapter/unit level but still needs the full
restart/partial-publication and real-daemon acceptance run. C-13 is not yet
closed: composition primitives exist, but the ordinary successor admission
path does not yet provision a dependency-derived workspace and persist its
input artifact versions. Provider execution and packaged/owner checks remain
separate and are not implied by this checkpoint.
