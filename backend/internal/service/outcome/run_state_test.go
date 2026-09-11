package outcome

import (
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func approvedPlanInputs() missionInputs {
	plan := schedulerPlanFixture()
	return missionInputs{
		planFound: true, plan: plan, planBinds: true,
		proof:    schedulerProofFixture(plan),
		schedule: &ScheduleView{Plan: plan, NextRunnableID: "wu-a"},
	}
}

func actionFor(actions []RunActionEligibility, want RunAction) RunActionEligibility {
	for _, action := range actions {
		if action.Action == want {
			return action
		}
	}
	return RunActionEligibility{Action: want, Reason: "MISSING_FROM_VOCABULARY"}
}

func TestDeriveMissionState_ReportsTheMilestoneItsFactsSupport(t *testing.T) {
	plan := schedulerPlanFixture()
	proposed := plan
	proposed.Status = domain.PlanStatusProposed
	running := domain.Attempt{ID: "att-1", Status: domain.AttemptRunning}
	paused := domain.Attempt{ID: "att-2", Status: domain.AttemptPaused}

	cases := []struct {
		name      string
		in        missionInputs
		wantState MissionState
		wantWhy   string
	}{
		{
			name:      "no plan is still being defined",
			in:        missionInputs{proof: schedulerProofFixture(plan)},
			wantState: MissionDefine,
		},
		{
			name:      "a proposed plan awaits authorization",
			in:        missionInputs{planFound: true, plan: proposed, planBinds: true, proof: schedulerProofFixture(plan)},
			wantState: MissionReadyToAuthorize,
		},
		{
			name: "an approved plan whose contract moved on needs the owner",
			in: missionInputs{
				planFound: true, plan: plan, planBinds: false, proof: schedulerProofFixture(plan),
			},
			wantState: MissionNeedsYou, wantWhy: ReasonPlanStale,
		},
		{
			name: "a running attempt is progress, not an owner decision",
			in: missionInputs{
				planFound: true, plan: plan, planBinds: true, proof: schedulerProofFixture(plan), active: &running,
			},
			wantState: MissionInProgress,
		},
		{
			name: "a paused attempt holds custody and needs the owner",
			in: missionInputs{
				planFound: true, plan: plan, planBinds: true, proof: schedulerProofFixture(plan), active: &paused,
			},
			wantState: MissionNeedsYou, wantWhy: ReasonAttemptPaused,
		},
		{
			name: "a runnable unit with nothing running is the owner's move",
			in: missionInputs{
				planFound: true, plan: plan, planBinds: true, proof: schedulerProofFixture(plan),
				schedule: &ScheduleView{Plan: plan, NextRunnableID: "wu-a"},
			},
			wantState: MissionNeedsYou, wantWhy: ReasonStartRequired,
		},
		{
			name: "proof that supports acceptance is ready for review, not accepted",
			in: missionInputs{
				planFound: true, plan: plan, planBinds: true,
				proof: ProofView{Status: ProofStatusReadyForAcceptance},
			},
			wantState: MissionReadyForReview,
		},
		{
			name: "only an acceptance decision reads as accepted",
			in: missionInputs{
				planFound: true, plan: plan, planBinds: true,
				proof: ProofView{Status: ProofStatusAccepted},
			},
			wantState: MissionAccepted,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state, why, blocker := deriveMissionState(tc.in)
			if state != tc.wantState {
				t.Fatalf("state = %q, want %q", state, tc.wantState)
			}
			if why != tc.wantWhy {
				t.Fatalf("attention reason = %q, want %q", why, tc.wantWhy)
			}
			// "Needs you" without a concrete reason is the exact failure this
			// projection exists to prevent.
			if state == MissionNeedsYou {
				if why == "" {
					t.Fatal("needs_you carried no attention reason")
				}
				if blocker == nil || blocker.Code == "" || blocker.Message == "" {
					t.Fatalf("needs_you carried no usable blocker: %+v", blocker)
				}
			}
			if state != MissionNeedsYou && blocker != nil {
				t.Fatalf("non-blocking state %q carried blocker %+v", state, blocker)
			}
		})
	}
}

