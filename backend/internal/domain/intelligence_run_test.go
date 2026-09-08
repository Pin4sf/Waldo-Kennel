package domain

import (
	"testing"
	"time"
)

func intelligenceDigest(label string) SHA256Digest { return DigestSHA256([]byte(label)) }

func TestIntelligenceRunContractAnalysisMayExistBeforeOutcome(t *testing.T) {
	run := IntelligenceRun{
		ID:             "intel-contract-1",
		Kind:           IntelligenceRunContractAnalysis,
		ProjectID:      "project-1",
		IntakeID:       "intake-1",
		SourceRevision: 0,
		Status:         IntelligenceRunRequested,
		InputDigest:    intelligenceDigest("contract input"),
		CreatedAt:      time.Now(),
	}
	if err := run.Validate(); err != nil {
		t.Fatalf("contract analysis before Outcome should be valid: %v", err)
	}
	if !run.OutcomeID.IsZero() {
		t.Fatalf("contract analysis unexpectedly requires Outcome: %q", run.OutcomeID)
	}
}

func TestIntelligenceRunLineageIsExclusiveByKind(t *testing.T) {
	base := func(kind IntelligenceRunKind) IntelligenceRun {
		return IntelligenceRun{
			ID:          "intel-lineage-1",
			Kind:        kind,
			ProjectID:   "project-1",
			Status:      IntelligenceRunRequested,
			InputDigest: intelligenceDigest("lineage input"),
			CreatedAt:   time.Now(),
		}
	}

	t.Run("contract analysis rejects outcome lineage", func(t *testing.T) {
		run := base(IntelligenceRunContractAnalysis)
		run.IntakeID = "intake-1"
		run.OutcomeID = "outcome-1"
		if err := run.Validate(); err == nil {
			t.Fatal("contract analysis accepted outcome lineage")
		}
	})

	t.Run("contract analysis rejects contract lineage", func(t *testing.T) {
		run := base(IntelligenceRunContractAnalysis)
		run.IntakeID = "intake-1"
		run.ContractRevisionID = "contract-1"
		if err := run.Validate(); err == nil {
			t.Fatal("contract analysis accepted contract lineage")
		}
	})

	t.Run("plan draft rejects intake lineage", func(t *testing.T) {
		run := base(IntelligenceRunPlanDraft)
		run.IntakeID = "intake-1"
		run.OutcomeID = "outcome-1"
		run.ContractRevisionID = "contract-1"
		run.SourceRevision = 1
		if err := run.Validate(); err == nil {
			t.Fatal("plan draft accepted intake lineage")
		}
	})
}

func TestIntelligenceRunPlanDraftRequiresExactContractRevision(t *testing.T) {
	run := IntelligenceRun{
		ID:             "intel-plan-1",
		Kind:           IntelligenceRunPlanDraft,
		ProjectID:      "project-1",
		OutcomeID:      "outcome-1",
		SourceRevision: 2,
		Status:         IntelligenceRunRequested,
		InputDigest:    intelligenceDigest("plan input"),
		CreatedAt:      time.Now(),
	}
	if err := run.Validate(); err == nil {
		t.Fatal("plan draft without ContractRevisionID should be invalid")
	}

	run.ContractRevisionID = "contract-2"
	if err := run.Validate(); err != nil {
		t.Fatalf("plan draft with exact ContractRevision should be valid: %v", err)
	}
}

func TestIntelligenceRunOfflineManualMayOmitProviderAndModel(t *testing.T) {
	completed := time.Now().UTC()
	run := IntelligenceRun{
		ID:             "intel-offline-1",
		Kind:           IntelligenceRunContractAnalysis,
		ProjectID:      "project-1",
		IntakeID:       "intake-1",
		SourceRevision: 1,
		Status:         IntelligenceRunFulfilled,
		InputDigest:    intelligenceDigest("offline input"),
		OutputDigest:   intelligenceDigest("offline output"),
		CreatedAt:      completed.Add(-time.Second),
		CompletedAt:    &completed,
	}
	if err := run.Validate(); err != nil {
		t.Fatalf("offline/manual intelligence run should be valid: %v", err)
	}
}

