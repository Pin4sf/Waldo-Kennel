package domain

import (
	"strings"
	"testing"
	"time"
)

func TestContractCriterionValidateBindsStableIdentityToRevision(t *testing.T) {
	criterion := ContractCriterion{
		ID:                 CriterionID("crit-1"),
		ContractRevisionID: ContractRevisionID("cr-1"),
		Position:           1,
		Text:               "A focus block can be recorded.",
	}
	if err := criterion.Validate(); err != nil {
		t.Fatalf("valid criterion rejected: %v", err)
	}

	for name, mutate := range map[string]func(*ContractCriterion){
		"missing id":       func(c *ContractCriterion) { c.ID = "" },
		"missing revision": func(c *ContractCriterion) { c.ContractRevisionID = "" },
		"zero position":    func(c *ContractCriterion) { c.Position = 0 },
		"blank text":       func(c *ContractCriterion) { c.Text = "  " },
	} {
		t.Run(name, func(t *testing.T) {
			got := criterion
			mutate(&got)
			if err := got.Validate(); err == nil {
				t.Fatalf("%s must be rejected", name)
			}
		})
	}
}

func TestEvidenceItemValidateRequiresExactCriterionAndSubjectRevision(t *testing.T) {
	now := time.Date(2026, 8, 25, 18, 0, 0, 0, time.UTC)
	evidence := EvidenceItem{
		ID:                 EvidenceItemID("ev-1"),
		OutcomeID:          OutcomeID("out-1"),
		ContractRevisionID: ContractRevisionID("cr-1"),
		CriterionID:        CriterionID("crit-1"),
		SubjectType:        ProofSubjectAttempt,
		SubjectID:          "att-1",
		SubjectRevision:    strings.Repeat("a", 64),
		Kind:               EvidenceSupporting,
		SourceType:         EvidenceSourceDeterministicCheck,
		SourceRef:          "go-test:focus-ledger",
		ProducerType:       EvidenceProducerTool,
		ProducerRef:        "go test ./...",
		Summary:            "Focus ledger deterministic checks passed.",
		ContentDigest:      strings.Repeat("b", 64),
		RequestKey:         "evidence-request-1",
		RequestFingerprint: strings.Repeat("c", 64),
		CreatedAt:          now,
	}
	if err := evidence.Validate(); err != nil {
		t.Fatalf("valid evidence rejected: %v", err)
	}

	for name, mutate := range map[string]func(*EvidenceItem){
		"missing criterion":        func(e *EvidenceItem) { e.CriterionID = "" },
		"missing subject revision": func(e *EvidenceItem) { e.SubjectRevision = "" },
		"invalid kind":             func(e *EvidenceItem) { e.Kind = "confidence" },
		"invalid source type":      func(e *EvidenceItem) { e.SourceType = "confidence_score" },
		"missing provenance":       func(e *EvidenceItem) { e.SourceRef = "" },
		"non-hex digest":           func(e *EvidenceItem) { e.ContentDigest = strings.Repeat("z", 64) },
	} {
		t.Run(name, func(t *testing.T) {
			got := evidence
			mutate(&got)
			if err := got.Validate(); err == nil {
				t.Fatalf("%s must be rejected", name)
			}
		})
	}
}

func TestVerificationRunValidateDeclaresActualIndependence(t *testing.T) {
	base := VerificationRun{
		ID:                 VerificationRunID("ver-1"),
		OutcomeID:          OutcomeID("out-1"),
		ContractRevisionID: ContractRevisionID("cr-1"),
		CriterionID:        CriterionID("crit-1"),
		SubjectType:        ProofSubjectAttempt,
		SubjectID:          "att-1",
		SubjectRevision:    strings.Repeat("a", 64),
		EvidenceItemIDs:    []EvidenceItemID{"ev-1"},
		Method:             "Read-only review against the frozen criterion.",
		IndependenceClass:  VerificationSeparateSession,
		Result:             VerificationPassed,
		ProducerRef:        "session-producer",
		VerifierRef:        "session-verifier",
		ProducerProvider:   "codex",
		VerifierProvider:   "codex",
		RequestKey:         "verify-request-1",
		RequestFingerprint: strings.Repeat("d", 64),
		CreatedAt:          time.Date(2026, 8, 25, 18, 1, 0, 0, time.UTC),
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid separate-session run rejected: %v", err)
	}

	t.Run("same session cannot claim separate-session review", func(t *testing.T) {
		got := base
		got.VerifierRef = got.ProducerRef
		if err := got.Validate(); err == nil || !strings.Contains(err.Error(), "different verifier") {
			t.Fatalf("same-session claim error = %v", err)
		}
	})

	t.Run("same provider cannot claim cross-provider review", func(t *testing.T) {
		got := base
		got.IndependenceClass = VerificationCrossProvider
		if err := got.Validate(); err == nil || !strings.Contains(err.Error(), "different provider") {
			t.Fatalf("same-provider claim error = %v", err)
		}
	})

	t.Run("producer self-check is explicitly non-independent", func(t *testing.T) {
		got := base
		got.IndependenceClass = VerificationProducerSelfCheck
		got.VerifierRef = got.ProducerRef
		if err := got.Validate(); err != nil {
			t.Fatalf("truthful producer self-check rejected: %v", err)
		}
		if got.IsIndependent() {
			t.Fatal("producer self-check must never derive independent")
		}
	})
}

func TestAcceptanceDecisionAndCorrectionPreserveExplicitUserLineage(t *testing.T) {
	now := time.Date(2026, 8, 25, 18, 2, 0, 0, time.UTC)
	decision := AcceptanceDecision{
		ID:                  AcceptanceDecisionID("acc-1"),
		OutcomeID:           OutcomeID("out-1"),
		ContractRevisionID:  ContractRevisionID("cr-1"),
		Kind:                AcceptanceRequestRework,
		ActorType:           AcceptanceActorUser,
		Summary:             "The restart behavior is still incomplete.",
		ResourceDisposition: ResourceDispositionRetain,
		RequestKey:          "acceptance-request-1",
		RequestFingerprint:  strings.Repeat("e", 64),
		CreatedAt:           now,
	}
	if err := decision.Validate(); err != nil {
		t.Fatalf("valid user decision rejected: %v", err)
	}
	provider := decision
	provider.ActorType = "provider"
	if err := provider.Validate(); err == nil {
		t.Fatal("a provider-authored AcceptanceDecision must be rejected")
	}

	correction := OutcomeCorrection{
		ID:                 OutcomeCorrectionID("corr-1"),
		DecisionID:         decision.ID,
		OutcomeID:          decision.OutcomeID,
		ContractRevisionID: decision.ContractRevisionID,
		Feedback:           decision.Summary,
		TargetType:         ReentryTargetAttempt,
		TargetID:           "att-1",
		CreatedAt:          now,
	}
	if err := correction.Validate(); err != nil {
		t.Fatalf("valid correction rejected: %v", err)
	}
	correction.TargetID = ""
	if err := correction.Validate(); err == nil {
		t.Fatal("correction without an explicit re-entry target must be rejected")
	}
}
