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

func TestOutcomeStore_EnsureWorkSpaceRejectsUnknownProject(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	if _, err := s.EnsureWorkResponsibilitySpace(ctx, "ghost"); err == nil {
		t.Fatal("space for unknown project must violate the projects foreign key")
	}
}

func focusLedgerContract(spaceID domain.ResponsibilitySpaceID, suffix string) (domain.Outcome, domain.ContractRevision) {
	outcome := domain.Outcome{
		ID:        domain.OutcomeID("out-" + suffix),
		SpaceID:   spaceID,
		Title:     "Local Focus Ledger",
		CreatedAt: time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC),
	}
	first := domain.ContractRevision{
		ID:              domain.ContractRevisionID("cr-" + suffix),
		OutcomeID:       outcome.ID,
		Goal:            "A user can record and review today's protected focus time locally.",
		SuccessCriteria: []string{"Entering positive whole minutes creates one focus block."},
		Review:          "Deterministic checks plus owner walkthrough.",
		Constraints:     []string{"Local only."},
		CreatedAt:       outcome.CreatedAt,
	}
	return outcome, first
}

func TestOutcomeStore_CreateOutcomeWithContractIsAtomic(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedProject(t, s, "mer")
	space, err := s.EnsureWorkResponsibilitySpace(ctx, "mer")
	if err != nil {
		t.Fatalf("ensure space: %v", err)
	}

	outcome, first := focusLedgerContract(space.ID, "1")
	const requestKey = "req-abc-123"
	if err := s.CreateOutcomeWithContract(ctx, outcome, first, requestKey); err != nil {
		t.Fatalf("create outcome with contract: %v", err)
	}

	got, ok, err := s.GetOutcome(ctx, outcome.ID)
	if err != nil || !ok {
		t.Fatalf("get outcome ok=%v err=%v", ok, err)
	}
	if got.CurrentRevisionNumber != 1 {
		t.Fatalf("pointer = %d, want 1 after first contract", got.CurrentRevisionNumber)
	}
	history, err := s.ListContractRevisions(ctx, outcome.ID)
	if err != nil || len(history) != 1 {
		t.Fatalf("history len=%d err=%v, want exactly one immutable revision", len(history), err)
	}
	if history[0].Number != 1 || history[0].Goal != first.Goal {
		t.Fatalf("revision 1 = %+v", history[0])
	}

	replayed, ok, err := s.FindOutcomeByIdempotencyKey(ctx, requestKey)
	if err != nil || !ok || replayed.ID != outcome.ID {
		t.Fatalf("replay ok=%v id=%s err=%v", ok, replayed.ID, err)
	}
	if _, ok, err := s.FindOutcomeByIdempotencyKey(ctx, "req-never-seen"); err != nil || ok {
		t.Fatalf("unknown key ok=%v err=%v", ok, err)
	}

	// A second create with the same client key is impossible.
	dupOutcome, dupFirst := focusLedgerContract(space.ID, "duplicate")
	if err := s.CreateOutcomeWithContract(ctx, dupOutcome, dupFirst, requestKey); err == nil {
		t.Fatal("duplicate idempotency key must violate the unique index")
	}
	// And nothing partial was written for the rejected duplicate.
	if _, ok, _ := s.GetOutcome(ctx, dupOutcome.ID); ok {
		t.Fatal("rejected duplicate must not persist an outcome row")
	}
}

func TestOutcomeStore_AppendContractRevisionHistoryAndConflictRollback(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedProject(t, s, "mer")
	space, err := s.EnsureWorkResponsibilitySpace(ctx, "mer")
	if err != nil {
		t.Fatalf("ensure space: %v", err)
	}
	outcome, first := focusLedgerContract(space.ID, "h")
	if err := s.CreateOutcomeWithContract(ctx, outcome, first, ""); err != nil {
		t.Fatalf("create outcome with contract: %v", err)
	}

	rev2 := domain.ContractRevision{
		ID:              domain.ContractRevisionID("cr-h-2"),
		OutcomeID:       outcome.ID,
		Goal:            "Revised goal.",
		SuccessCriteria: []string{"Criterion A.", "Criterion B."},
		Review:          "Deterministic checks.",
		NonGoals:        []string{"No timers."},
		Clarification:   "today means local calendar day",
		CreatedAt:       first.CreatedAt.Add(time.Minute),
	}
	number, err := s.AppendContractRevision(ctx, outcome.ID, 1, rev2)
	if err != nil || number != 2 {
		t.Fatalf("append revision number=%d err=%v, want 2", number, err)
	}

	got, _, err := s.GetOutcome(ctx, outcome.ID)
	if err != nil {
		t.Fatalf("reload outcome: %v", err)
	}
	if got.CurrentRevisionNumber != 2 {
		t.Fatalf("pointer = %d, want 2", got.CurrentRevisionNumber)
	}

	history, err := s.ListContractRevisions(ctx, outcome.ID)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(history) != 2 || history[0].Number != 1 || history[1].Number != 2 {
		t.Fatalf("history = %+v", history)
	}
	if history[0].Goal != first.Goal {
		t.Fatal("revision 1 content drifted")
	}
	if len(history[1].SuccessCriteria) != 2 ||
		len(history[1].NonGoals) != 1 || history[1].NonGoals[0] != "No timers." ||
		history[1].Clarification != "today means local calendar day" {
		t.Fatalf("revision 2 optional fields round-trip failed: %+v", history[1])
	}

	// A stale expected pointer conflicts AND persists nothing: the append-only
	// history must be identical after the rejected attempt.
	stale := rev2
	stale.ID = domain.ContractRevisionID("cr-h-stale")
	_, err = s.AppendContractRevision(ctx, outcome.ID, 1, stale)
	var conflict *ports.OutcomeConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("stale append err = %v, want OutcomeConflictError", err)
	}
	if conflict.CurrentRevisionNum != 2 || conflict.ExpectedRevisionNum != 1 {
		t.Fatalf("conflict detail = %+v", conflict)
	}
	historyAfter, err := s.ListContractRevisions(ctx, outcome.ID)
	if err != nil || len(historyAfter) != 2 {
		t.Fatalf("rejected append changed history: len=%d err=%v", len(historyAfter), err)
	}
	for _, rev := range historyAfter {
		if rev.ID == stale.ID {
			t.Fatal("rolled-back revision leaked into history")
		}
	}
}
