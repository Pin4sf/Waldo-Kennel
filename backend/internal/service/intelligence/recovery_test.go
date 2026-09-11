package intelligence

import (
	"context"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

type recoveryStore struct {
	runs    []domain.IntelligenceRun
	updates []domain.IntelligenceRunStatus
}

func (s *recoveryStore) CreateIntelligenceRun(context.Context, domain.IntelligenceRun) error {
	return nil
}
func (s *recoveryStore) GetIntelligenceRun(context.Context, domain.IntelligenceRunID) (domain.IntelligenceRun, bool, error) {
	return domain.IntelligenceRun{}, false, nil
}
func (s *recoveryStore) ListNonTerminalIntelligenceRuns(context.Context) ([]domain.IntelligenceRun, error) {
	return s.runs, nil
}
func (s *recoveryStore) RecordIntelligenceRunEffectiveProvenance(context.Context, domain.IntelligenceRunID, domain.IntelligenceProviderID, string, string) error {
	return nil
}
func (s *recoveryStore) RecordIntelligenceRunMetrics(context.Context, domain.IntelligenceRunID, *int64, *int64, *int64) error {
	return nil
}
func (s *recoveryStore) UpdateIntelligenceRunStatus(_ context.Context, _ domain.IntelligenceRunID, status domain.IntelligenceRunStatus, _ domain.SHA256Digest, _ string, _ string, _ *time.Time) error {
	s.updates = append(s.updates, status)
	return nil
}

var _ ports.IntelligenceRunStore = (*recoveryStore)(nil)

func TestReconcileInterruptedRunsExpiresWithoutRetryingProvider(t *testing.T) {
	store := &recoveryStore{runs: []domain.IntelligenceRun{{ID: "intel-1", Status: domain.IntelligenceRunRunning}, {ID: "intel-2", Status: domain.IntelligenceRunRequested}}}
	count, err := ReconcileInterruptedRuns(context.Background(), store, func() time.Time { return time.Unix(10, 0).UTC() })
	if err != nil || count != 2 {
		t.Fatalf("reconcile = %d, %v; want two", count, err)
	}
	for _, status := range store.updates {
		if status != domain.IntelligenceRunExpired {
			t.Fatalf("status = %q, want expired", status)
		}
	}
}
