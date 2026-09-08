package outcome_test

import (
	"context"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// The scheduler consults proof to decide what is still owed before a WorkUnit
// is runnable, so every fake store used with WithExecution must be able to
// answer proof questions. These live on the base fake so the plan and attempt
// fakes inherit them.

type proofFakeState struct {
	evidence    []domain.EvidenceItem
	runs        []domain.VerificationRun
	decisions   []domain.AcceptanceDecision
	corrections []domain.OutcomeCorrection
}

func (f *fakeStore) proofState() *proofFakeState {
	if f.proof == nil {
		f.proof = &proofFakeState{}
	}
	return f.proof
}

func (f *fakeStore) CreateEvidenceItem(_ context.Context, item domain.EvidenceItem) error {
	state := f.proofState()
	state.evidence = append(state.evidence, item)
	return nil
}

func (f *fakeStore) FindEvidenceItemByRequestKey(_ context.Context, key string) (domain.EvidenceItem, bool, error) {
	for _, item := range f.proofState().evidence {
		if item.RequestKey == key && key != "" {
			return item, true, nil
		}
	}
	return domain.EvidenceItem{}, false, nil
}

func (f *fakeStore) GetEvidenceItem(_ context.Context, outcomeID domain.OutcomeID, id domain.EvidenceItemID) (domain.EvidenceItem, bool, error) {
	for _, item := range f.proofState().evidence {
		if item.ID == id && item.OutcomeID == outcomeID {
			return item, true, nil
		}
	}
	return domain.EvidenceItem{}, false, nil
}

func (f *fakeStore) ListEvidenceItems(_ context.Context, outcomeID domain.OutcomeID) ([]domain.EvidenceItem, error) {
	var out []domain.EvidenceItem
	for _, item := range f.proofState().evidence {
		if item.OutcomeID == outcomeID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeStore) CreateVerificationRun(_ context.Context, run domain.VerificationRun) error {
	state := f.proofState()
	state.runs = append(state.runs, run)
	return nil
}

func (f *fakeStore) FindVerificationRunByRequestKey(_ context.Context, key string) (domain.VerificationRun, bool, error) {
	for _, run := range f.proofState().runs {
		if run.RequestKey == key && key != "" {
			return run, true, nil
		}
	}
	return domain.VerificationRun{}, false, nil
}

func (f *fakeStore) ListVerificationRuns(_ context.Context, outcomeID domain.OutcomeID) ([]domain.VerificationRun, error) {
	var out []domain.VerificationRun
	for _, run := range f.proofState().runs {
		if run.OutcomeID == outcomeID {
			out = append(out, run)
		}
	}
	return out, nil
}

func (f *fakeStore) CreateAcceptanceDecision(_ context.Context, decision domain.AcceptanceDecision, correction *domain.OutcomeCorrection) error {
	state := f.proofState()
	state.decisions = append(state.decisions, decision)
	if correction != nil {
		state.corrections = append(state.corrections, *correction)
	}
	return nil
}

func (f *fakeStore) CreateAcceptanceDecisionBatch(_ context.Context, decisions []domain.AcceptanceDecision) error {
	state := f.proofState()
	for _, decision := range decisions {
		if err := decision.Validate(); err != nil {
			return fmt.Errorf("batch decision %s: %w", decision.ID, err)
		}
	}
	state.decisions = append(state.decisions, decisions...)
	return nil
}

func (f *fakeStore) FindAcceptanceDecisionByRequestKey(_ context.Context, key string) (domain.AcceptanceDecision, bool, error) {
	for _, decision := range f.proofState().decisions {
		if decision.RequestKey == key && key != "" {
			return decision, true, nil
		}
	}
	return domain.AcceptanceDecision{}, false, nil
}

func (f *fakeStore) ListAcceptanceDecisions(_ context.Context, outcomeID domain.OutcomeID) ([]domain.AcceptanceDecision, error) {
	var out []domain.AcceptanceDecision
	for _, decision := range f.proofState().decisions {
		if decision.OutcomeID == outcomeID {
			out = append(out, decision)
		}
	}
	return out, nil
}

func (f *fakeStore) ListOutcomeCorrections(_ context.Context, outcomeID domain.OutcomeID) ([]domain.OutcomeCorrection, error) {
	var out []domain.OutcomeCorrection
	for _, correction := range f.proofState().corrections {
		if correction.OutcomeID == outcomeID {
			out = append(out, correction)
		}
	}
	return out, nil
}
