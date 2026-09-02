package domain

import "testing"

func TestHarnessAdmissionPredicates(t *testing.T) {
	for _, harness := range []AgentHarness{HarnessCodex, HarnessClaudeCode, HarnessOpenCode, HarnessCursor, HarnessPi} {
		t.Run(string(harness), func(t *testing.T) {
			if !harness.IsRecognizedPersisted() {
				t.Fatalf("provider %q should be recognized", harness)
			}
			if !harness.IsSelectableForNewWork() {
				t.Fatalf("provider %q should be selectable for new work", harness)
			}
		})
	}
	if HarnessFake.IsSelectableForNewWork() {
		t.Fatal("fake harness must never be selectable for product work")
	}
	if AgentHarness("aider").IsRecognizedPersisted() || AgentHarness("aider").IsSelectableForNewWork() {
		t.Fatal("removed providers must not remain in the Kennel vocabulary")
	}
	if AgentHarness("unknown").IsKnown() {
		t.Fatal("unknown provider should not be recognized")
	}
}

func TestReviewerHarnessAdmissionPredicates(t *testing.T) {
	for _, harness := range []ReviewerHarness{ReviewerCodex, ReviewerClaudeCode, ReviewerOpenCode, ReviewerCursor, ReviewerPi} {
		t.Run(string(harness), func(t *testing.T) {
			if !harness.IsRecognizedPersisted() || !harness.IsSelectableForNewWork() {
				t.Fatalf("review provider %q should be active", harness)
			}
		})
	}
	if ReviewerHarness("copilot").IsKnown() {
		t.Fatal("removed reviewer providers must not remain recognized")
	}
}

func TestAllHarnessesIsExactlyKennelCore(t *testing.T) {
	want := []AgentHarness{HarnessCodex, HarnessClaudeCode, HarnessOpenCode, HarnessCursor, HarnessPi}
	if len(AllHarnesses) != len(want) {
		t.Fatalf("AllHarnesses len = %d, want %d: %#v", len(AllHarnesses), len(want), AllHarnesses)
	}
	for i := range want {
		if AllHarnesses[i] != want[i] {
			t.Fatalf("AllHarnesses[%d] = %q, want %q", i, AllHarnesses[i], want[i])
		}
	}
}
