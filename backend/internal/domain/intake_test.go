package domain

import (
	"testing"
	"time"
)

func validIntakeSession() IntakeSession {
	return IntakeSession{
		ID:            "intake-1",
		SourceSurface: IntakeSourceWork,
		Purpose:       IntakePurposeOutcome,
		ProjectID:     "project-1",
		Statement:     "Add keyboard navigation to the settings screen",
		Status:        IntakeStatusCaptured,
		CreatedAt:     time.Date(2026, 8, 26, 1, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 8, 26, 1, 0, 0, 0, time.UTC),
	}
}

func TestIntakeSessionAllowsOnlyCanonicalStateTransitions(t *testing.T) {
	session := validIntakeSession()
	if err := session.Validate(); err != nil {
		t.Fatalf("valid captured IntakeSession: %v", err)
	}

	allowed := map[IntakeStatus][]IntakeStatus{
		IntakeStatusCaptured:       {IntakeStatusAnalyzing, IntakeStatusCancelled},
		IntakeStatusAnalyzing:      {IntakeStatusNeedsUser, IntakeStatusReady, IntakeStatusAnalysisFailed},
		IntakeStatusNeedsUser:      {IntakeStatusAnalyzing, IntakeStatusCancelled},
		IntakeStatusReady:          {IntakeStatusAnalyzing, IntakeStatusConfirmed, IntakeStatusCancelled},
		IntakeStatusAnalysisFailed: {IntakeStatusAnalyzing, IntakeStatusCancelled},
	}
	for from, destinations := range allowed {
		for _, to := range destinations {
			if !CanTransitionIntake(from, to) {
				t.Errorf("CanTransitionIntake(%q, %q) = false, want true", from, to)
			}
		}
	}

	denied := [][2]IntakeStatus{
		{IntakeStatusCaptured, IntakeStatusConfirmed},
		{IntakeStatusNeedsUser, IntakeStatusReady},
		{IntakeStatusConfirmed, IntakeStatusAnalyzing},
		{IntakeStatusCancelled, IntakeStatusReady},
	}
	for _, pair := range denied {
		if CanTransitionIntake(pair[0], pair[1]) {
			t.Errorf("CanTransitionIntake(%q, %q) = true, want false", pair[0], pair[1])
		}
	}
}

func TestIntakeSessionPermitsAtMostOneMaterialClarification(t *testing.T) {
	session := validIntakeSession()
	if !session.CanAskMaterialClarification() {
		t.Fatal("new intake should permit one material clarification")
	}
	session.ClarificationCount = 1
	if session.CanAskMaterialClarification() {
		t.Fatal("intake permitted a second material clarification")
	}
}

func TestOutcomeContractProposalRequiresStableCoreAndTypedFacets(t *testing.T) {
	proposal := OutcomeContractProposal{
		ID:           "proposal-1",
		IntakeID:     "intake-1",
		Revision:     1,
		Title:        "Keyboard settings navigation",
		DesiredState: "Every settings control is reachable and operable by keyboard.",
		Criteria: []ProposedCriterion{{
			ID:               "proposal-criterion-1",
			Text:             "Tab and Shift+Tab reach every interactive settings control in logical order.",
			EvidenceExpected: []string{"A deterministic keyboard-navigation component test passes."},
		}},
		ReviewMethod: "Run the component test and complete an owner keyboard walkthrough.",
		AuthorityCeiling: ProposedAuthority{
			ReadWorkspace:  true,
			WriteWorkspace: true,
			ExecuteLocal:   true,
		},
		StopConditions: []string{"Stop before any remote effect."},
		Facets: []ContractFacet{{
			Kind:         ContractFacetSoftware,
			Summary:      "Desktop renderer accessibility",
			Requirements: []string{"Preserve mouse behavior and visible focus."},
		}},
		CreatedAt: time.Date(2026, 8, 26, 1, 1, 0, 0, time.UTC),
	}
	if err := proposal.Validate(); err != nil {
		t.Fatalf("valid proposal: %v", err)
	}

	proposal.Facets[0].Kind = "free_form_model_schema"
	if err := proposal.Validate(); err == nil {
		t.Fatal("proposal accepted an untyped adaptive facet")
	}
}

func TestIntakeConversationProvenanceReferencesIDsNotTranscriptContent(t *testing.T) {
	ref := IntakeConversationRef{
		EpisodeID: "episode-1",
		TurnID:    "turn-7",
		Position:  1,
	}
	if err := ref.Validate(); err != nil {
		t.Fatalf("valid conversation reference: %v", err)
	}
}

func TestResponsibilityLinkEndPreservesBothResponsibilities(t *testing.T) {
	link := ResponsibilityLink{
		ID:                   "rlink-1",
		SourceOpenLoopID:     "loop-1",
		DestinationOutcomeID: "outcome-1",
		Creator:              ResponsibilityLinkCreatorOwner,
		Reason:               "Continue the confirmed Home responsibility as bounded Project work.",
		CreatedAt:            time.Date(2026, 8, 26, 1, 2, 0, 0, time.UTC),
	}
	if err := link.Validate(); err != nil {
		t.Fatalf("valid responsibility link: %v", err)
	}

	endedAt := time.Date(2026, 8, 27, 1, 2, 0, 0, time.UTC)
	ended, err := link.End(ResponsibilityLinkCreatorOwner, "The lineage is no longer active.", endedAt)
	if err != nil {
		t.Fatalf("end responsibility link: %v", err)
	}
	if ended.SourceOpenLoopID != link.SourceOpenLoopID || ended.DestinationOutcomeID != link.DestinationOutcomeID {
		t.Fatalf("ending link mutated responsibility identity: before=%+v after=%+v", link, ended)
	}
	if link.EndedAt != nil || link.EndedReason != "" {
		t.Fatalf("End mutated original immutable value: %+v", link)
	}
	if ended.EndedAt == nil || !ended.EndedAt.Equal(endedAt) || ended.EndedReason == "" {
		t.Fatalf("ended lineage missing terminal receipt: %+v", ended)
	}
}
