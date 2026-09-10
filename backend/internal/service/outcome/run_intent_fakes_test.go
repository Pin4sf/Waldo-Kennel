package outcome_test

import (
	"context"
	"sync"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// runIntentFakeStore is the append-only authorization history. It reproduces
// the two properties the design rests on: the generation is chosen by the
// store, and a repeated request key returns the generation it already
// produced rather than authorizing another.
type runIntentFakeStore struct {
	mu         sync.Mutex
	byOutcome  map[domain.OutcomeID][]domain.OutcomeRunIntent
	byRequest  map[string]domain.OutcomeRunIntent
	appendedAt int
}

func newRunIntentFakeStore() *runIntentFakeStore {
	return &runIntentFakeStore{
		byOutcome: map[domain.OutcomeID][]domain.OutcomeRunIntent{},
		byRequest: map[string]domain.OutcomeRunIntent{},
	}
}

func (f *runIntentFakeStore) AppendRunIntent(_ context.Context, intent domain.OutcomeRunIntent) (domain.OutcomeRunIntent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if existing, ok := f.byRequest[intent.RequestKey]; ok {
		return existing, nil
	}
	history := f.byOutcome[intent.OutcomeID]
	intent.Generation = int64(len(history) + 1)
	if err := intent.Validate(); err != nil {
		return domain.OutcomeRunIntent{}, err
	}
	f.byOutcome[intent.OutcomeID] = append(history, intent)
	f.byRequest[intent.RequestKey] = intent
	f.appendedAt++
	return intent, nil
}

func (f *runIntentFakeStore) CurrentRunIntent(_ context.Context, outcomeID domain.OutcomeID) (domain.OutcomeRunIntent, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	history := f.byOutcome[outcomeID]
	if len(history) == 0 {
		return domain.OutcomeRunIntent{}, false, nil
	}
	return history[len(history)-1], true, nil
}

func (f *runIntentFakeStore) FindRunIntentByRequestKey(_ context.Context, key string) (domain.OutcomeRunIntent, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	intent, ok := f.byRequest[key]
	return intent, ok, nil
}

func (f *runIntentFakeStore) ListRunIntents(_ context.Context, outcomeID domain.OutcomeID) ([]domain.OutcomeRunIntent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]domain.OutcomeRunIntent(nil), f.byOutcome[outcomeID]...), nil
}

func (f *runIntentFakeStore) ListOutcomesWithRunIntent(_ context.Context, desired domain.RunIntentDesired) ([]domain.OutcomeRunIntent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.OutcomeRunIntent
	for _, history := range f.byOutcome {
		if len(history) == 0 {
			continue
		}
		// Only the CURRENT generation counts: an older one that said running
		// must never restart work the owner has since paused.
		if current := history[len(history)-1]; current.Desired == desired {
			out = append(out, current)
		}
	}
	return out, nil
}

func (f *runIntentFakeStore) AcknowledgeRunIntent(_ context.Context, outcomeID domain.OutcomeID, generation int64, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	history := f.byOutcome[outcomeID]
	for i := range history {
		if history[i].Generation != generation || history[i].AcknowledgedAt != nil {
			continue
		}
		acknowledged := at.UTC()
		history[i].AcknowledgedAt = &acknowledged
		f.byRequest[history[i].RequestKey] = history[i]
	}
	return nil
}

func (f *runIntentFakeStore) generations(outcomeID domain.OutcomeID) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.byOutcome[outcomeID])
}

var _ ports.RunIntentStore = (*runIntentFakeStore)(nil)
