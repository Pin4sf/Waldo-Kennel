package domain

import "testing"

func TestAllHarnessesContainsOnlySupportedHarnesses(t *testing.T) {
	want := map[AgentHarness]bool{
		HarnessClaudeCode: true,
		HarnessCodex:      true,
		HarnessOpenCode:   true,
		HarnessCursor:     true,
	}
	if len(AllHarnesses) != len(want) {
		t.Fatalf("AllHarnesses = %#v, want exactly %#v", AllHarnesses, want)
	}
	for _, harness := range AllHarnesses {
		if !want[harness] {
			t.Fatalf("unsupported selectable harness %q", harness)
		}
	}
	if HarnessPrimeAgent.IsKnown() || HarnessOMP.IsKnown() {
		t.Fatal("retired harnesses must not be selectable")
	}
}
