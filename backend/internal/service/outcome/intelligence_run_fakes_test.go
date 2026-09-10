package outcome_test

import (
	"context"
	"fmt"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// The Outcome service discovers reasoning provenance by type-asserting its
// store, so every fake store used with WithPlanning must record runs too.
// These live on the base fake so the plan and attempt fakes inherit them.

func (f *fakeStore) ensureIntelRuns() {
	if f.intelRuns == nil {
		f.intelRuns = map[domain.IntelligenceRunID]domain.IntelligenceRun{}
	}
}

func (f *fakeStore) CreateIntelligenceRun(_ context.Context, run domain.IntelligenceRun) error {
	if err := run.Validate(); err != nil {
		return err
	}
	f.ensureIntelRuns()
	if _, exists := f.intelRuns[run.ID]; exists {
		return fmt.Errorf("duplicate intelligence run %s", run.ID)
	}
	f.intelRuns[run.ID] = run
	return nil
}

func (f *fakeStore) GetIntelligenceRun(_ context.Context, id domain.IntelligenceRunID) (domain.IntelligenceRun, bool, error) {
	f.ensureIntelRuns()
	run, ok := f.intelRuns[id]
	return run, ok, nil
}

func (f *fakeStore) ListNonTerminalIntelligenceRuns(context.Context) ([]domain.IntelligenceRun, error) {
	f.ensureIntelRuns()
	var out []domain.IntelligenceRun
	for _, run := range f.intelRuns {
		if !run.Status.Terminal() {
			out = append(out, run)
		}
	}
	return out, nil
}

func (f *fakeStore) RecordIntelligenceRunEffectiveProvenance(_ context.Context, id domain.IntelligenceRunID, provider domain.IntelligenceProviderID, model, native string) error {
	f.ensureIntelRuns()
	run, ok := f.intelRuns[id]
	if !ok {
		return fmt.Errorf("missing intelligence run %s", id)
	}
	run.EffectiveProvider, run.EffectiveModel, run.NativeSessionRef = provider, model, native
	f.intelRuns[id] = run
	return nil
}

func (f *fakeStore) RecordIntelligenceRunMetrics(_ context.Context, id domain.IntelligenceRunID, input, output, duration *int64) error {
	f.ensureIntelRuns()
	run, ok := f.intelRuns[id]
	if !ok {
		return fmt.Errorf("missing intelligence run %s", id)
	}
	if run.InputTokens == nil {
		run.InputTokens = input
	}
	if run.OutputTokens == nil {
		run.OutputTokens = output
	}
	if run.DurationMS == nil {
		run.DurationMS = duration
	}
	f.intelRuns[id] = run
	return nil
}

func (f *fakeStore) UpdateIntelligenceRunStatus(_ context.Context, id domain.IntelligenceRunID, status domain.IntelligenceRunStatus, output domain.SHA256Digest, failureCode, failureSummary string, completedAt *time.Time) error {
	f.ensureIntelRuns()
	run, ok := f.intelRuns[id]
	if !ok {
		return fmt.Errorf("missing intelligence run %s", id)
	}
	run.Status, run.OutputDigest = status, output
	run.FailureCode, run.FailureDetail, run.CompletedAt = failureCode, failureSummary, completedAt
	f.intelRuns[id] = run
	return nil
}
