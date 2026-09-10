package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
)

// Daemon-owned user preferences.
//
// The row is seeded by migration, so a read is a plain SELECT and no caller has
// to handle "settings do not exist yet".

// AppSettings is the durable preference set. Field-compatible with
// service/settings.Snapshot, which the daemon wiring adapts.
type AppSettings struct {
	// DefaultSessionMode is the interface a new session gets when the spawn does
	// not name one. Never applied to an existing session: only an explicit
	// interface transition changes a live session's committed mode, so
	// changing this only affects sessions created afterwards.
	DefaultSessionMode domain.SessionMode
	ReasoningProvider  string
	ReasoningModel     string
	ReasoningEffort    string
	// ReasoningVerifiedAt is when an actual probe last succeeded, and the
	// provider/model pair it succeeded for. Nil means never verified: a stored
	// credential alone proves nothing about whether reasoning works.
	ReasoningVerifiedAt              *time.Time
	ReasoningVerifiedProvider        string
	ReasoningVerifiedModel           string
	ReasoningGeneration              int64
	ReasoningVerifiedGeneration      int64
	ReasoningVerificationFingerprint string
	UpdatedAt                        time.Time
}

// GetAppSettings reads the preference row.
func (s *Store) GetAppSettings(ctx context.Context) (AppSettings, error) {
	row, err := s.qr.GetAppSettings(ctx)
	if err != nil {
		return AppSettings{}, fmt.Errorf("read app settings: %w", err)
	}
	return AppSettings{
		// Normalized on read: a value written by a build that knows a mode this
		// one does not must still resolve to something dispatchable.
		DefaultSessionMode:               domain.NormalizeSessionMode(row.DefaultSessionMode),
		ReasoningProvider:                row.ReasoningProvider,
		ReasoningModel:                   row.ReasoningModel,
		ReasoningEffort:                  row.ReasoningEffort,
		ReasoningVerifiedAt:              parseVerifiedAt(row.ReasoningVerifiedAt),
		ReasoningVerifiedProvider:        row.ReasoningVerifiedProvider,
		ReasoningVerifiedModel:           row.ReasoningVerifiedModel,
		ReasoningGeneration:              row.ReasoningGeneration,
		ReasoningVerifiedGeneration:      row.ReasoningVerifiedGeneration,
		ReasoningVerificationFingerprint: row.ReasoningVerificationFingerprint,
		UpdatedAt:                        row.UpdatedAt,
	}, nil
}

// SetReasoningSettings stores only non-secret reasoning preferences. The
// credential is owned by the daemon secret store, never this SQLite row.
func (s *Store) SetReasoningSettings(ctx context.Context, provider, model, effort string, now time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.qw.SetReasoningSettings(ctx, gen.SetReasoningSettingsParams{
		ReasoningProvider: provider, ReasoningModel: model, ReasoningEffort: effort, UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("set reasoning settings: %w", err)
	}
	return nil
}

// SetDefaultSessionMode persists the default interface for new sessions.
func (s *Store) SetDefaultSessionMode(ctx context.Context, mode domain.SessionMode, now time.Time) error {
	if !mode.Valid() {
		return fmt.Errorf("invalid session mode %q", mode)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.qw.SetDefaultSessionMode(ctx, gen.SetDefaultSessionModeParams{
		DefaultSessionMode: mode,
		UpdatedAt:          now,
	}); err != nil {
		return fmt.Errorf("set default session mode: %w", err)
	}
	return nil
}

// SetReasoningVerification records the outcome of an actual reasoning probe.
//
// A nil time clears the record, which is what a provider or model change must
// do: verification belongs to the exact pair that was probed and cannot be
// inherited by another one.
func (s *Store) SetReasoningVerification(ctx context.Context, verifiedAt *time.Time, provider, model string, now time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	stamp := sql.NullString{}
	if verifiedAt != nil {
		stamp = sql.NullString{String: verifiedAt.UTC().Format(time.RFC3339Nano), Valid: true}
	}
	if err := s.qw.SetReasoningVerification(ctx, gen.SetReasoningVerificationParams{
		ReasoningVerifiedAt:       stamp,
		ReasoningVerifiedProvider: provider,
		ReasoningVerifiedModel:    model,
		UpdatedAt:                 now,
	}); err != nil {
		return fmt.Errorf("set reasoning verification: %w", err)
	}
	return nil
}

// SetReasoningVerificationForGeneration conditionally applies a probe result
// only while the persisted reasoning configuration generation is unchanged.
func (s *Store) SetReasoningVerificationForGeneration(ctx context.Context, verifiedAt *time.Time, provider, model string, generation int64, fingerprint string, now time.Time) (bool, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	stamp := sql.NullString{}
	if verifiedAt != nil {
		stamp = sql.NullString{String: verifiedAt.UTC().Format(time.RFC3339Nano), Valid: true}
	}
	clear := verifiedAt == nil
	var generationArg interface{} = generation
	var fingerprintArg interface{} = fingerprint
	if clear {
		generationArg = nil
		fingerprintArg = nil
	}
	rows, err := s.qw.SetReasoningVerificationForGeneration(ctx, gen.SetReasoningVerificationForGenerationParams{
		ReasoningVerifiedAt: stamp, ReasoningVerifiedProvider: provider, ReasoningVerifiedModel: model,
		Column4: generationArg, ReasoningVerifiedGeneration: generation,
		Column6: fingerprintArg, ReasoningVerificationFingerprint: fingerprint,
		UpdatedAt: now, ReasoningGeneration: generation,
	})
	if err != nil {
		return false, fmt.Errorf("set reasoning verification for generation %d: %w", generation, err)
	}
	return rows == 1, nil
}

// parseVerifiedAt tolerates an unreadable stamp by reporting "never verified".
// Treating a corrupt value as a successful verification would be the unsafe
// direction of that failure.
func parseVerifiedAt(raw sql.NullString) *time.Time {
	if !raw.Valid || raw.String == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw.String)
	if err != nil {
		return nil
	}
	return &parsed
}
