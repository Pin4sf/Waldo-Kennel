package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

func TestOutcomeStore_EnsureWorkResponsibilitySpaceIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedProject(t, s, "mer")

	first, err := s.EnsureWorkResponsibilitySpace(ctx, "mer")
	if err != nil {
		t.Fatalf("ensure space: %v", err)
	}
	if first.Kind != domain.ResponsibilitySpaceWorkProject || first.ProjectID != "mer" {
		t.Fatalf("space = %+v", first)
	}
	second, err := s.EnsureWorkResponsibilitySpace(ctx, "mer")
	if err != nil {
		t.Fatalf("re-ensure space: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("space id changed across calls: %s then %s", first.ID, second.ID)
	}

	// Distinct projects get distinct spaces.
	seedProject(t, s, "other")
	third, err := s.EnsureWorkResponsibilitySpace(ctx, "other")
	if err != nil {
		t.Fatalf("ensure other space: %v", err)
	}
	if third.ID == first.ID {
		t.Fatal("distinct projects must not share a Work space")
	}
}

func outcomeFixture(spaceID domain.ResponsibilitySpaceID) domain.Outcome {
	return domain.Outcome{
		ID:        domain.OutcomeID("out-" + spaceID + "-1"),
		SpaceID:   spaceID,
		Title:     "Local Focus Ledger",
		CreatedAt: time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC),
	}
}