// runningIntentFor authorizes continuation of exactly the plan in these inputs.
func runningIntentFor(in missionInputs, desired domain.RunIntentDesired) *domain.OutcomeRunIntent {
	return &domain.OutcomeRunIntent{
		Generation: 3, Desired: desired, PlanRevisionID: in.plan.ID,
	}
}

// TestDeriveMissionState_FollowsDurableRunIntentBetweenWorkUnits is the KUX-002
// state half: with nothing in flight, what happens next is decided by the
// recorded authorization, not by the schedule alone.
func TestDeriveMissionState_FollowsDurableRunIntentBetweenWorkUnits(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*missionInputs)
		wantState MissionState
		wantWhy   string
	}{
		{
			name:      "no recorded intent leaves starting to the owner",
			mutate:    func(*missionInputs) {},
			wantState: MissionNeedsYou, wantWhy: ReasonStartRequired,
		},
		{
			name: "an authorized run continues without the owner",
			mutate: func(in *missionInputs) {
				in.intent = runningIntentFor(*in, domain.RunIntentRunning)
			},
			wantState: MissionInProgress,
		},
		{
			name: "a paused run names Resume, not Start",
			mutate: func(in *missionInputs) {
				in.intent = runningIntentFor(*in, domain.RunIntentPaused)
			},
			wantState: MissionNeedsYou, wantWhy: ReasonRunPaused,
		},
		{
			name: "a cancelled run is the owner's to restart",
			mutate: func(in *missionInputs) {
				in.intent = runningIntentFor(*in, domain.RunIntentCancelled)
			},
			wantState: MissionNeedsYou, wantWhy: ReasonStartRequired,
		},
		{
			name: "an authorization naming a superseded Plan can admit nothing",
			mutate: func(in *missionInputs) {
				intent := runningIntentFor(*in, domain.RunIntentRunning)
				intent.PlanRevisionID = "plan-previous"
				in.intent = intent
			},
			wantState: MissionNeedsYou, wantWhy: ReasonRunIntentPlanSuperseded,
		},
		{
			name: "an authorized run with nothing runnable still reports the schedule's reason",
			mutate: func(in *missionInputs) {
				in.intent = runningIntentFor(*in, domain.RunIntentRunning)
				in.schedule = &ScheduleView{Plan: in.plan, NoRunnableReason: NoRunnableAwaitingProof}
			},
			wantState: MissionNeedsYou, wantWhy: string(NoRunnableAwaitingProof),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := approvedPlanInputs()
			in.runIntentsEnabled = true
			tc.mutate(&in)
			state, why, blocker := deriveMissionState(in)
			if state != tc.wantState || why != tc.wantWhy {
				t.Fatalf("state = %q/%q, want %q/%q", state, why, tc.wantState, tc.wantWhy)
			}
			if state == MissionNeedsYou && (blocker == nil || blocker.Code != why || blocker.Message == "") {
				t.Fatalf("needs_you carried no usable blocker: %+v", blocker)
			}
			if state != MissionNeedsYou && blocker != nil {
				t.Fatalf("non-blocking state %q carried blocker %+v", state, blocker)
			}
		})
	}
}

// TestDeriveEligibleActions_RunCommandsMatchTheTransitionPolicy proves the
// offered moves and domain.NextRunIntent cannot disagree — the invariant whose
// absence produced KUX-002.
func TestDeriveEligibleActions_RunCommandsMatchTheTransitionPolicy(t *testing.T) {
	commands := map[RunAction]domain.RunCommand{
		RunActionStart:  domain.RunCommandStart,
		RunActionPause:  domain.RunCommandPause,
		RunActionResume: domain.RunCommandResume,
		RunActionCancel: domain.RunCommandCancel,
	}

	for _, desired := range []domain.RunIntentDesired{
		domain.RunIntentIdle, domain.RunIntentRunning, domain.RunIntentPaused, domain.RunIntentCancelled,
	} {
		t.Run(string(desired), func(t *testing.T) {
			in := approvedPlanInputs()
			in.runIntentsEnabled = true
			in.intent = runningIntentFor(in, desired)
			actions := deriveEligibleActions(RunStateView{}, in)

			for action, command := range commands {
				_, applies := domain.NextRunIntent(desired, command)
				got := actionFor(actions, action)
				// Everything else about these inputs — approved binding Plan,
				// runnable unit, clear gate, no Attempt — permits the command,
				// so the transition policy is the only thing left deciding.
				if got.Available != applies {
					t.Fatalf("%s available = %v (reason %q), but NextRunIntent(%s, %s) applies = %v",
						action, got.Available, got.Reason, desired, command, applies)
				}
			}
		})
	}
}

