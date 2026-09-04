package registry

import (
	"reflect"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func TestHarnessedContainsOnlyKennelActiveProviders(t *testing.T) {
	got := make([]domain.AgentHarness, 0, len(Harnessed()))
	for _, item := range Harnessed() {
		got = append(got, item.Harness)
	}
	want := []domain.AgentHarness{
		domain.HarnessCodex,
		domain.HarnessClaudeCode,
		domain.HarnessOpenCode,
		domain.HarnessCursor,
		domain.HarnessPi,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Harnessed() = %#v, want %#v", got, want)
	}
}
