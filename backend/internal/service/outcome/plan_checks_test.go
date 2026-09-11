package outcome

import (
	"errors"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
)

func planCheckAliases() map[string]domain.CriterionID {
	return map[string]domain.CriterionID{"C1": "crit-a", "C2": "crit-b"}
}

func compileCheckCode(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("expected a refusal, got none")
	}
	var api *apierr.Error
	if !errors.As(err, &api) {
		t.Fatalf("err = %v, want a typed apierr", err)
	}
	return api.Code
}

// TestCompileApprovedChecks_ReDecidesEveryPartOfTheProposal is the model-output
// boundary. A proposal may suggest a command; the daemon decides its identity,
// its criterion and its bound.
func TestCompileApprovedChecks_ReDecidesEveryPartOfTheProposal(t *testing.T) {
	checks, err := compileApprovedChecks("wu-a", []domain.PlanDraftCheck{
		{CriterionAlias: "C1", Argv: []string{"go", "test", "./..."}, TimeoutSeconds: 0},
		{CriterionAlias: "C1", Argv: []string{"go", "vet", "./..."}, TimeoutSeconds: 99999},
	}, planCheckAliases(), []domain.CriterionID{"crit-a"})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(checks) != 2 {
		t.Fatalf("checks = %d, want 2", len(checks))
	}
	for _, check := range checks {
		if check.CriterionID != "crit-a" {
			t.Fatalf("alias was not resolved to internal identity: %+v", check)
		}
		// An unbounded or absurd proposed timeout becomes the operational
		// bound, never the proposal's own number.
		if check.TimeoutSeconds != defaultApprovedCheckTimeoutSeconds {
			t.Fatalf("timeout = %d, want the daemon's bound %d", check.TimeoutSeconds, defaultApprovedCheckTimeoutSeconds)
		}
		if !strings.HasPrefix(string(check.ID), "chk-") {
			t.Fatalf("check id %q was not minted by the daemon", check.ID)
		}
	}
	if checks[0].ID == checks[1].ID {
		t.Fatal("two checks share one identity")
	}
}

func TestCompileApprovedChecks_RefusesWhatApprovalCouldNotHonestlyCover(t *testing.T) {
	cases := []struct {
		name     string
		proposed domain.PlanDraftCheck
		owned    []domain.CriterionID
		wantCode string
	}{
		{
			// A shell turns one reviewed line into an unreviewable program.
			name:     "a shell command",
			proposed: domain.PlanDraftCheck{CriterionAlias: "C1", Argv: []string{"sh", "-c", "rm -rf /"}, TimeoutSeconds: 30},
			owned:    []domain.CriterionID{"crit-a"},
			wantCode: "PLAN_DRAFT_CHECK_INVALID",
		},
		{
			name:     "an executable given as a path",
			proposed: domain.PlanDraftCheck{CriterionAlias: "C1", Argv: []string{"/usr/bin/env", "python"}, TimeoutSeconds: 30},
			owned:    []domain.CriterionID{"crit-a"},
			wantCode: "PLAN_DRAFT_CHECK_INVALID",
		},
		{
			name:     "an unknown criterion alias",
			proposed: domain.PlanDraftCheck{CriterionAlias: "C9", Argv: []string{"go", "build"}, TimeoutSeconds: 30},
			owned:    []domain.CriterionID{"crit-a"},
			wantCode: "PLAN_DRAFT_CHECK_CRITERION_UNKNOWN",
		},
		{
			// A check bound to somebody else's criterion would let one
			// WorkUnit's result satisfy another's obligation.
			name:     "a criterion this WorkUnit does not own",
			proposed: domain.PlanDraftCheck{CriterionAlias: "C2", Argv: []string{"go", "build"}, TimeoutSeconds: 30},
			owned:    []domain.CriterionID{"crit-a"},
			wantCode: "PLAN_DRAFT_CHECK_INVALID",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := compileApprovedChecks("wu-a", []domain.PlanDraftCheck{tc.proposed}, planCheckAliases(), tc.owned)
			if code := compileCheckCode(t, err); code != tc.wantCode {
				t.Fatalf("refusal = %q, want %q", code, tc.wantCode)
			}
		})
	}
}