func TestDeriveMissionState_BlockedContributionOutranksPlanProgress(t *testing.T) {
	in := approvedPlanInputs()
	in.gate = domain.ContributionStartGate{
		Blocked: []domain.UpstreamBlock{{Ref: "c1", Title: "Upstream", Reason: "not accepted"}},
	}
	state, why, blocker := deriveMissionState(in)
	if state != MissionNeedsYou || why != ReasonContributionBlocked {
		t.Fatalf("state = %q/%q, want needs_you/%s", state, why, ReasonContributionBlocked)
	}
	if blocker == nil || blocker.Code != ReasonContributionBlocked {
		t.Fatalf("blocker = %+v, want %s", blocker, ReasonContributionBlocked)
	}
}

func TestDeriveEligibleActions_OffersTheWholeVocabularyWithAReasonWhenRefused(t *testing.T) {
	in := approvedPlanInputs()
	actions := deriveEligibleActions(RunStateView{}, in)

	if len(actions) != 12 {
		t.Fatalf("action count = %d, want the full 12-action vocabulary", len(actions))
	}
	seen := map[RunAction]bool{}
	for _, action := range actions {
		if seen[action.Action] {
			t.Fatalf("action %q offered twice", action.Action)
		}
		seen[action.Action] = true
		if !action.Available && action.Reason == "" {
			t.Fatalf("action %q is unavailable without a reason", action.Action)
		}
		if action.Available && action.Reason != "" {
			t.Fatalf("available action %q carried refusal reason %q", action.Action, action.Reason)
		}
	}
}

func TestDeriveEligibleActions_StartRequiresAnApprovedRunnablePlanAndNothingRunning(t *testing.T) {
	running := domain.Attempt{ID: "att-1", Status: domain.AttemptRunning}
	paused := domain.Attempt{ID: "att-2", Status: domain.AttemptPaused}

	cases := []struct {
		name      string
		mutate    func(*missionInputs)
		wantStart bool
		wantWhy   string
	}{
		{name: "approved, runnable, idle", mutate: func(*missionInputs) {}, wantStart: true},
		{
			name:    "an unapproved plan cannot start",
			mutate:  func(in *missionInputs) { in.plan.Status = domain.PlanStatusProposed },
			wantWhy: ReasonPlanNotApproved,
		},
		{
			name:    "a stale plan cannot start",
			mutate:  func(in *missionInputs) { in.planBinds = false },
			wantWhy: ReasonPlanStale,
		},
		{
			name:    "nothing runnable cannot start",
			mutate:  func(in *missionInputs) { in.schedule = &ScheduleView{Plan: in.plan} },
			wantWhy: ReasonNothingRunnable,
		},
		{
			name:    "a running attempt blocks a second start",
			mutate:  func(in *missionInputs) { in.active = &running },
			wantWhy: ReasonAttemptActive,
		},
		{
			name:    "a paused attempt blocks a start",
			mutate:  func(in *missionInputs) { in.active = &paused },
			wantWhy: ReasonAttemptPaused,
		},
		{
			name: "a blocked contribution cannot start",
			mutate: func(in *missionInputs) {
				in.gate = domain.ContributionStartGate{Blocked: []domain.UpstreamBlock{{Ref: "c1"}}}
			},
			wantWhy: ReasonContributionBlocked,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := approvedPlanInputs()
			tc.mutate(&in)
			start := actionFor(deriveEligibleActions(RunStateView{}, in), RunActionStart)
			if start.Available != tc.wantStart {
				t.Fatalf("start available = %v (reason %q), want %v", start.Available, start.Reason, tc.wantStart)
			}
			if !tc.wantStart && start.Reason != tc.wantWhy {
				t.Fatalf("start refusal = %q, want %q", start.Reason, tc.wantWhy)
			}
		})
	}
}

