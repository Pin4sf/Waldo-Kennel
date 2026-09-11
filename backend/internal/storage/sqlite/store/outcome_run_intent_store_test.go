package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func runIntent(outcomeID domain.OutcomeID, planID domain.PlanRevisionID, desired domain.RunIntentDesired, key string) domain.OutcomeRunIntent {
	return domain.OutcomeRunIntent{
		ID: domain.RunIntentID("ri-" + key), OutcomeID: outcomeID, Desired: desired,
		PlanRevisionID: planID, ContractRevisionNumber: 1,
		RequestKey: key, RequestedAt: time.Unix(100, 0).UTC(),
	}
}

func commandedRunIntent(outcomeID domain.OutcomeID, planID domain.PlanRevisionID, desired domain.RunIntentDesired, command domain.RunCommand, expectedGeneration int64, key, fingerprint string) domain.OutcomeRunIntent {
	intent := runIntent(outcomeID, planID, desired, key)
	intent.Command = command
	intent.ExpectedGeneration = expectedGeneration
	intent.ExpectedGenerationSet = true
	intent.RequestFingerprint = fingerprint
	return intent
}

func TestAppendRunIntent_ConflictsOnCompleteReplayIdentityAndStaleGeneration(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "runintent-fences")

	started, err := s.AppendRunIntent(ctx, commandedRunIntent(outcomeID, plan.ID, domain.RunIntentRunning, domain.RunCommandStart, 0, "same-key", "start/fences"))
	if err != nil {
		t.Fatalf("append start: %v", err)
	}
	paused, err := s.AppendRunIntent(ctx, commandedRunIntent(outcomeID, plan.ID, domain.RunIntentPaused, domain.RunCommandPause, started.Generation, "pause-key", "pause/fences"))
	if err != nil {
		t.Fatalf("append pause: %v", err)
	}
	if paused.Generation != 2 {
		t.Fatalf("pause generation = %d, want 2", paused.Generation)
	}

	_, err = s.AppendRunIntent(ctx, commandedRunIntent(outcomeID, plan.ID, domain.RunIntentRunning, domain.RunCommandStart, 0, "same-key", "different-start/fences"))
	var replayConflict *ports.RunIntentReplayConflictError
	if !errors.As(err, &replayConflict) {
		t.Fatalf("changed replay = %v, want RunIntentReplayConflictError", err)
	}

	_, err = s.AppendRunIntent(ctx, commandedRunIntent(outcomeID, plan.ID, domain.RunIntentRunning, domain.RunCommandResume, started.Generation, "stale-key", "stale-resume/fences"))
	var generationConflict *ports.RunIntentGenerationConflictError
	if !errors.As(err, &generationConflict) {
		t.Fatalf("stale generation = %v, want RunIntentGenerationConflictError", err)
	}
	if generationConflict.Current != paused.Generation {
		t.Fatalf("stale generation current = %d, want %d", generationConflict.Current, paused.Generation)
	}

	secondPlan, secondOutcome := seedApprovedPlan(t, s, "runintent-fences-second")
	_, err = s.AppendRunIntent(ctx, commandedRunIntent(secondOutcome, secondPlan.ID, domain.RunIntentRunning, domain.RunCommandStart, 0, "same-key", "start/fences"))
	if !errors.As(err, &replayConflict) {
		t.Fatalf("cross-outcome replay = %v, want RunIntentReplayConflictError", err)
	}
	if replayConflict.Existing.OutcomeID != outcomeID {
		t.Fatalf("cross-outcome conflict named %s, want original %s", replayConflict.Existing.OutcomeID, outcomeID)
	}
}

func TestAppendRunIntent_ConcurrentInitialCompareAndSwapHasOneWinner(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "runintent-concurrent-cas")
	intents := []domain.OutcomeRunIntent{
		commandedRunIntent(outcomeID, plan.ID, domain.RunIntentRunning, domain.RunCommandStart, 0, "cas-a", "cas-a"),
		commandedRunIntent(outcomeID, plan.ID, domain.RunIntentRunning, domain.RunCommandStart, 0, "cas-b", "cas-b"),
	}
	results := make(chan error, len(intents))
	for _, intent := range intents {
		go func(candidate domain.OutcomeRunIntent) {
			_, err := s.AppendRunIntent(ctx, candidate)
			results <- err
		}(intent)
	}

	winners := 0
	for range intents {
		err := <-results
		if err == nil {
			winners++
			continue
		}
		var conflict *ports.RunIntentGenerationConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("concurrent append = %v, want the loser to fail its CAS", err)
		}
	}
	if winners != 1 {
		t.Fatalf("concurrent append winners = %d, want exactly one", winners)
	}
	history, err := s.ListRunIntents(ctx, outcomeID)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 || history[0].Generation != 1 {
		t.Fatalf("history = %+v, want one generation-1 authorization", history)
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

func TestRecordRunAdmissionFailure_IsWriteOnceAndRejectsStaleGeneration(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "runfailure")
	started, err := s.AppendRunIntent(ctx, commandedRunIntent(outcomeID, plan.ID, domain.RunIntentRunning, domain.RunCommandStart, 0, "runfailure-start", "start/runfailure"))
	if err != nil {
		t.Fatal(err)
	}
	failure := domain.RunAdmissionFailure{Code: "AGENT_PROFILE_NOT_READY", Message: "Profile is unavailable", DetailJSON: `{"detail":"sign in"}`, WorkUnitID: plan.WorkUnits[0].ID, OccurredAt: time.Unix(500, 0).UTC()}
	recorded, err := s.RecordRunAdmissionFailure(ctx, outcomeID, started.Generation, failure)
	if err != nil || !recorded {
		t.Fatalf("recorded=%v err=%v", recorded, err)
	}
	recorded, err = s.RecordRunAdmissionFailure(ctx, outcomeID, started.Generation, domain.RunAdmissionFailure{Code: "OTHER", Message: "rewrite", DetailJSON: `{}`, WorkUnitID: plan.WorkUnits[0].ID, OccurredAt: time.Unix(600, 0).UTC()})
	if err != nil || recorded {
		t.Fatalf("rewrite recorded=%v err=%v, want write-once no-op", recorded, err)
	}
	current, found, err := s.CurrentRunIntent(ctx, outcomeID)
	if err != nil || !found || current.AdmissionFailure == nil || current.AdmissionFailure.Code != failure.Code {
		t.Fatalf("current = %+v found=%v err=%v", current, found, err)
	}
	paused, err := s.AppendRunIntent(ctx, commandedRunIntent(outcomeID, plan.ID, domain.RunIntentPaused, domain.RunCommandPause, started.Generation, "runfailure-pause", "pause/runfailure"))
	if err != nil {
		t.Fatal(err)
	}
	recorded, err = s.RecordRunAdmissionFailure(ctx, outcomeID, started.Generation, failure)
	if err != nil || recorded {
		t.Fatalf("stale failure recorded=%v err=%v, want no-op after generation %d", recorded, err, paused.Generation)
	}
	history, err := s.ListRunIntents(ctx, outcomeID)
	if err != nil || len(history) != 2 || history[0].AdmissionFailure == nil || history[1].AdmissionFailure != nil {
		t.Fatalf("history = %+v err=%v, want preserved old blocker and clean new generation", history, err)
	}
}
