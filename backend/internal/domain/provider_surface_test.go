package domain

import (
	"reflect"
	"testing"
)

func TestKennelActiveProviderSurfaceIsExactlyFiveHarnesses(t *testing.T) {
	want := []AgentHarness{
		HarnessCodex,
		HarnessClaudeCode,
		HarnessOpenCode,
		HarnessCursor,
		HarnessPi,
	}
	if !reflect.DeepEqual(AllHarnesses, want) {
		t.Fatalf("AllHarnesses = %#v, want %#v", AllHarnesses, want)
	}
	for _, harness := range want {
		if !harness.IsRecognizedPersisted() {
			t.Errorf("%s is not recognized", harness)
		}
		if !harness.IsSelectableForNewWork() {
			t.Errorf("%s is not admitted for worker execution", harness)
		}
	}
}

func TestKennelCoordinatorAdmissionMatchesStructuredChatCapability(t *testing.T) {
	for _, harness := range []AgentHarness{HarnessCodex, HarnessClaudeCode, HarnessOpenCode} {
		if !harness.IsSelectableAsCoordinator() {
			t.Errorf("%s must be coordinator-admitted", harness)
		}
	}
	for _, harness := range []AgentHarness{HarnessCursor, HarnessPi} {
		if harness.IsSelectableAsCoordinator() {
			t.Errorf("%s must stay worker-only until it has a structured coordinator driver", harness)
		}
	}
}

func TestKennelSwitchTargetsRequireProvenContinuation(t *testing.T) {
	for _, harness := range []AgentHarness{HarnessCodex, HarnessClaudeCode, HarnessOpenCode} {
		if !harness.IsSelectableAsSwitchTarget() {
			t.Errorf("%s must be switch-target-admitted", harness)
		}
	}
	for _, harness := range []AgentHarness{HarnessCursor, HarnessPi} {
		if harness.IsSelectableAsSwitchTarget() {
			t.Errorf("%s must not be switch-target-admitted without the full continuation contract", harness)
		}
	}
}

func TestKennelReviewerSurfaceMatchesActiveProviders(t *testing.T) {
	want := []ReviewerHarness{
		ReviewerCodex,
		ReviewerClaudeCode,
		ReviewerOpenCode,
		ReviewerCursor,
		ReviewerPi,
	}
	if !reflect.DeepEqual(AllReviewerHarnesses, want) {
		t.Fatalf("AllReviewerHarnesses = %#v, want %#v", AllReviewerHarnesses, want)
	}
	for _, harness := range want {
		if !harness.IsSelectableForNewWork() {
			t.Errorf("%s must be selectable for a new review", harness)
		}
	}
}
