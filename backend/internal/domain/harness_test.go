package domain

import "testing"

func TestHarnessAdmissionPredicates(t *testing.T) {
	tests := []struct {
		name                 string
		harness              AgentHarness
		wantRecognized       bool
		wantSelectableForNew bool
	}{
		{name: "codex", harness: HarnessCodex, wantRecognized: true, wantSelectableForNew: true},
		{name: "deepseek harness", harness: HarnessDeepSeekHarness, wantRecognized: true, wantSelectableForNew: true},
		{name: "historical claude", harness: HarnessClaudeCode, wantRecognized: true, wantSelectableForNew: false},
		{name: "historical prime agent", harness: HarnessPrimeAgent, wantRecognized: true, wantSelectableForNew: false},
		{name: "historical fake", harness: HarnessFake, wantRecognized: true, wantSelectableForNew: false},
		{name: "unknown", harness: "unknown", wantRecognized: false, wantSelectableForNew: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.harness.IsRecognizedPersisted(); got != tt.wantRecognized {
				t.Fatalf("IsRecognizedPersisted() = %v, want %v", got, tt.wantRecognized)
			}
			if got := tt.harness.IsSelectableForNewWork(); got != tt.wantSelectableForNew {
				t.Fatalf("IsSelectableForNewWork() = %v, want %v", got, tt.wantSelectableForNew)
			}
		})
	}
}

func TestReviewerHarnessAdmissionPredicates(t *testing.T) {
	tests := []struct {
		name                 string
		harness              ReviewerHarness
		wantRecognized       bool
		wantSelectableForNew bool
	}{
		{name: "codex", harness: ReviewerCodex, wantRecognized: true, wantSelectableForNew: true},
		{name: "historical claude", harness: ReviewerClaudeCode, wantRecognized: true, wantSelectableForNew: false},
		{name: "historical cursor", harness: ReviewerCursor, wantRecognized: true, wantSelectableForNew: false},
		{name: "unknown", harness: "unknown", wantRecognized: false, wantSelectableForNew: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.harness.IsRecognizedPersisted(); got != tt.wantRecognized {
				t.Fatalf("IsRecognizedPersisted() = %v, want %v", got, tt.wantRecognized)
			}
			if got := tt.harness.IsSelectableForNewWork(); got != tt.wantSelectableForNew {
				t.Fatalf("IsSelectableForNewWork() = %v, want %v", got, tt.wantSelectableForNew)
			}
		})
	}
}

func TestDeepSeekHarnessIsKnown(t *testing.T) {
	if HarnessDeepSeekHarness != AgentHarness("deepseek-harness") {
		t.Fatalf("HarnessDeepSeekHarness = %q, want deepseek-harness", HarnessDeepSeekHarness)
	}
	if !HarnessDeepSeekHarness.IsKnown() {
		t.Fatal("HarnessDeepSeekHarness.IsKnown() = false, want true")
	}
	found := false
	for _, harness := range AllHarnesses {
		if harness == HarnessDeepSeekHarness {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("AllHarnesses does not contain HarnessDeepSeekHarness")
	}
}

func TestPrimeAgentHarnessIsKnown(t *testing.T) {
	if HarnessPrimeAgent != AgentHarness("prime-agent") {
		t.Fatalf("HarnessPrimeAgent = %q, want prime-agent", HarnessPrimeAgent)
	}
	if !HarnessPrimeAgent.IsKnown() {
		t.Fatal("HarnessPrimeAgent.IsKnown() = false, want true")
	}
	found := false
	for _, harness := range AllHarnesses {
		if harness == HarnessPrimeAgent {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("AllHarnesses does not contain HarnessPrimeAgent")
	}
}
func TestOMPHarnessIsKnown(t *testing.T) {
	if HarnessOMP != AgentHarness("omp") {
		t.Fatalf("HarnessOMP = %q, want omp", HarnessOMP)
	}
	if !HarnessOMP.IsKnown() {
		t.Fatal("HarnessOMP.IsKnown() = false, want true")
	}
	found := false
	for _, harness := range AllHarnesses {
		if harness == HarnessOMP {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("AllHarnesses does not contain HarnessOMP")
	}
}

func TestRoleGatedPredicatesExcludeDeepSeek(t *testing.T) {
	if HarnessDeepSeekHarness.IsSelectableAsCoordinator() {
		t.Fatal("deepseek-harness must not be coordinator-admitted yet")
	}
	if HarnessDeepSeekHarness.IsSelectableAsSwitchTarget() {
		t.Fatal("deepseek-harness must not be switch-target-admitted yet: no continuation identity")
	}
	if !HarnessCodex.IsSelectableAsCoordinator() || !HarnessCodex.IsSelectableAsSwitchTarget() {
		t.Fatal("codex must keep coordinator and switch-target admission")
	}
}