func TestIntelligenceRunProviderIdentityIsOpaqueAndIndependentFromHarness(t *testing.T) {
	run := IntelligenceRun{
		ID:                "intel-api-1",
		Kind:              IntelligenceRunContractAnalysis,
		ProjectID:         "project-1",
		IntakeID:          "intake-1",
		SourceRevision:    1,
		RequestedProvider: IntelligenceProviderID("waldo-hosted-reasoner"),
		RequestedModel:    "planner-v2",
		EffectiveProvider: IntelligenceProviderID("direct-api.example/v1"),
		EffectiveModel:    "planner-2026-09",
		NativeSessionRef:  "request-123",
		Status:            IntelligenceRunRunning,
		InputDigest:       intelligenceDigest("api input"),
		CreatedAt:         time.Now(),
	}
	if err := run.Validate(); err != nil {
		t.Fatalf("opaque intelligence provider should not require AgentHarness registration: %v", err)
	}
}

func TestIntelligenceRunModelProvenanceRequiresItsProvider(t *testing.T) {
	run := IntelligenceRun{
		ID:             "intel-model-1",
		Kind:           IntelligenceRunContractAnalysis,
		ProjectID:      "project-1",
		IntakeID:       "intake-1",
		SourceRevision: 1,
		RequestedModel: "model-x",
		Status:         IntelligenceRunRequested,
		InputDigest:    intelligenceDigest("model input"),
		CreatedAt:      time.Now(),
	}
	if err := run.Validate(); err == nil {
		t.Fatal("requested intelligence model without requested provider should be invalid")
	}

	run.RequestedProvider = "provider-x"
	if err := run.Validate(); err != nil {
		t.Fatalf("requested model with provider should be valid: %v", err)
	}
}

func TestIntelligenceRunRequiresSHA256Digests(t *testing.T) {
	run := IntelligenceRun{
		ID:          "intel-digest-1",
		Kind:        IntelligenceRunContractAnalysis,
		ProjectID:   "project-1",
		IntakeID:    "intake-1",
		Status:      IntelligenceRunRequested,
		InputDigest: SHA256Digest("not-a-digest"),
		CreatedAt:   time.Now(),
	}
	if err := run.Validate(); err == nil {
		t.Fatal("arbitrary input digest string was accepted")
	}
}

func TestIntelligenceRunCompletionTimestampMatchesTerminalState(t *testing.T) {
	now := time.Now().UTC()
	run := IntelligenceRun{
		ID: "intel-time-1", Kind: IntelligenceRunContractAnalysis, ProjectID: "project-1", IntakeID: "intake-1",
		Status: IntelligenceRunRunning, InputDigest: intelligenceDigest("time input"), CreatedAt: now.Add(-time.Second),
		CompletedAt: &now,
	}
	if err := run.Validate(); err == nil {
		t.Fatal("non-terminal run accepted a completion timestamp")
	}

	run.Status = IntelligenceRunFailed
	run.CompletedAt = nil
	if err := run.Validate(); err == nil {
		t.Fatal("terminal run without completion timestamp was accepted")
	}
}

func TestIntelligenceRunTransitionUsesControlPlaneCompletionTime(t *testing.T) {
	run := IntelligenceRun{Status: IntelligenceRunRunning}
	completed := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	if err := run.TransitionTo(IntelligenceRunFailed, &completed); err != nil {
		t.Fatalf("transition failed: %v", err)
	}
	if run.CompletedAt == nil || !run.CompletedAt.Equal(completed) {
		t.Fatalf("completion time = %v, want %v", run.CompletedAt, completed)
	}
	if err := run.TransitionTo(IntelligenceRunRunning, nil); err == nil {
		t.Fatal("terminal status returned to running")
	}
}

func TestIntelligenceRunTerminalReplayCannotChangeCompletionTime(t *testing.T) {
	first := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	second := first.Add(time.Second)
	run := IntelligenceRun{Status: IntelligenceRunFailed, CompletedAt: &first}
	if err := run.TransitionTo(IntelligenceRunFailed, &second); err == nil {
		t.Fatal("same terminal status changed completion time")
	}
	if err := run.TransitionTo(IntelligenceRunFailed, &first); err != nil {
		t.Fatalf("identical terminal replay should be idempotent: %v", err)
	}
}
