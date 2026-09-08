package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestIntelligenceRunContractAnalysisMayExistBeforeOutcome(t *testing.T) {
	run := IntelligenceRun{
		ID:             IntelligenceRunID("intel-contract-1"),
		Kind:           IntelligenceRunContractAnalysis,
		ProjectID:      ProjectID("project-1"),
		IntakeID:       IntakeSessionID("intake-1"),
		SourceRevision: 0,
		Status:         IntelligenceRunRequested,
		InputDigest:    "input-digest",
		CreatedAt:      time.Now(),
	}
	if err := run.Validate(); err != nil {
		t.Fatalf("contract analysis before Outcome should be valid: %v", err)
	}
	if !run.OutcomeID.IsZero() {
		t.Fatalf("contract analysis unexpectedly requires Outcome: %q", run.OutcomeID)
	}
}

func TestIntelligenceRunPlanDraftRequiresExactContractRevision(t *testing.T) {
	run := IntelligenceRun{
		ID:             IntelligenceRunID("intel-plan-1"),
		Kind:           IntelligenceRunPlanDraft,
		ProjectID:      ProjectID("project-1"),
		OutcomeID:      OutcomeID("outcome-1"),
		SourceRevision: 2,
		Status:         IntelligenceRunRequested,
		InputDigest:    "input-digest",
		CreatedAt:      time.Now(),
	}
	if err := run.Validate(); err == nil {
		t.Fatal("plan draft without ContractRevisionID should be invalid")
	}

	run.ContractRevisionID = ContractRevisionID("contract-2")
	if err := run.Validate(); err != nil {
		t.Fatalf("plan draft with exact ContractRevision should be valid: %v", err)
	}
}

func TestIntelligenceRunOfflineManualMayOmitProviderAndModel(t *testing.T) {
	run := IntelligenceRun{
		ID:             IntelligenceRunID("intel-offline-1"),
		Kind:           IntelligenceRunContractAnalysis,
		ProjectID:      ProjectID("project-1"),
		IntakeID:       IntakeSessionID("intake-1"),
		SourceRevision: 1,
		Status:         IntelligenceRunFulfilled,
		InputDigest:    "input-digest",
		OutputDigest:   "output-digest",
		CreatedAt:      time.Now(),
	}
	if err := run.Validate(); err != nil {
		t.Fatalf("offline/manual intelligence run should be valid: %v", err)
	}
}

func TestIntelligenceRunExplicitModelRequiresProvider(t *testing.T) {
	run := IntelligenceRun{
		ID:             IntelligenceRunID("intel-model-1"),
		Kind:           IntelligenceRunContractAnalysis,
		ProjectID:      ProjectID("project-1"),
		IntakeID:       IntakeSessionID("intake-1"),
		SourceRevision: 1,
		ModelSelection: IntelligenceRunModelExplicit,
		Model:          "model-x",
		Status:         IntelligenceRunRequested,
		InputDigest:    "input-digest",
		CreatedAt:      time.Now(),
	}
	if err := run.Validate(); err == nil {
		t.Fatal("explicit intelligence model without provider should be invalid")
	}

	run.Provider = HarnessCodex
	if err := run.Validate(); err != nil {
		t.Fatalf("explicit model with provider should be valid: %v", err)
	}
}

func TestIntelligenceRunTerminalStatusCannotReturnToRunning(t *testing.T) {
	terminal := []IntelligenceRunStatus{
		IntelligenceRunFulfilled,
		IntelligenceRunFailed,
		IntelligenceRunCancelled,
		IntelligenceRunExpired,
	}
	for _, status := range terminal {
		run := IntelligenceRun{Status: status}
		if err := run.TransitionTo(IntelligenceRunRunning); err == nil {
			t.Fatalf("terminal status %q returned to running", status)
		}
	}
}

func TestIntelligenceRunNativeSessionReferenceIsProvenanceOnly(t *testing.T) {
	run := IntelligenceRun{
		ID:               IntelligenceRunID("intel-native-1"),
		Kind:             IntelligenceRunContractAnalysis,
		ProjectID:        ProjectID("project-1"),
		IntakeID:         IntakeSessionID("intake-1"),
		SourceRevision:   1,
		Provider:         HarnessCodex,
		ModelSelection:   IntelligenceRunModelProviderDefault,
		NativeSessionRef: "provider-native-thread-123",
		Status:           IntelligenceRunRunning,
		InputDigest:      "input-digest",
		CreatedAt:        time.Now(),
	}
	if err := run.Validate(); err != nil {
		t.Fatalf("provider provenance should not imply execution authority: %v", err)
	}

	typeOfRun := reflect.TypeOf(run)
	for _, forbidden := range []string{"AttemptID", "AgentSessionRef", "WorkUnitID", "CapabilityGrantID", "ExecutionBinding"} {
		if _, ok := typeOfRun.FieldByName(forbidden); ok {
			t.Fatalf("IntelligenceRun must not carry execution-authority field %s", forbidden)
		}
	}
}
