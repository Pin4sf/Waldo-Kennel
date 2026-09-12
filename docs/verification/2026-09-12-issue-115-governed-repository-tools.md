# Issue #115 governed repository tool verification

Date: 2026-09-12
Base: `origin/beta` at `830d47a8b68eb936a3a0143d7b24cbf6582ea0a8`
Branch: `codex/issue-115-real-work`
Scope: local verification only; no push, PR, merge, deployment or owner Acceptance

## What was proved

- The frozen Attempt policy carries the admitted WorkUnit's exact approved
  checks and the allocated workspace root. Its digest changes if a check vector
  or workspace binding changes, and recovery refuses a different root.
- Codex fresh launch and restore compile the same private `kennel_governed`
  tool server from the frozen policy and leased workspace.
- Governed Codex execution keeps Codex's native sandbox read-only and marks the
  private MCP server required with a positive tool allowlist. Generic
  shell/unified-exec, web, plugin, app, browser, computer-use and multi-agent
  surfaces are also disabled. Unsupported capability sets and executable
  policies without an exact approved check fail before provider launch.
- Governed App Server execution fails closed until that transport has verified
  private-tool injection; Kennel does not silently widen it to native tools.
- The private tool server exposes repository listing and UTF-8 reads, exposes
  writes only with `worktree.write`, and exposes execution only as an approved
  check ID. Descriptor-rooted operations reject absolute paths, traversal,
  symlink escapes and `.git` custody metadata, including concurrent path swaps.
- The exact approved check is handed to the existing sandboxed governed-check
  runner as an argv vector; the tool accepts no arbitrary command string.

## Direct provider probes

The fixture was `/private/tmp/kennel-issue115-baseline2-20260912`. Its opaque
`source.txt` digest was:

```text
046444071b0caff58c6118a036e69a25265e7db40985d50b0dcd8dd3bffe7899
```

Observed Codex 0.153.4 probes:

| Thread | Policy | Observed result |
| --- | --- | --- |
| `01a0959e-acb4-73b1-8784-04ffc8c6f4f0` | read/write/exact check | Read the opaque source, wrote the report, and passed the frozen check. |
| `01a0959f-e252-7273-9088-a206b8976959` | read-only | Read succeeded; no write tool was exposed; `../kennel-issue115-policy.json` was refused as outside the lease. |
| `01a095a1-968b-7b90-a396-4e65cbf6e0e2` | replacement with same frozen policy | Repaired a deliberately corrupted report and passed the same exact check. |
| `01a095ae-267c-7a23-b47d-a5de1598141f` | packaged embedded tool server | Repaired the report; `../packaged-escape.txt` was refused; the exact check passed. |
| `01a095cb-c9b7-7aa1-85b2-d87b60f3b79a` | hardened root-bound policy, native read-only Codex | Repaired the report and passed the exact check; `.git` and `../review-escape.txt` were both refused. |
| `01a095da-e7fd-7ab1-9473-5ec8bf45bcbe` | final packaged daemon | Repaired the report and passed the exact check; mixed-case `.GIT/config` and `../packaged-final-escape.txt` were both refused. |

`/private/tmp/packaged-escape.txt`, `/private/tmp/review-escape.txt` and
`/private/tmp/packaged-final-escape.txt` were absent after the probes. The
fixture source digest remained unchanged. The final packaged daemon used for
the last probe had SHA-256:

```text
1f43661ed0d908bb51adadddab4ddc96e213d5d6fe59e0145a392bb787912749
```

## Real packaged Outcome canary

The packaged application was built from this worktree and launched with an
isolated Electron profile and daemon data directory. Both embedded daemon
binaries had SHA-256:

```text
8b2f34c72d498e11d8301f96a42f045cf0f1130e402083339593d7034facf1c0
```

Canonical IDs:

- Outcome: `out-d9f62eb1-a545-4882-8124-f41313b6b233`
- Contract: `cr-c7e28e36-2a2c-49ac-9706-ed9dd7cc4aa5`, revision 1
- Plan: `plan-ada97b3e-b245-4921-8925-b21703bb06cf`
- WorkUnit: `wu-69149ded-1f2b-49ca-90de-f39efd387e12`
- Check: `chk-fea5611f-2656-485a-b008-1e15cc70c17b-1`
- Prelaunch Attempt: `att-a1f8fca2-f424-4a9a-933b-18f760b3cc6a`
- Replacement Attempt: `att-9b14cb6d-b2c1-4b89-91a6-897bab2c424c`
- Session: `issue115-canary-1`; Codex thread
  `01a095b2-d949-7f12-bc35-d8065bc3586f`

The first Attempt failed during workspace preparation because the disposable
repository initially had no resolvable remote default branch. It created no
provider session, retained its custody fence, and returned `needs_attention`.
After adding a local bare remote/default HEAD, explicit confirmed replacement
recorded the first Attempt as `lost` with a `replacement_attempt` receipt.

Attempt 2 launched one Codex session in Kennel's leased worktree. The recorded
provider sequence was:

1. `list_repository`
2. `read_text_file(source.txt)` and `read_text_file(report.md)`
3. `write_text_file(report.md, "violet orchard 7193\n")`
4. `run_approved_check(chk-fea5611f-2656-485a-b008-1e15cc70c17b-1)`
5. final report, followed by provider exit 0

The governed check reported `enforced_by=macos-seatbelt-workspace-write` and
identical source/report digests. Kennel then classified Attempt 2 as succeeded.
The original registered checkout still contained the deliberately incorrect
report digest `29a93b4804967e733cc008fd750ce466f2673824bb0a3708167dd9e1c118cba8`;
only the leased worktree report changed to the source digest above.

After stopping and restarting the packaged app with the same isolated data,
the daemon retained exactly the lost Attempt 1 and succeeded Attempt 2, plus one
terminated provider session. It did not launch a replacement or duplicate
Attempt during recovery.

## Verification gates

Passed in this worktree:

- two independent final branch-diff reviews: Spec and engineering/security;
  both reported no actionable findings after their findings were fixed
- focused Go tests for domain policy, governed tools, Codex adapter, App Server
  fail-closed behavior, runtime argv, Outcome service, prelaunch persistence,
  path-swap resistance, CLI and telemetry
- scoped `golangci-lint` over every touched backend package (`0 issues`)
- `cd backend && go build ./...`
- `cd backend && go test ./...`
- `cd backend && go test -race ./...`
- `cd backend && go vet ./...`
- `npm run bootstrap`
- `npm run sqlc` (no generated diff)
- `npm run api` (no generated diff)
- `npm run frontend:typecheck`
- `cd frontend && npm run build` (packaged macOS app)
- `npx @redwoodjs/agent-ci run --all` (exit 0; the tool reported its package
  rename and no relevant workflow for this branch)

The repository-wide `npm run lint` wrapper remains noisy on inherited generated
sqlc duplication/unchecked-close findings and stale analyzer paths outside this
worktree. The scoped lint invocation over all touched packages is clean. No
generated file was hand-edited to conceal the inherited wrapper failure.

## Boundary of this evidence

Attempt success is execution evidence, not owner Acceptance. No
`AcceptanceDecision` was created. Retained-artifact ingestion and canonical
WorkUnit-scoped Evidence/Verification remain later launch work; this record does
not infer them from a provider exit, matching hashes or a green local check.
