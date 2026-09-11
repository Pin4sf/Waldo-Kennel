# Plan and check proposal quality — evaluation and repair

- **Date:** 11 September 2026
- **Addresses:** KUX-003, "model-generated checks can pass without testing the criterion"
- **Base:** PR #103 head `e573a7229`, on `codex/pr103-launch-repairs`

## The failure this is about

The committed live planning evidence records a proposed check that printed
`Hello World` and exited zero against a criterion about `Hello Kennel` — and
did so again after explicit replan feedback. The check was structurally
perfect: an argument vector, no shell, a bound, a criterion it named and its
WorkUnit owned. Every deterministic rule Kennel had accepted it.

The rules could not have caught it, because none of them ask the only question
that matters: **would this command have noticed if the work had not been done?**
A zero exit is proof of the criterion only when a non-zero exit was possible.

## What was added

### A bounded falsifier: `backend/internal/planquality`

Grades one proposed check against a known-wrong workspace and a correct one,
and reports which of four things it is:

| Verdict | Meaning | Repair |
|---|---|---|
| `discriminating` | failed known-wrong, passed correct | trustworthy — the only verdict that is |
| `vacuous` | passed both | the exit status is independent of the criterion; this is the recorded live failure |
| `broken` | failed both | can never report the criterion met — usually pointed at something absent |
| `inverted` | passed known-wrong, failed correct | measures the criterion backwards |
| `unusable` | could not be graded | the control plane would refuse the check, or the host cannot run it |

Only `discriminating` returns `Trustworthy() == true`.

Three properties keep the grade honest:

- it validates the check with `domain.ApprovedCheck.Validate()` **before**
  running anything, so it can never grade — and so never bless — a command the
  Plan compiler would refuse. A shell is refused without touching a fixture.
- a command that cannot be resolved is `unusable`, not `broken`. "This machine
  has no such tool" is not a verdict about the check.
- an **interrupted** run — a timeout, or a cancelled context — is rejected
  before anything is read as pass or fail, so it is `unusable` on either side.
  This one was wrong in the first version of the grader and is worth stating
  plainly: a command that times out on the known-wrong fixture and exits zero on
  the correct one has the exact shape of discrimination, one side not passing
  and the other passing. It was graded `discriminating` and `Trustworthy()`,
  which would bless a check on the strength of a hang. Nobody knows what a
  timed-out run would have concluded. The symmetric case only ever looked
  `broken`, which is why covering it alone missed this.

It is evaluation, not runtime policy. It runs owner-written fixtures, never a
real workspace, and decides nothing about acceptance.

### Why a falsifier and not a denylist

The tempting fix is to refuse `echo`, `true`, `printf` and friends as check
commands. That would be folklore in the sense `AGENTS.md` forbids: it encodes a
list of mechanisms instead of the invariant, it is wrong for the next vacuous
command nobody listed, and it would reject a legitimate `echo` used as one
argument of a real test harness. The invariant is *discrimination*, and the only
honest way to establish it is to demonstrate it.

### A planning-quality gate that approval can see

`validateWorkUnitChecksAreExecutable` refuses a Plan whose WorkUnit proposes
deterministic checks without requiring `worktree.exec`.

This was a real gap, found while confirming the capability-ceiling half of this
work. `governedcheck.Run` refuses to launch any check unless the Attempt's
frozen policy carries worktree execution — correct, and fail-closed. But
`RequiredCapabilities` is derived purely from the model's declared intent, so a
proposal saying `intent: "inspect"` while carrying `checkCommands` compiled
happily. The owner would then approve a Plan whose criterion coverage looked
complete, run an Attempt, and only afterwards find the checks recorded as
unavailable and the criteria unproved.

The refusal deliberately does **not** widen the unit's capabilities to fit its
checks. Execution authority granted to satisfy a check is also granted to the
provider doing the work, which is a larger authority than anyone proposed.
Naming the wrong intent is the proposal's defect to correct — and when the
Contract's ceiling genuinely forbids local execution, the existing
`PLAN_AUTHORITY_REQUIRED` refusal now fires on a Plan that would otherwise have
been approved with checks that could never run.

New refusal code: `PLAN_CHECK_EXECUTION_REQUIRED` (409). It names the WorkUnit,
the proposed commands and the capabilities it actually has. No request or
response shape changed, so no contract regeneration was required.

## What was confirmed intact

