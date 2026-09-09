package store

import (
	"context"
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
	UpdatedAt          time.Time
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
		DefaultSessionMode: domain.NormalizeSessionMode(row.DefaultSessionMode),
		ReasoningProvider:  row.ReasoningProvider,
		ReasoningModel:     row.ReasoningModel,
		ReasoningEffort:    row.ReasoningEffort,
		UpdatedAt:          row.UpdatedAt,
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