func TestOutcomeStore_CreateGetAndIdempotentReplay(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedProject(t, s, "mer")
	space, err := s.EnsureWorkResponsibilitySpace(ctx, "mer")
	if err != nil {
		t.Fatalf("ensure space: %v", err)
	}

	outcome := outcomeFixture(space.ID)
	const requestKey = "req-abc-123"
	if err := s.CreateOutcome(ctx, outcome, requestKey); err != nil {
		t.Fatalf("create outcome: %v", err)
	}

	got, ok, err := s.GetOutcome(ctx, outcome.ID)
	if err != nil || !ok {
		t.Fatalf("get outcome ok=%v err=%v", ok, err)
	}
	if got.ID != outcome.ID || got.SpaceID != space.ID || got.Title != outcome.Title {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.CurrentRevisionNumber != 0 {
		t.Fatalf("fresh outcome revision pointer = %d, want 0", got.CurrentRevisionNumber)
	}

	replayed, ok, err := s.FindOutcomeByIdempotencyKey(ctx, requestKey)
	if err != nil || !ok {
		t.Fatalf("find by idempotency key ok=%v err=%v", ok, err)
	}
	if replayed.ID != outcome.ID {
		t.Fatalf("replay resolved %s, want original %s", replayed.ID, outcome.ID)
	}

	if _, ok, err := s.FindOutcomeByIdempotencyKey(ctx, "req-never-seen"); err != nil || ok {
		t.Fatalf("unknown key ok=%v err=%v", ok, err)
	}

	// A second create with the same client key must be impossible.
	dup := outcome
	dup.ID = domain.OutcomeID("out-duplicate")
	if err := s.CreateOutcome(ctx, dup, requestKey); err == nil {
		t.Fatal("duplicate idempotency key must violate the unique index")
	}
}

func TestOutcomeStore_ContractRevisionAppendOnlyHistory(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedProject(t, s, "mer")
	space, err := s.EnsureWorkResponsibilitySpace(ctx, "mer")
	if err != nil {
		t.Fatalf("ensure space: %v", err)
	}
	outcome := outcomeFixture(space.ID)
	if err := s.CreateOutcome(ctx, outcome, ""); err != nil {
		t.Fatalf("create outcome: %v", err)
	}

	rev1 := domain.ContractRevision{
		ID:              domain.ContractRevisionID("cr-1"),
		OutcomeID:       outcome.ID,
		Goal:            "Record focus locally.",
		SuccessCriteria: []string{"Positive minutes create one block."},
		Review:          "Deterministic checks.",
		CreatedAt:       time.Date(2026, 8, 23, 10, 1, 0, 0, time.UTC),
	}
	num, err := s.NextContractRevisionNumber(ctx, outcome.ID)
	if err != nil || num != 1 {
		t.Fatalf("first revision number = %d err=%v, want 1", num, err)
	}
	rev1.Number = num
	if err := s.CreateContractRevision(ctx, rev1); err != nil {
		t.Fatalf("create revision 1: %v", err)
	}

	rev2 := rev1
	rev2.ID = domain.ContractRevisionID("cr-2")
	rev2.Goal = "Revised goal."
	rev2.Constraints = []string{"Local only."}
	rev2.NonGoals = []string{"No timers."}
	rev2.Clarification = "today means local calendar day"
	rev2.CreatedAt = rev1.CreatedAt.Add(time.Minute)
	if num, err = s.NextContractRevisionNumber(ctx, outcome.ID); err != nil || num != 2 {
		t.Fatalf("second revision number = %d err=%v, want 2", num, err)
	}
	rev2.Number = num
	if err := s.CreateContractRevision(ctx, rev2); err != nil {
		t.Fatalf("create revision 2: %v", err)
	}

	history, err := s.ListContractRevisions(ctx, outcome.ID)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(history) != 2 || history[0].Number != 1 || history[1].Number != 2 {
		t.Fatalf("history = %+v", history)
	}
	if history[0].Goal != rev1.Goal || history[1].Goal != rev2.Goal {
		t.Fatal("revision content drifted from what was written")
	}
	if len(history[1].Constraints) != 1 || history[1].Constraints[0] != "Local only." ||
		len(history[1].NonGoals) != 1 || history[1].NonGoals[0] != "No timers." ||
		history[1].Clarification != "today means local calendar day" {
		t.Fatalf("optional fields round-trip failed: %+v", history[1])
	}

	// Duplicate numbers are rejected by storage even if handed out wrong.
	dupNumber := rev1
	dupNumber.ID = domain.ContractRevisionID("cr-dup")
	if err := s.CreateContractRevision(ctx, dupNumber); err == nil {
		t.Fatal("duplicate revision number must be rejected")
	}
}

func TestOutcomeStore_AdvanceContractCASConflicts(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedProject(t, s, "mer")
	space, err := s.EnsureWorkResponsibilitySpace(ctx, "mer")
	if err != nil {
		t.Fatalf("ensure space: %v", err)
	}
	outcome := outcomeFixture(space.ID)
	if err := s.CreateOutcome(ctx, outcome, ""); err != nil {
		t.Fatalf("create outcome: %v", err)
	}
	at := time.Date(2026, 8, 23, 10, 2, 0, 0, time.UTC)

	// Stale CAS against revision 3 while current is 0 conflicts.
	err = s.AdvanceOutcomeContract(ctx, outcome.ID, 3, 4, at)
	var conflict *ports.OutcomeConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("stale advance err = %v, want OutcomeConflictError", err)
	}
	if conflict.CurrentRevisionNum != 0 || conflict.ExpectedRevisionNum != 3 {
		t.Fatalf("conflict detail = %+v", conflict)
	}

	if err := s.AdvanceOutcomeContract(ctx, outcome.ID, 0, 1, at); err != nil {
		t.Fatalf("advance 0 -> 1: %v", err)
	}
	got, _, err := s.GetOutcome(ctx, outcome.ID)
	if err != nil {
		t.Fatalf("reload outcome: %v", err)
	}
	if got.CurrentRevisionNumber != 1 {
		t.Fatalf("pointer = %d, want 1", got.CurrentRevisionNumber)
	}

	// Replaying the same swap now conflicts with the new pointer.
	err = s.AdvanceOutcomeContract(ctx, outcome.ID, 0, 1, at)
	if !errors.As(err, &conflict) {
		t.Fatalf("replayed advance err = %v, want OutcomeConflictError", err)
	}

	// Advancing an unknown outcome also conflicts rather than silently passing.
	err = s.AdvanceOutcomeContract(ctx, domain.OutcomeID("out-missing"), 0, 1, at)
	if !errors.As(err, &conflict) {
		t.Fatalf("unknown-outcome advance err = %v, want OutcomeConflictError", err)
	}
}