func TestDeriveEligibleActions_AcceptNeedsProofAndExportNeedsDelivery(t *testing.T) {
	in := approvedPlanInputs()
	in.proof.Status = ProofStatusReadyForAcceptance
	actions := deriveEligibleActions(RunStateView{}, in)

	if accept := actionFor(actions, RunActionAccept); !accept.Available {
		t.Fatalf("accept unavailable at ready_for_acceptance: %q", accept.Reason)
	}
	// Export must stay refused with its real reason while delivery is unbuilt.
	// Offering it would be the one control an owner would most reasonably act on.
	if export := actionFor(actions, RunActionExport); export.Available || export.Reason != ReasonDeliveryUnavailable {
		t.Fatalf("export = %+v, want unavailable/%s", export, ReasonDeliveryUnavailable)
	}

	in.proof.Status = ProofStatusActive
	if accept := actionFor(deriveEligibleActions(RunStateView{}, in), RunActionAccept); accept.Available {
		t.Fatal("accept was offered without proof that supports acceptance")
	}
}

func TestDeriveEligibleActions_AcceptedOutcomeStopsOfferingChanges(t *testing.T) {
	in := approvedPlanInputs()
	in.proof.Status = ProofStatusAccepted
	actions := deriveEligibleActions(RunStateView{}, in)

	for _, action := range []RunAction{RunActionAccept, RunActionRequestChanges, RunActionProposePlan, RunActionClarify} {
		got := actionFor(actions, action)
		if got.Available {
			t.Fatalf("action %q was offered on an accepted Outcome", action)
		}
		if got.Reason != ReasonAlreadyAccepted {
			t.Fatalf("action %q refusal = %q, want %s", action, got.Reason, ReasonAlreadyAccepted)
		}
	}
}

func TestDeriveEligibleActions_PauseAndResumeRefuseWhileRunIntentIsUnbuilt(t *testing.T) {
	in := approvedPlanInputs()
	in.active = &domain.Attempt{ID: "att-1", Status: domain.AttemptRunning}
	actions := deriveEligibleActions(RunStateView{}, in)

	for _, action := range []RunAction{RunActionPause, RunActionResume} {
		got := actionFor(actions, action)
		if got.Available || got.Reason != ReasonRunIntentUnavailable {
			t.Fatalf("action %q = %+v, want unavailable/%s", action, got, ReasonRunIntentUnavailable)
		}
	}
	// Cancel is real today because a single Attempt can be cancelled.
	if cancel := actionFor(actions, RunActionCancel); !cancel.Available {
		t.Fatalf("cancel unavailable with a running attempt: %q", cancel.Reason)
	}
}

func TestActiveAttempt_IgnoresEndedAttempts(t *testing.T) {
	now := time.Now().UTC()
	attempts := []domain.Attempt{
		{ID: "att-old", Status: domain.AttemptRunning, CreatedAt: now.Add(-time.Hour)},
		{ID: "att-done", Status: domain.AttemptSucceeded, CreatedAt: now},
		{ID: "att-lost", Status: domain.AttemptLost, CreatedAt: now},
	}
	active := activeAttempt(attempts)
	if active == nil || active.ID != "att-old" {
		t.Fatalf("active attempt = %+v, want att-old", active)
	}
	if activeAttempt([]domain.Attempt{{ID: "att-done", Status: domain.AttemptSucceeded}}) != nil {
		t.Fatal("a succeeded attempt was reported as active")
	}
}

func TestCriterionCounts_ExcludesDelegatedCriteria(t *testing.T) {
	proof := ProofView{Criteria: []CriterionProofView{
		{Criterion: domain.ContractCriterion{ID: "c1"}, Ready: true},
		{Criterion: domain.ContractCriterion{ID: "c2"}},
		{Criterion: domain.ContractCriterion{ID: "c3"}, Delegated: true},
	}}
	required, proven := criterionCounts(proof)
	if required != 2 || proven != 1 {
		t.Fatalf("counts = %d/%d, want 1/2 proven/required", proven, required)
	}
}