// TestValidateWorkUnitChecksAreExecutable is the planning-quality gate for the
// case the runtime already fails closed on but approval could not see.
//
// governedcheck refuses to launch any check unless the Attempt's frozen policy
// carries worktree execution, so checks on a unit that only reads are recorded
// as unavailable — truthfully, but after approval, after an Attempt, and after
// the owner believed the criterion was covered. Approval has to be able to see
// it, so a Plan that proposes checks a WorkUnit could never run is refused.
func TestValidateWorkUnitChecksAreExecutable(t *testing.T) {
	check := domain.ApprovedCheck{ID: "chk-1", CriterionID: "crit-a", Argv: []string{"go", "test", "./..."}, TimeoutSeconds: 60}

	cases := []struct {
		name         string
		intent       domain.WorkUnitIntent
		checks       []domain.ApprovedCheck
		wantRefusal  bool
		wantErrorHas string
	}{
		{
			name:   "an inspecting unit may carry no checks",
			intent: domain.WorkUnitIntentInspect,
		},
		{
			name:         "an inspecting unit may not carry checks it cannot run",
			intent:       domain.WorkUnitIntentInspect,
			checks:       []domain.ApprovedCheck{check},
			wantRefusal:  true,
			wantErrorHas: domain.CapabilityWorktreeExec,
		},
		{
			name:         "a modifying unit may not either",
			intent:       domain.WorkUnitIntentModify,
			checks:       []domain.ApprovedCheck{check},
			wantRefusal:  true,
			wantErrorHas: domain.CapabilityWorktreeExec,
		},
		{
			name:   "an executing unit may",
			intent: domain.WorkUnitIntentExecute,
			checks: []domain.ApprovedCheck{check},
		},
		{
			name:   "so may one that modifies and executes",
			intent: domain.WorkUnitIntentModifyAndExecute,
			checks: []domain.ApprovedCheck{check},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capabilities, err := tc.intent.RequiredCapabilities()
			if err != nil {
				t.Fatalf("capabilities: %v", err)
			}
			unit := domain.WorkUnit{ID: "wu-a", RequiredCapabilities: capabilities, Checks: tc.checks}
			err = validateWorkUnitChecksAreExecutable(unit)
			if !tc.wantRefusal {
				if err != nil {
					t.Fatalf("unexpected refusal: %v", err)
				}
				return
			}
			if code := compileCheckCode(t, err); code != "PLAN_CHECK_EXECUTION_REQUIRED" {
				t.Fatalf("code = %q, want PLAN_CHECK_EXECUTION_REQUIRED", code)
			}
			if !strings.Contains(err.Error(), tc.wantErrorHas) {
				t.Fatalf("refusal %q does not name %q", err, tc.wantErrorHas)
			}
		})
	}
}

// TestApprovedChecksAreFrozenByThePlanDigest is what makes them authority: if
// the command could change after approval, the owner did not approve what runs.
func TestApprovedChecksAreFrozenByThePlanDigest(t *testing.T) {
	plan := schedulerPlanFixture()
	contract := schedulerProofFixture(plan).Contract
	withCheck := func(argv []string, timeout int64) string {
		t.Helper()
		units := append([]domain.WorkUnit(nil), plan.WorkUnits...)
		units[0].Checks = []domain.ApprovedCheck{{
			ID: "chk-1", CriterionID: "crit-a", Argv: argv, TimeoutSeconds: timeout,
		}}
		digest, err := domain.ComputePlanRunBriefCoreDigest(contract, units, plan.Grants)
		if err != nil {
			t.Fatalf("digest: %v", err)
		}
		return digest
	}

	base := withCheck([]string{"go", "test", "./..."}, 300)
	if base == withCheck([]string{"go", "test", "-run", "Nothing"}, 300) {
		t.Fatal("changing an approved check's command did not change the frozen digest")
	}
	if base == withCheck([]string{"go", "test", "./..."}, 60) {
		t.Fatal("changing an approved check's timeout did not change the frozen digest")
	}
	noChecks, err := domain.ComputePlanRunBriefCoreDigest(contract, plan.WorkUnits, plan.Grants)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	if base == noChecks {
		t.Fatal("adding an approved check did not change the frozen digest")
	}
}
