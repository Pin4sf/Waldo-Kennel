package store_test

import (
	"context"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func TestOutcomeStore_CanonicalContractWriterRoundTripsExecutionPreference(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedProject(t, s, "mer")
	space, err := s.EnsureWorkResponsibilitySpace(ctx, "mer")
	if err != nil {
		t.Fatalf("ensure space: %v", err)
	}
	outcome, first := focusLedgerContract(space.ID, "preference-canonical")
	first.ExecutionPreference = &domain.ExecutionPreference{
		Provider:       domain.HarnessClaudeCode,
		ModelSelection: domain.ExecutionPreferenceModelExplicit,
		Model:          "sonnet-test",
	}
	if err := s.CreateOutcomeWithContract(ctx, outcome, first, "req-preference-canonical"); err != nil {
		t.Fatalf("create outcome: %v", err)
	}

	history, err := s.ListContractRevisions(ctx, outcome.ID)
	if err != nil {
		t.Fatalf("list first revision: %v", err)
	}
	if len(history) != 1 || history[0].ExecutionPreference == nil {
		t.Fatalf("first revision preference = %+v", history)
	}
	if got := *history[0].ExecutionPreference; got.Provider != domain.HarnessClaudeCode || got.ModelSelection != domain.ExecutionPreferenceModelExplicit || got.Model != "sonnet-test" {
		t.Fatalf("first preference = %+v", got)
	}

	second := first
	second.ID = domain.ContractRevisionID("cr-preference-canonical-2")
	second.Number = 0 // storage assigns the immutable next number
	second.Goal = "Second immutable goal with a provider-default preference."
	second.ExecutionPreference = &domain.ExecutionPreference{
		Provider:       domain.HarnessCodex,
		ModelSelection: domain.ExecutionPreferenceModelProviderDefault,
	}
	for i := range second.Criteria {
		second.Criteria[i].ContractRevisionID = second.ID
		second.Criteria[i].ID = domain.CriterionID("crit-preference-canonical-2")
	}
	if _, err := s.AppendContractRevision(ctx, outcome.ID, 1, second); err != nil {
		t.Fatalf("append revision: %v", err)
	}

	history, err = s.ListContractRevisions(ctx, outcome.ID)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(history) != 2 || history[1].ExecutionPreference == nil {
		t.Fatalf("second revision preference = %+v", history)
	}
	if got := *history[1].ExecutionPreference; got.Provider != domain.HarnessCodex || got.ModelSelection != domain.ExecutionPreferenceModelProviderDefault || got.Model != "" {
		t.Fatalf("second preference = %+v", got)
	}
}
