# Post-PR99 launch baseline: source and test evidence

Date: 2026-09-08. Source: `67d6946fdd5e5bba1aca7f7002ba75185e7da998`, fetched current beta; GitHub PR99 merged into beta. Audit worktree: `/tmp/kennel-launch-audit-67d6946`. Original user checkout/untracked work preserved.

This is a **documentation handoff check**, not product release acceptance. No product source fix is included. The temporary regression below was run and removed; it is retained here for the first implementer to promote into a permanent test with its fix.

## Executed checks

| Command | Result | Scope/limitation |
|---|---|---|
| `npm run bootstrap` | PASS | Installed locked dependencies; package-manager advisory warnings are not a completed dependency audit |
| backend `go test ./...` | PASS | Existing backend suite; some packages cached; does not include temporary failing regression below |
| backend `go build ./...` | PASS | Build only |
| backend `go vet ./...` | PASS | Standalone vet; broader golangci configuration reports additional issues |
| backend `go test -race ./internal/service/outcome ./internal/service/intake ./internal/daemon ./internal/storage/sqlite/store` | PASS | Targeted race suite, not `go test -race ./...` |
| `npm run frontend:typecheck` | PASS | Typecheck only |
| `npm --prefix frontend test` | FAIL | 4 failed / 226 passed files; 23 failed / 2752 passed / 6 skipped tests |
| `npm run sqlc` and `npm run api` | PASS | Generation produced no tracked source drift |
| `npm --prefix frontend run build` | PASS | Packaged macOS arm64 Electron app; includes daemon/ACP runtime build; app not launched |
| `npm --prefix frontend run package:identity` | PASS | Bundle in.heywaldo.kennel, executable kennel, protocol kennel-app, release repo Pin4sf/Waldo-Kennel |
| `npm run lint` | FAIL | Existing Go suite passes, then golangci-lint reports 190 issues |
| `npx --yes @redwoodjs/agent-ci run --all` | NO COVERAGE | Exit 0 but tool says renamed to run-local-ci and no relevant workflows for audit feature branch; **not a passing CI gate** |
| `git diff --check` and changed-document relative-link/code-fence validation | PASS | Documentation validation; temporary test source removed |
| Temporary Manager-boundary exact-model regression | FAIL (expected RED) | Both explicit and provider-default inherit mutable Project model |

Not executed: full race matrix, full foundation wrapper, live API/model/provider calls, real Electron launch/UX, runtime effect enforcement, external-user acceptance. The foundation wrapper is not inferred green from its component passes; frontend tests remain red. No provider version/model conformance is implied by successful packaging.

Local raw logs from this check are `/tmp/kennel-plan-checks/{bootstrap,go-test,go-build,go-vet,race,typecheck,frontend-test,sqlc,api,frontend-build,package-identity,lint,agent-ci,exact-binding-repro}.log`. These temporary paths are conveniences, not portable evidence dependencies. Re-run commands from this record; salient portable output is preserved below.

## Confirmed exact-binding launch defect

Source chain:

1. `backend/internal/service/session/attempt_spawn.go`, `SpawnExactAttempt`, sets cfg.ExactExecutionBinding and clears cfg.AgentConfig.Model.
2. `backend/internal/session_manager/manager.go`, `Manager.Spawn`, resolves ordinary Project configuration at readiness and launch. It has no use of ExactExecutionBinding or prepareSpawnExecution.
3. `backend/internal/session_manager/exact_execution_binding.go`, prepareSpawnExecution, is unused (also reported by lint). spawnExecutionConfig is used by readiness and its pure tests, so those can pass while actual launch is wrong.
4. The recording adapter below receives mutable-project-model for both approved semantics. This tests actual Manager control flow with fake process/workspace/store; it is not a real provider process test.

Reproduce by putting this source in `backend/internal/session_manager/launch_audit_temp_test.go` (package-local existing test helpers), then running from backend:

```sh
go test ./internal/session_manager -run '^TestLaunchAuditExactBindingAtManagerBoundary$' -count=1
```

```go
package sessionmanager
import (
 "testing"
 "github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
 "github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)
func TestLaunchAuditExactBindingAtManagerBoundary(t *testing.T) {
 for _, tc := range []struct { name string; selection domain.ExecutionBindingModelSelection; model string; want string }{
  {"explicit", domain.ExecutionBindingModelExplicit, "approved-model", "approved-model"},
  {"provider_default", domain.ExecutionBindingModelProviderDefault, "", ""},
 } {
  t.Run(tc.name, func(t *testing.T) {
   st := newFakeStore()
   st.projects["mer"] = domain.ProjectRecord{ID:"mer", Config:domain.ProjectConfig{Worker:domain.RoleOverride{Harness:domain.HarnessCodex, AgentConfig:domain.AgentConfig{Model:"mutable-project-model"}}}}
   agent := &recordingAgent{}
   m := New(Deps{Runtime:&fakeRuntime{}, Agents:singleAgent{agent:agent}, Workspace:&fakeWorkspace{}, Store:st, Messenger:&fakeMessenger{}, Lifecycle:&fakeLCM{store:st}, LookPath:func(string)(string,error){return "/bin/true",nil}})
   binding := domain.ExecutionBinding{Provider:domain.HarnessCodex, ModelSelection:tc.selection, Model:tc.model}
   _,_,_,err := m.Spawn(ctx,ports.SpawnConfig{ProjectID:"mer", Kind:domain.KindWorker, Harness:domain.HarnessCodex, ExactExecutionBinding:&binding})
   if err != nil {t.Fatal(err)}
   if got:=agent.lastLaunch.Config.Model;got!=tc.want {t.Fatalf("actual adapter launch model = %q; approved semantic requires %q",got,tc.want)}
  })
 }
}
```

