package domain

import "testing"

func TestHarnessAdmissionPredicates(t *testing.T) {
	tests := []struct {
		name            string
		harness         AgentHarness
		worker          bool
		coordinator     bool
		switchTarget    bool
	}{
		{name: "codex", harness: HarnessCodex, worker: true, coordinator: true, switchTarget: true},
		{name: "claude-code", harness: HarnessClaudeCode, worker: true, coordinator: true, switchTarget: true},
		{name: "opencode", harness: HarnessOpenCode, worker: true, coordinator: true, switchTarget: true},
		{name: "cursor", harness: HarnessCursor, worker: true},
		{name: "pi", harness: HarnessPi, worker: true},
		{name: "unknown", harness: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.harness.IsSelectableForNewWork(); got != tt.worker {
				t.Fatalf("IsSelectableForNewWork() = %v, want %v", got, tt.worker)
			}
			if got := tt.harness.IsSelectableAsCoordinator(); got != tt.coordinator {
				t.Fatalf("IsSelectableAsCoordinator() = %v, want %v", got, tt.coordinator)
			}
			if got := tt.harness.IsSelectableAsSwitchTarget(); got != tt.switchTarget {
				t.Fatalf("IsSelectableAsSwitchTarget() = %v, want %v", got, tt.switchTarget)
			}
		})
	}
}

func TestOnlyActiveProvidersAreRecognized(t *testing.T) {
	for _, harness := range AllHarnesses {
		if !harness.IsKnown() || !harness.IsRecognizedPersisted() {
			t.Fatalf("active provider %q is not recognized", harness)
		}
	}
	for _, retired := range []AgentHarness{"deepseek-harness", "prime-agent", "omp", "goose", "aider"} {
		if retired.IsKnown() || retired.IsRecognizedPersisted() || retired.IsSelectableForNewWork() {
			t.Fatalf("retired provider %q leaked back into the active domain vocabulary", retired)
		}
	}
}

func TestFakeHarnessIsFixtureOnly(t *testing.T) {
	if !HarnessFake.IsRecognizedPersisted() {
		t.Fatal("fake harness must remain usable by focused test fixtures")
	}
	if HarnessFake.IsSelectableForNewWork() || HarnessFake.IsSelectableAsCoordinator() || HarnessFake.IsSelectableAsSwitchTarget() {
		t.Fatal("fake harness must never be admitted to product execution")
	}
}

func TestReviewerHarnessAdmissionPredicates(t *testing.T) {
	for _, harness := range AllReviewerHarnesses {
		if !harness.IsKnown() || !harness.IsSelectableForNewWork() {
			t.Fatalf("active reviewer %q is not admitted", harness)
		}
	}
	for _, retired := range []ReviewerHarness{"goose", "muse", "deepseek-harness"} {
		if retired.IsKnown() || retired.IsSelectableForNewWork() {
			t.Fatalf("retired reviewer %q leaked back into the product surface", retired)
		}
	}
}
