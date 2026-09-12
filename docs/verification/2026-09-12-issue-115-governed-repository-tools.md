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
- Replacing an existing text file preserves its exact file mode, including an
  executable Git mode, even under a restrictive process umask.
- A governed check is conservatively marked uncertain before launch. An
  unconfirmed termination durably latches the private server, refuses later
  write/check effects, survives private-server restart, and is imported into
  canonical Outcome recovery before custody can be released.

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
| `01a095da-e7fd-7ab1-9473-5ec8bf45bcbe` | then-current pre-repair packaged daemon | Repaired the report and passed the exact check; mixed-case `.GIT/config` and `../packaged-final-escape.txt` were both refused. |

`/private/tmp/packaged-escape.txt`, `/private/tmp/review-escape.txt` and
`/private/tmp/packaged-final-escape.txt` were absent after the probes. The
fixture source digest remained unchanged. The then-current pre-repair packaged
daemon used for the last probe had SHA-256:

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

- two independent branch-diff reviews at the pre-repair checkpoint: Spec and
  engineering/security; both reported no actionable findings after their
  findings were fixed
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

An immutable `origin/beta` archive was analyzed separately with the same
pinned linter. It also failed and reported 206 issues while reading the same
stale external-worktree cache paths (86 `dupl`, 93 `errcheck`, 19 `gosec`, six
`nilerr`, one `revive`, and one `staticcheck`). The branch-wide wrapper reported
186 inherited/stale-path issues, while the scoped invocation over every touched
backend package reported `0 issues`. This comparison substantiates that the
repository-wide wrapper failure is not introduced by the Issue #115 diff; it
also shows that cache-dependent totals are not stable enough to use as an
acceptance signal.

## Post-review final-code rerun

The correctness-repair code checkpoint is
`3192b27ea4fa659edde5320d5077755d1ab5dd6d`. A fresh packaged build from that
checkpoint produced:

- embedded daemon SHA-256
  `aa631fbd194a38f27c8fa31ed8f1289912bd8635930c8a4afccf0c3c9de2bcd4`;
- Electron main binary SHA-256
  `d1383e07b3dd17b2783c3364a1df372403b6cc50c3372046f06bb85b92d12920`.

The final-code canary used
`/private/tmp/kennel-issue115-final-canary.fPP6Sr` with an isolated profile,
data directory, and disposable Git repository. The Electron main process was
invoked with that profile but exited before establishing an isolated window,
consistent with an already-running single-instance application. No UI action
is claimed. Canonical setup below was performed through the loopback API of the
daemon embedded in the fresh package.

The final-code API setup created and approved:

- Outcome `out-0ea8100a-8f3f-46dd-ae28-3c3242875d96`;
- Contract `cr-39c4bf5e-3167-4b82-84cf-7ec8133205dd`, revision 1;
- Plan `plan-3970d2c0-c1f9-4d1b-a851-01b6311003fb`, revision 2;
- WorkUnit `wu-1ff373c8-39db-4899-ba0e-441e2b2341f3`, frozen to Codex with
  `provider_default` model semantics;
- approved check `chk-0e08d2af-49ae-4ea8-98ef-2a08ba3ea986-1` with exact argv
  `cmp source.txt report.md`;
- run-brief core digest
  `c7d1491078e1ebb0c36bd680ad4dc965b9ffab4c89dd6594c3c30a7e94c48c46`.

Before the disposable repository had a remote default branch, Attempt
`att-442a3772-ad9d-4999-b25c-7d3749f50b7b` returned
`ATTEMPT_START_UNRESOLVED` and remained canonical/unconfirmed, as required. A
local bare remote and `main` default were then added. The execution environment
refused the next explicit `replace` recovery call because
`confirmProviderStopped=true` is an owner assertion that can authorize duplicate
effects under ambiguous liveness. That call was not bypassed. The isolated
daemon was stopped cleanly, and no replacement Attempt or provider session was
launched.

Therefore the earlier packaged Outcome canary remains valid evidence at its
recorded pre-repair revision, but the requested full replacement, execution,
and restart sequence has **not** been repeated end-to-end on checkpoint
`3192b27ea`. This final-code rerun proves fresh packaging, exact canonical
Plan/route/check setup, and the safe unconfirmed-start boundary only. It does not
upgrade the earlier runtime evidence or establish UI acceptance.

## Independent corrected-code execution and restart canary

A second isolated canary at
`/private/tmp/kennel-issue115-final2.OLionH` left the earlier unconfirmed
Attempt and its data untouched. Its disposable repository and local bare remote
were initialized with a resolvable `main` default before any Attempt. The
packaged daemon reported build revision
`3192b27ea4fa659edde5320d5077755d1ab5dd6d` and the package hashes recorded
above.