Observed output (irrelevant fake-binary warnings omitted):

```text
--- FAIL: TestLaunchAuditExactBindingAtManagerBoundary
    --- FAIL: TestLaunchAuditExactBindingAtManagerBoundary/explicit
        actual adapter launch model = "mutable-project-model"; approved semantic requires "approved-model"
    --- FAIL: TestLaunchAuditExactBindingAtManagerBoundary/provider_default
        actual adapter launch model = "mutable-project-model"; approved semantic requires ""
FAIL github.com/Pin4sf/Waldo-Kennel/backend/internal/session_manager
```

L1a must turn this red at the Manager boundary to green. Do not “fix” it by only testing spawnExecutionConfig, clearing all Project config for ordinary sessions, or assuming the adapter effective model equals the request.

## Frontend baseline failures

Observed categories include missing expected Codex text/default-model values. Several names encode historical defaults; this is a lead for fixture/behavior triage, not a verdict that all failures are obsolete. Preserve keyboard, errors and explicit selection coverage while removing hidden-default assumptions.

```text
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > renders one continuous composer surface with a visible settings-style title
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > prefills an Outcome selected from a codebase suggestion
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > dismisses the chrome-free card with Escape
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > submits Codex for a historical project worker with an optional model
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > offers an explicit Terminal UI retry when Chat preflight fails
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > offers only admitted agents when the catalog includes retired agents
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > allows selecting Codex when its auth status is unknown
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > requires an outcome before delegation
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > shows an empty Model field for scratch projects and omits it from delegation
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > submits on Enter and inserts a newline on Shift+Enter in the task
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > does not submit on Alt+Enter or Shift+Enter but does on plain Enter in the task
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > displays daemon start errors for 'UNKNOWN_HARNESS'
FAIL  src/renderer/components/NewTaskDialog.test.tsx > NewTaskDialog > displays daemon start errors for 'INTERNAL'
FAIL  src/renderer/components/Sidebar.test.tsx > Sidebar > defaults a new project to Codex when the catalog includes a worker-only provider
FAIL  src/renderer/components/Sidebar.test.tsx > Sidebar > updates project agent options when the catalog loads after the dialog opens
FAIL  src/renderer/components/SwitchAgentDialog.test.tsx > SwitchAgentDialog > preselects the only ready switch target and excludes worker-only providers
FAIL  src/renderer/components/TaskComposer.test.tsx > TaskComposer > preselects the project worker agent and spawns with it
FAIL  src/renderer/components/TaskComposer.test.tsx > TaskComposer > renders a known default agent without an empty intermediate selection
FAIL  src/renderer/components/TaskComposer.test.tsx > TaskComposer > uses Codex for a historical global default and submits the admitted harness
FAIL  src/renderer/components/TaskComposer.test.tsx > TaskComposer > preselects the agent's default model when the project configures none
FAIL  src/renderer/components/TaskComposer.test.tsx > TaskComposer > does not inherit a historical project's model or mode
FAIL  src/renderer/components/TaskComposer.test.tsx > TaskComposer > shows the same no-override label on the trigger and in the menu
FAIL  src/renderer/components/TaskComposer.test.tsx > TaskComposer > uses the project worker model as the new task model default
Test Files 4 failed | 226 passed (230)
Tests 23 failed | 2752 passed | 6 skipped (2781)
```

## Lint baseline

```text
190 issues:
errcheck 1
goimports 20
govet 1
nilerr 1
revive 151
sqlclosecheck 3
staticcheck 7
unconvert 2
unused 4
```

Representative locations and interpretation:

- `storage/sqlite/store/intelligence_run_store.go:67`: unchecked rows.Close.
- `storage/sqlite/store/outcome_plan_store.go:375,397,419`: nested query rows lifetime/close checks; inspect lifetime before changing defer placement.
- `daemon/attempt_wiring.go:46`: nilerr on invalid binding converted to negative readiness. Determine intended typed contract before altering it.
- `session_manager/exact_execution_binding.go:15`: unused prepareSpawnExecution is a missing production integration, not mere dead code to delete.
- `service/session/attempt_spawn.go:31`: unused normalizedExactModel; remove only after tracing the replacement path.
- `service/agent/roles_test.go`: unused fakeProfileAgent and method.
- `previewserver/manager_test.go:284`: tautological nilness condition under the lint analyzer.
- Most revive issues are exported-comment/naming concerns. Resolve concisely or narrow visibility; do not add long architecture narration or disable checks wholesale.

## Source checks and documentation change scope

Rechecked scheduler proof subject filtering, UI first-unit/harness/proof writes, model request construction, IntelligenceRun runtime callers, Plan proposal reuse, spawn config resolution and fresh migration inventory. Latest merged migration is 0115. API/sqlc output remained unchanged after generation.

Only AGENTS guidance and docs are intended for the beta push. AGENTS edits reconcile existing ADR0012 policy; no new architecture decision is introduced. Current STATUS and the existing implementation plan are updated in place rather than introducing a second execution plan. Historical ADRs remain unchanged.
