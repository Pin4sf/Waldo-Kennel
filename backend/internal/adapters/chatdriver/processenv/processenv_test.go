package processenv

import (
	"slices"
	"strings"
	"testing"
)

func TestMergeInheritsDaemonEnvironmentAndAppliesOverlay(t *testing.T) {
	t.Setenv("KENNEL_PROCESSENV_INHERITED", "parent")
	t.Setenv("KENNEL_PROCESSENV_REPLACED", "old")

	got := Merge(map[string]string{
		"KENNEL_PROCESSENV_REPLACED": "new",
		"KENNEL_PROCESSENV_SESSION":  "session",
	})
	if !slices.IsSorted(got) {
		t.Fatalf("environment is not sorted: %v", got)
	}
	want := map[string]string{
		"KENNEL_PROCESSENV_INHERITED": "parent",
		"KENNEL_PROCESSENV_REPLACED":  "new",
		"KENNEL_PROCESSENV_SESSION":   "session",
	}
	for _, entry := range got {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			if expected, exists := want[key]; exists {
				if value != expected {
					t.Fatalf("%s = %q, want %q", key, value, expected)
				}
				delete(want, key)
			}
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing environment values: %v", want)
	}
}