This launch used the supported absolute `KENNEL_ELECTRON_DATA_DIR` override,
plus isolated `KENNEL_DATA_DIR`, `KENNEL_RUN_FILE`, and `KENNEL_PORT` values.
Unlike the first rerun, this established the real packaged Electron main
window. Through the UI, the first-run prompt was dismissed, the disposable
repository was imported through the native folder picker, Codex was selected,
and Project `repo` rendered in the Outcome board. Contract and Plan creation
used the loopback API for exact reproducibility. The UI then showed the Outcome
move from `In progress` to `Ready for review`, exposed the succeeded Attempt in
Mission Control, and displayed the retained terminal output.

Canonical identities:

- Outcome `out-c9e277e8-b9df-42c3-9e4f-9b12d2910b9f`;
- Contract `cr-5bc0bf29-b97e-4608-8918-bc4bb71c2aa4`, revision 1;
- criterion `crit-d1792bd3-19c5-405f-9722-bd169cf916d4`;
- Plan `plan-306d3188-8849-45dd-8c9e-b57471fdf4cf`, revision 1;
- WorkUnit `wu-4d62ea03-1009-4a51-ba86-6f4e60d981b2`;
- approved check `chk-a13fb7f0-6bef-4c7e-8675-042279a2191b-1` with argv
  `cmp source.txt report.md`;
- run-brief core digest
  `d195bee1d4bcbe02677157e0624dff2ece5894f0e21089bad2e564a4f1ff1dd9`;
- Attempt `att-d1710d20-5761-4199-899b-78be060039cb`;
- session reference `asr-522d1ff6-ae6f-48bd-aedb-8847200e56fa` and Codex
  session `repo-1`.

The immutable base revision was
`da3c5c74aaf86d79d1ff5ef969bd0cd2f31bcb1e`. Before execution, its source and
report digests were respectively:

```text
7cc28cc977e32de7185ee5455227e25042da1d932d5356fb44e0b0707717234d
d0ddf87eb1b13b41792650e2ef05630791f411f00a58b32ca0479437fb627d2d
```

The stored check record independently confirms `baselineRan=true` and
`baselinePassed=false`. The retained leased worktree was
`/private/tmp/kennel-issue115-final2.OLionH/data/worktrees/repo/repo-1`.
After execution, its source, report, and retained artifact all had the source
digest `7cc28c...7234d`; the registered checkout report retained the original
`d0ddf8...27d2d` digest and a clean Git status. The leased worktree contained
only the intended `report.md` modification and preserved mode `0644`.

The real Codex terminal showed governed repository reads, one governed
`write_text_file`, post-write reads, and
`run_approved_check(chk-a13fb7f0-6bef-4c7e-8675-042279a2191b-1)`. It reported
the exact payload `opal meadow 8621\n`, check exit 0, and stopped after
verification. Canonical storage recorded one observed check run with
`enforced_by=macos-seatbelt-workspace-write`, `termination_unknown=false`, an
artifact version of
`937a09a99404232b6e7e530aae24a8a8bd6784a2b56ffd11711a859a343902ea`,
one supporting deterministic-check Evidence item, and one passed deterministic
Verification run.

The app was quit through its UI and relaunched with the same isolated profile
and daemon data. Before any post-restart inspector action, the API and UI both
showed exactly one succeeded Attempt, one terminated bound session, zero new
Attempts, and the Outcome still `Ready for review`. This establishes canonical
restart persistence without automatic duplication on corrected code.

While closing the post-restart session inspector through accessibility
automation, a stale element action unintentionally invoked `Restore session` on
the already-terminated `repo-1`. It reused the same canonical session identity;
no new Attempt, session row, check run, or file change appeared. A second
restore call was rejected as `duplicate session: repo-1`, and final app shutdown
left no canary daemon/provider process. This occurred after the clean restart
snapshot above, is not automatic recovery evidence, and remains an observed
session-inspector/runtime issue outside Issue #115 rather than a hidden pass.
Both app exits also emitted an unhandled `Object has been destroyed` disposal
warning despite exit 0 and successful daemon shutdown.

Safe uncertainty fault injection was rerun without an owner assertion:

```text
go test ./internal/governedtools -run 'TestServeRecordsUnknownTerminationAndRefusesLaterEffects|TestUncertaintyStorePersistsUnknownTerminationAcrossInstances|TestUncertaintyStoreClearsMarkerOnlyOnConfirmedTermination' -count=1 -v
go test ./internal/service/outcome -run 'TestLivenessLoopBlocksCompletionAfterUnknownGovernedCheckTerminationUntilOwnerReconciles' -count=1 -v
```

All four tests passed. They exercise the actual private MCP request sequence,
durable server restart latch, later-effect refusal, and recovery-before-liveness
import. They are controlled fault-injection evidence, not a packaged live
process-termination probe. The earlier rejected replacement of
`att-442a3772-ad9d-4999-b25c-7d3749f50b7b` remains explicitly open.

## Boundary of this evidence

Attempt success, retained output, deterministic Evidence, and a passed
Verification are not owner Acceptance. No `AcceptanceDecision` was created.
The Outcome remains `Ready for review`; the original rejected replacement and
the post-restart session-inspector observations are not treated as closed by the
successful independent canary.
