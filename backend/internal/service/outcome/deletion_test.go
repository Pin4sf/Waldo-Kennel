package outcome

import (
	"context"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

type deletionServiceStore struct {
	ports.OutcomeStore
	preview ports.OutcomeDeletionPreview
	steps   []string
}

func (f *deletionServiceStore) PreviewOutcomeDeletion(context.Context, domain.OutcomeID) (ports.OutcomeDeletionPreview, error) {
	return f.preview, nil
}
func (f *deletionServiceStore) ListTrashedOutcomes(context.Context, domain.ProjectID) ([]ports.OutcomeTrashEntry, error) {
	return nil, nil
}
func (f *deletionServiceStore) ChangeOutcomeTrash(_ context.Context, _ domain.OutcomeID, _ int64, trash bool) error {
	f.steps = append(f.steps, "trash")
	f.preview.Trashed = trash
	return nil
}
func (f *deletionServiceStore) BeginOutcomePurge(context.Context, domain.OutcomeID, int64) error {
	f.steps = append(f.steps, "begin")
	f.preview.Erasing = true
	return nil
}
func (f *deletionServiceStore) PurgeOutcomeRecords(context.Context, domain.OutcomeID, int64) error {
	f.steps = append(f.steps, "purge")
	return nil
}
func TestPermanentDeletionRequiresConfirmationAndCurrentPreview(t *testing.T) {
	for _, tc := range []struct {
		name, word     string
		revision       int64
		blocked, allow bool
	}{{"missing", "", 2, false, false}, {"stale", "confirm", 1, false, false}, {"active", "confirm", 2, true, false}, {"explicit", "confirm", 2, false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			f := &deletionServiceStore{preview: ports.OutcomeDeletionPreview{OutcomeID: "out-1", Title: "Visible Outcome", Revision: 2}}
			if tc.blocked {
				f.preview.Blockers = []string{"unknown Attempt"}
			}
			s := &Service{store: f}
			err := s.ChangeOutcomeDeletion(context.Background(), "out-1", tc.revision, "permanent", tc.word)
			if tc.allow {
				if err != nil || len(f.steps) != 3 || f.steps[0] != "trash" || f.steps[1] != "begin" || f.steps[2] != "purge" {
					t.Fatalf("steps=%v err=%v", f.steps, err)
				}
			} else if err == nil || len(f.steps) > 0 {
				t.Fatalf("refusal mutated state: %v %v", f.steps, err)
			}
		})
	}
}