**Explicit provider and model choice.** Plan compilation routes each WorkUnit
through `domain.RouteExecution` with the owner's preference, refuses with
`PLAN_NO_VALID_ROUTE` when no admissible candidate exists, and freezes the
resulting binding on the WorkUnit. Nothing in this slice touches routing, and no
substitution path was added.

**Capability ceilings.** `contractCapabilityCeiling` derives the allowed set
from the confirmed Contract and never invents authority for a zero ceiling;
`validateWorkUnitWithinContractCeiling` refuses any WorkUnit exceeding it. At
runtime, `governedcheck` independently refuses a check without
`worktree.exec` and starts no process — covered by the existing
`TestRunRejectsShellAndCapabilityWidening`, `TestDeniedCheckStartsNoProcess` and
`TestReadOnlyPolicyDeniesEvenTheWorkspace`. The new gate is an additional,
earlier refusal, not a replacement for either.

**Proof strength.** Nothing was weakened. A check that did not run still yields
an inconclusive verification, and a criterion is still ready only when the
latest supporting evidence carries a passing verification bound to it.

## Runtime gating — now enforced

> Added after this evaluation was first written, in the commit that follows it.

The falsifier above grades *proposals*. It did not gate runtime proof:
`checkVerdict` mapped any passing check to `VerificationPassed`, so a vacuous
check that survived review still produced criterion proof. That is now closed.

**The gate.** A check that passed supports its criterion only if it was also
shown to be able to fail. The demonstration is a **known-wrong baseline**: the
same command, under the same frozen Attempt policy, run against a pristine empty
workspace holding none of the work. A command that passes there passes whatever
the Attempt did.

| Real run | Baseline | Verification |
|---|---|---|
| passed | ran, failed | `passed` — the result depended on the work |
| passed | ran, passed | `inconclusive` — this is the recorded live failure |
| passed | not established | `inconclusive` — nothing showed it could fail |
| failed | any | `failed`, as before |
| did not run | any | `inconclusive`, as before |

The evidence summary says which of the two inconclusive cases it was, because
"this check is useless" and "we could not establish a baseline" send an owner
somewhere different. The baseline is durable (migration 0129) so a replayed or
interrupted run does not silently lose it, and an unestablished baseline reads
back as unestablished rather than as one that failed.

**What this claims, precisely.** Failing the baseline establishes *workspace
dependence*. That is a **necessary** condition for a check to be evidence, not a
sufficient one: a check that fails on an empty workspace could still be testing
the wrong thing. Demonstrating that it tests the *right* thing needs
owner-supplied known-wrong and correct fixtures, which is what
`internal/planquality` is for and which is still not wired into approval.

**Costs, stated plainly.** Every check now runs twice — once against the result
and once against an empty directory — bounded by the same timeout. The baseline
runs first, in its own temporary directory that is removed afterwards, so it
cannot observe or disturb the result it is about.

**Consequence for existing Plans.** Any check whose result does not depend on
the workspace stops proving its criterion. That is the intended effect and it is
deliberately loud: such a check was never evidence, and the Outcome it belonged
to was reaching review on a zero exit that meant nothing.

## What is still open

Owner fixtures are not wired into Plan approval, so nothing yet establishes that
a workspace-dependent check tests the *correct* criterion. The evaluator exists
and is regression-tested; connecting it to approval — and deciding whether an
owner must attest discrimination before approving a Plan — remains a product
decision.

## Evidence and limits

Automated, on this branch: `go build ./...`, `go vet ./...`, `go test ./...`,
`go test -race` over `internal/service/outcome` and `internal/planquality`,
`npm run api` (no diff), frontend typecheck, and the frontend suite at 222/222
files with 2,725 passed and 6 skipped — all exit 0.

Red-green: `validateWorkUnitChecksAreExecutable` was written against a failing
test that did not compile until the function existed, and its refusal cases
fail without it.

Not established here, and not claimed:

- **no live provider call.** The vacuous check graded here is the recorded shape
  of the committed live failure, reproduced as a fixture; no model was asked to
  propose a check during this work, so nothing here measures whether a real
  provider now proposes better ones.
- no owner reviewed or accepted any Plan.
- the fixtures are two small greeting workspaces. They demonstrate the grading,
  not the quality of any real Plan.
- `grep` and `sleep` are assumed present; the evaluator reports `unusable`
  rather than a verdict where they are not.
