package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
)

var _ ports.AttemptCheckRunStore = (*Store)(nil)

// ReserveAttemptCheckRun claims the right to invoke one check.
//
// The unique (attempt, check, artifact version) index is what makes this a
// reservation rather than a hint: two reconcilers racing produce one insert
// and one ErrCheckRunAlreadyReserved, so the command is launched once.
func (s *Store) ReserveAttemptCheckRun(ctx context.Context, run ports.AttemptCheckRun) error {
	if run.AttemptID.IsZero() || strings.TrimSpace(string(run.CheckID)) == "" || strings.TrimSpace(run.ArtifactVersion) == "" {
		return fmt.Errorf("check run reservation requires attempt, check and artifact identity")
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	err := s.qw.ReserveAttemptCheckRun(ctx, gen.ReserveAttemptCheckRunParams{
		ID: run.ID, AttemptID: string(run.AttemptID), CheckID: string(run.CheckID),
		ArtifactVersion: run.ArtifactVersion, ReservationEpoch: run.ReservationEpoch, ReservedAt: run.ReservedAt.UTC(),
	})
	if err == nil {
		return nil
	}
	// A losing insert means the row already exists, which is precisely the
	// signal the caller needs: somebody else owns this invocation.
	if _, found, getErr := s.getCheckRunLocked(ctx, run.AttemptID, run.CheckID, run.ArtifactVersion); getErr == nil && found {
		return ports.ErrCheckRunAlreadyReserved
	}
	return fmt.Errorf("reserve check run %s/%s: %w", run.AttemptID, run.CheckID, err)
}

// GetAttemptCheckRun reads one durable check run.
func (s *Store) GetAttemptCheckRun(ctx context.Context, attemptID domain.AttemptID, checkID domain.ApprovedCheckID, artifactVersion string) (ports.AttemptCheckRun, bool, error) {
	row, err := s.qr.GetAttemptCheckRun(ctx, gen.GetAttemptCheckRunParams{
		AttemptID: string(attemptID), CheckID: string(checkID), ArtifactVersion: artifactVersion,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return ports.AttemptCheckRun{}, false, nil
	}
	if err != nil {
		return ports.AttemptCheckRun{}, false, fmt.Errorf("read check run %s/%s: %w", attemptID, checkID, err)
	}
	return checkRunFromGetRow(row), true, nil
}

// ListAttemptCheckRuns returns every recorded run for one retained artifact.
func (s *Store) ListAttemptCheckRuns(ctx context.Context, attemptID domain.AttemptID, artifactVersion string) ([]ports.AttemptCheckRun, error) {
	rows, err := s.qr.ListAttemptCheckRuns(ctx, gen.ListAttemptCheckRunsParams{
		AttemptID: string(attemptID), ArtifactVersion: artifactVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("list check runs for %s: %w", attemptID, err)
	}
	runs := make([]ports.AttemptCheckRun, 0, len(rows))
	for _, row := range rows {
		runs = append(runs, checkRunFromListRow(row))
	}
	return runs, nil
}

// RecordAttemptCheckObservation completes a reservation exactly once. A
// second call changes nothing: the observation is what later proof is
// rebuilt from, so it must not move.
func (s *Store) RecordAttemptCheckObservation(ctx context.Context, run ports.AttemptCheckRun) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	observedAt := run.Observation.EndedAt
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	if _, err := s.qw.RecordAttemptCheckObservation(ctx, gen.RecordAttemptCheckObservationParams{
		Ran: boolToInt(run.Observation.Ran), Passed: boolToInt(run.Observation.Passed),
		ExitCode: int64(run.Observation.ExitCode), EnforcedBy: run.Observation.EnforcedBy,
		TimedOut: boolToInt(run.Observation.TimedOut), Cancelled: boolToInt(run.Observation.Cancelled),
		TerminationUnknown: boolToInt(run.Observation.TerminationUnknown),
		OutputTruncated:    boolToInt(run.Observation.OutputTruncated),
		Output:             run.Observation.Output, Unavailable: run.Observation.Unavailable,
		BaselineRan:     boolToInt(run.Observation.BaselineRan),
		BaselinePassed:  boolToInt(run.Observation.BaselinePassed),
		BaselineDetail:  run.Observation.BaselineDetail,
		ArtifactChanged: boolToInt(run.ArtifactChanged), ObservedArtifactVersion: run.ObservedArtifactVersion,
		ObservedAt: sql.NullTime{Time: observedAt.UTC(), Valid: true},
		AttemptID:  string(run.AttemptID), CheckID: string(run.CheckID), ArtifactVersion: run.ArtifactVersion,
	}); err != nil {
		return fmt.Errorf("record check observation %s/%s: %w", run.AttemptID, run.CheckID, err)
	}
	return nil
}

// MarkAttemptCheckRunUnknown records a reservation that never completed.
func (s *Store) MarkAttemptCheckRunUnknown(ctx context.Context, attemptID domain.AttemptID, checkID domain.ApprovedCheckID, artifactVersion string, at time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, err := s.qw.MarkAttemptCheckRunUnknown(ctx, gen.MarkAttemptCheckRunUnknownParams{
		ObservedAt: sql.NullTime{Time: at.UTC(), Valid: true},
		AttemptID:  string(attemptID), CheckID: string(checkID), ArtifactVersion: artifactVersion,
	}); err != nil {
		return fmt.Errorf("mark check run unknown %s/%s: %w", attemptID, checkID, err)
	}
	return nil
}

func checkRunFromGetRow(row gen.GetAttemptCheckRunRow) ports.AttemptCheckRun {
	run := checkRunFromValues(row.ID, row.AttemptID, row.CheckID, row.ArtifactVersion, row.State, row.ReservationEpoch, row.Ran, row.Passed, row.ExitCode, row.EnforcedBy, row.TimedOut, row.Cancelled, row.TerminationUnknown, row.OutputTruncated, row.Output, row.Unavailable, row.ArtifactChanged, row.ObservedArtifactVersion, row.ReservedAt, row.ObservedAt)
	return withBaseline(run, row.BaselineRan, row.BaselinePassed, row.BaselineDetail)
}

func checkRunFromListRow(row gen.ListAttemptCheckRunsRow) ports.AttemptCheckRun {
	run := checkRunFromValues(row.ID, row.AttemptID, row.CheckID, row.ArtifactVersion, row.State, row.ReservationEpoch, row.Ran, row.Passed, row.ExitCode, row.EnforcedBy, row.TimedOut, row.Cancelled, row.TerminationUnknown, row.OutputTruncated, row.Output, row.Unavailable, row.ArtifactChanged, row.ObservedArtifactVersion, row.ReservedAt, row.ObservedAt)
	return withBaseline(run, row.BaselineRan, row.BaselinePassed, row.BaselineDetail)
}

// withBaseline attaches the known-wrong baseline to a stored run.
//
// It is applied after the positional mapping rather than threaded through it:
// that signature is already at the limit of what is readable, and the baseline
// is exactly the kind of field that must not be silently dropped by landing in
// the wrong position. Historical rows default to no baseline, which is the
// truthful answer — nothing was established for them.
func withBaseline(run ports.AttemptCheckRun, ran, passed int64, detail string) ports.AttemptCheckRun {
	run.Observation.BaselineRan = ran == 1
	run.Observation.BaselinePassed = passed == 1
	run.Observation.BaselineDetail = detail
	return run
}

func checkRunFromValues(id, attemptID, checkID, artifactVersion, state, reservationEpoch string, ran, passed, exitCode int64, enforcedBy string, timedOut, cancelled, terminationUnknown, outputTruncated int64, output, unavailable string, artifactChanged int64, observedArtifactVersion string, reservedAt time.Time, observedAt sql.NullTime) ports.AttemptCheckRun {
	run := ports.AttemptCheckRun{
		ID: id, AttemptID: domain.AttemptID(attemptID),
		CheckID: domain.ApprovedCheckID(checkID), ArtifactVersion: artifactVersion,
		State: ports.CheckRunState(state), ReservationEpoch: reservationEpoch,
		Observation: ports.AttemptCheckObservation{
			ArtifactVersion: artifactVersion,
			EnforcedBy:      enforcedBy, Ran: ran == 1, ExitCode: int(exitCode),
			Passed: passed == 1, TimedOut: timedOut == 1, Cancelled: cancelled == 1,
			TerminationUnknown: terminationUnknown == 1, OutputTruncated: outputTruncated == 1,
			Output: output, Unavailable: unavailable, StartedAt: reservedAt,
		},
		ArtifactChanged: artifactChanged == 1, ObservedArtifactVersion: observedArtifactVersion,
		ReservedAt: reservedAt,
	}
	if observedAt.Valid {
		at := observedAt.Time
		run.ObservedAt = &at
		run.Observation.EndedAt = at
	}
	return run
}

// getCheckRunLocked reads a run without taking the write lock again.
func (s *Store) getCheckRunLocked(ctx context.Context, attemptID domain.AttemptID, checkID domain.ApprovedCheckID, artifactVersion string) (ports.AttemptCheckRun, bool, error) {
	row, err := s.qr.GetAttemptCheckRun(ctx, gen.GetAttemptCheckRunParams{
		AttemptID: string(attemptID), CheckID: string(checkID), ArtifactVersion: artifactVersion,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return ports.AttemptCheckRun{}, false, nil
	}
	if err != nil {
		return ports.AttemptCheckRun{}, false, err
	}
	return checkRunFromGetRow(row), true, nil
}
