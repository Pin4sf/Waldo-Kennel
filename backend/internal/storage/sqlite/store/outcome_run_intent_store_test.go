package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func runIntent(outcomeID domain.OutcomeID, planID domain.PlanRevisionID, desired domain.RunIntentDesired, key string) domain.OutcomeRunIntent {
	return domain.OutcomeRunIntent{
		ID: domain.RunIntentID("ri-" + key), OutcomeID: outcomeID, Desired: desired,
		PlanRevisionID: planID, ContractRevisionNumber: 1,
		RequestKey: key, RequestedAt: time.Unix(100, 0).UTC(),
	}
}

// TestAppendRunIntent_NumbersGenerationsAndReplaysARepeatedCommand pins the
// two properties continuation depends on: the store chooses the generation,
// and a repeated command returns the one it already produced.
func TestAppendRunIntent_NumbersGenerationsAndReplaysARepeatedCommand(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "runintent")

	started, err := s.AppendRunIntent(ctx, runIntent(outcomeID, plan.ID, domain.RunIntentRunning, "rk-start"))
	if err != nil {
		t.Fatalf("append start: %v", err)
	}
	if started.Generation != 1 {
		t.Fatalf("first generation = %d, want 1", started.Generation)
	}
	paused, err := s.AppendRunIntent(ctx, runIntent(outcomeID, plan.ID, domain.RunIntentPaused, "rk-pause"))
	if err != nil {
		t.Fatalf("append pause: %v", err)
	}
	if paused.Generation != 2 {
		t.Fatalf("second generation = %d, want 2", paused.Generation)
	}

	// The caller's generation is ignored: a client that guessed would
	// otherwise be able to overwrite somebody else's authorization.
	guessed := runIntent(outcomeID, plan.ID, domain.RunIntentRunning, "rk-resume")
	guessed.Generation = 99
	resumed, err := s.AppendRunIntent(ctx, guessed)
	if err != nil {
		t.Fatalf("append resume: %v", err)
	}
	if resumed.Generation != 3 {
		t.Fatalf("third generation = %d, want the store's own numbering", resumed.Generation)
	}

	replayed, err := s.AppendRunIntent(ctx, runIntent(outcomeID, plan.ID, domain.RunIntentPaused, "rk-pause"))
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if replayed.Generation != paused.Generation {
		t.Fatalf("replay produced generation %d, want the original %d", replayed.Generation, paused.Generation)
	}
	history, err := s.ListRunIntents(ctx, outcomeID)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("history = %d generations, want three; a replay must not append", len(history))
	}
}

// TestCurrentRunIntent_IsTheLatestGenerationOnly is what a restarted daemon
// reads. An older generation that once said running must never restart work
// the owner has since paused.
func TestCurrentRunIntent_IsTheLatestGenerationOnly(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "runintentcurrent")

	if _, err := s.AppendRunIntent(ctx, runIntent(outcomeID, plan.ID, domain.RunIntentRunning, "rk-start")); err != nil {
		t.Fatalf("append start: %v", err)
	}
	if _, err := s.AppendRunIntent(ctx, runIntent(outcomeID, plan.ID, domain.RunIntentPaused, "rk-pause")); err != nil {
		t.Fatalf("append pause: %v", err)
	}

	current, found, err := s.CurrentRunIntent(ctx, outcomeID)
	if err != nil || !found {
		t.Fatalf("current: found=%v err=%v", found, err)
	}
	if current.Desired != domain.RunIntentPaused || current.Generation != 2 {
		t.Fatalf("current = %s/%d, want paused/2", current.Desired, current.Generation)
	}

	running, err := s.ListOutcomesWithRunIntent(ctx, domain.RunIntentRunning)
	if err != nil {
		t.Fatalf("list running: %v", err)
	}
	for _, intent := range running {
		if intent.OutcomeID == outcomeID {
			t.Fatalf("a paused Outcome was listed as authorized to run: %+v", intent)
		}
	}
	paused, err := s.ListOutcomesWithRunIntent(ctx, domain.RunIntentPaused)
	if err != nil {
		t.Fatalf("list paused: %v", err)
	}
	if len(paused) != 1 || paused[0].OutcomeID != outcomeID {
		t.Fatalf("paused listing = %+v, want just this Outcome", paused)
	}
}

// TestAcknowledgeRunIntent_IsWriteOnce keeps a pause that has taken effect
// from looking like one still waiting, and vice versa.
func TestAcknowledgeRunIntent_IsWriteOnce(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "runintentack")
	if _, err := s.AppendRunIntent(ctx, runIntent(outcomeID, plan.ID, domain.RunIntentRunning, "rk-start")); err != nil {
		t.Fatalf("append: %v", err)
	}

	first := time.Unix(500, 0).UTC()
	if err := s.AcknowledgeRunIntent(ctx, outcomeID, 1, first); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	if err := s.AcknowledgeRunIntent(ctx, outcomeID, 1, time.Unix(900, 0).UTC()); err != nil {
		t.Fatalf("repeat acknowledge: %v", err)
	}

	current, _, err := s.CurrentRunIntent(ctx, outcomeID)
	if err != nil {
		t.Fatalf("current: %v", err)
	}
	if current.AcknowledgedAt == nil || !current.AcknowledgedAt.Equal(first) {
		t.Fatalf("acknowledgement = %v, want the first one to stand", current.AcknowledgedAt)
	}
}
