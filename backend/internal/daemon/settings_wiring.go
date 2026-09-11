package daemon

import (
	"context"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	settingssvc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/settings"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
)

// settingsStore adapts the SQLite store to the settings service's Store.
//
// The two define their own snapshot types so neither depends on the other's; this
// is the one place that knows both, keeping the translation in the wiring.
type settingsStore struct{ store *sqlite.Store }

var _ settingssvc.Store = settingsStore{}
var _ settingssvc.VerificationGenerationStore = settingsStore{}

func (s settingsStore) GetAppSettings(ctx context.Context) (settingssvc.Snapshot, error) {
	row, err := s.store.GetAppSettings(ctx)
	if err != nil {
		return settingssvc.Snapshot{}, err
	}
	return settingssvc.Snapshot{
		DefaultSessionMode:               row.DefaultSessionMode,
		ReasoningProvider:                row.ReasoningProvider,
		ReasoningModel:                   row.ReasoningModel,
		ReasoningEffort:                  row.ReasoningEffort,
		ReasoningVerifiedAt:              row.ReasoningVerifiedAt,
		ReasoningVerifiedProvider:        row.ReasoningVerifiedProvider,
		ReasoningVerifiedModel:           row.ReasoningVerifiedModel,
		ReasoningGeneration:              row.ReasoningGeneration,
		ReasoningVerifiedGeneration:      row.ReasoningVerifiedGeneration,
		ReasoningVerificationFingerprint: row.ReasoningVerificationFingerprint,
		UpdatedAt:                        row.UpdatedAt,
	}, nil
}

func (s settingsStore) SetReasoningSettings(ctx context.Context, provider, model, effort string, now time.Time) error {
	return s.store.SetReasoningSettings(ctx, provider, model, effort, now)
}

func (s settingsStore) SetReasoningVerification(
	ctx context.Context,
	verifiedAt *time.Time,
	provider, model string,
	now time.Time,
) error {
	return s.store.SetReasoningVerification(ctx, verifiedAt, provider, model, now)
}

// Forward the generation-fenced write so successful probes retain the same
// configuration binding that GetAppSettings returns to the settings service.
func (s settingsStore) SetReasoningVerificationForGeneration(
	ctx context.Context, verifiedAt *time.Time, provider, model string,
	generation int64, fingerprint string, now time.Time,
) (bool, error) {
	return s.store.SetReasoningVerificationForGeneration(ctx, verifiedAt, provider, model, generation, fingerprint, now)
}

func (s settingsStore) SetDefaultSessionMode(
	ctx context.Context,
	mode domain.SessionMode,
	now time.Time,
) error {
	return s.store.SetDefaultSessionMode(ctx, mode, now)
}
