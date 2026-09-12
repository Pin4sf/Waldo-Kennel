# PR #110 launch-fix execution ledger

Date: 2026-09-12

## Automated verification

| Gate | Result | Evidence |
| --- | --- | --- |
| `npm run lint` | Pass | Full Go suite passed; golangci-lint reported `0 issues` |
| `cd backend && go test -race ./...` | Pass | Full backend race suite exited 0 |
| `cd backend && go build ./...` | Pass | Exited 0 |
| `cd backend && go vet ./...` | Pass | Exited 0 |
| `npm run sqlc` | Pass | Generated storage contracts refreshed |
| `npm run api` | Pass | OpenAPI and frontend TypeScript contracts regenerated together |
| `npm run frontend:typecheck` | Pass | `tsc --noEmit` exited 0 |
| `npm --prefix frontend test` | Pass | 231 files; 2,790 passed and 6 skipped |
| `npm --prefix frontend run package` | Pass | arm64 macOS Electron package finalized successfully |
| `npm --prefix frontend run package:identity` | Pass | app id `in.heywaldo.kennel`, executable `kennel`, release repo `Pin4sf/Waldo-Kennel` |
| `npx @redwoodjs/agent-ci run --all` | Pass/no applicable workflow | Exited 0; tool reports its package rename and no relevant workflows for this branch |

Focused regression coverage also passed for Settings PATCH persistence and
read-failure boundaries, exact RunBrief compilation, replacement admission and
proof replay, Codex App Server cancellation, Codex symlink/sidecar resolution,
governed Python checks, provider branding, Plan/graph separation, Execution
lineage, and actionable Outcome attention.

## Packaged real-app verification

Isolated profile: `/private/tmp/kennel-pr110-electron.nOIT6i`

Packaged application:

`frontend/out/Kennel-darwin-arm64/Kennel.app`

Observed UI behavior:

- Plan text and Work graph are separate selectable views.
- Execution shows the Work graph once, followed by a five-Attempt lineage.
- Every bound Attempt is branded Codex; only the current Attempt exposes Engage.
- Usage remains visible in the execution lineage.
- The Outcome header states the exact attention reason.
- An ended/unclassified Attempt exposes `Replace attempt`.

Stress evidence included six Contract criteria, a long Outcome title, five
Attempts, and a 151,260-token Codex session without collapsing the hierarchy.

## Real Codex execution provenance

- Outcome: `out-832a96a5-dfce-4e09-90e8-447f62bda594`
- Plan: `plan-601565dc-b5cf-443c-8d3e-ed908119ef54`
- WorkUnit: `wu-e73907a1-8779-43b0-b903-c04035de6f6c`
- Replacement Attempt 5: `att-4c92c474-8464-4feb-96a8-daa0f9682952`
- AgentSessionRef: `asr-2079a119-7002-41b3-a614-62c0c275423d`
- Session: `agent-orchestrator-pr110-real-loop-5`
- Harness: `codex`
- Session terminal state: `terminated`
- Attempt terminal state: `reconciled`
- Provider observation: `provider_exit`, exit code `0`
- Transcript: `/Users/shivanshfulper/.codex/sessions/2026/09/12/rollout-2026-09-12T13-33-57-01a094a5-055a-7231-a4c2-8b1fbd784960.jsonl`

The session emitted Kennel activity hooks, initialized through Codex App Server,
and enumerated structured tools/resources. This is direct evidence that the
symlinked CLI no longer loses its native `codex-code-mode-host` sidecar.

The agent stopped with `KENNEL_WORK_STATUS: needs_you`: the approved WorkUnit
permitted its validator command but did not provide a separate read capability
for repository/lease inspection. It authored no report. The Attempt worktree is
clean at `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` and remains inspectable at:

`/private/tmp/kennel-pr110-electron.nOIT6i/data/worktrees/agent-orchestrator-pr110-real-loop/agent-orchestrator-pr110-real-loop-5`

The original repository remained clean and unchanged at
`e01c02eeef1be7b5a7d48f916919a16ae3770740`.

## Open boundaries

- No `AcceptanceDecision` was created.
- PR #110 is not merged by this handoff.
- No GitHub release exists, so updater download/install behavior cannot yet be
  accepted despite verified package identity.
- Mobile rendering is outside this desktop Electron slice.
- Reduced-motion CSS behavior was preserved by code/tests but was not manually
  toggled in the packaged-app run.
