package outcome_test

import (
	"context"
	"sync"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// checkRunFakeStore is the durable check-run record. It enforces the one
// property the design rests on: a reservation for a given (attempt, check,
// artifact) exists at most once, so only one caller may invoke the command.
type checkRunFakeStore struct {
	mu   sync.Mutex
	runs map[string]ports.AttemptCheckRun
}

func newCheckRunFakeStore() *checkRunFakeStore {
	return &checkRunFakeStore{runs: map[string]ports.AttemptCheckRun{}}
}

func checkRunFakeKey(attemptID domain.AttemptID, checkID domain.ApprovedCheckID, version string) string {
	return string(attemptID) + "|" + string(checkID) + "|" + version
}

func (f *checkRunFakeStore) ReserveAttemptCheckRun(_ context.Context, run ports.AttemptCheckRun) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := checkRunFakeKey(run.AttemptID, run.CheckID, run.ArtifactVersion)
	if _, exists := f.runs[key]; exists {
		return ports.ErrCheckRunAlreadyReserved
	}
	run.State = ports.CheckRunReserved
	f.runs[key] = run
	return nil
}

func (f *checkRunFakeStore) GetAttemptCheckRun(_ context.Context, attemptID domain.AttemptID, checkID domain.ApprovedCheckID, version string) (ports.AttemptCheckRun, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	run, ok := f.runs[checkRunFakeKey(attemptID, checkID, version)]
	return run, ok, nil
}

func (f *checkRunFakeStore) ListAttemptCheckRuns(_ context.Context, attemptID domain.AttemptID, version string) ([]ports.AttemptCheckRun, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []ports.AttemptCheckRun
	for _, run := range f.runs {
		if run.AttemptID == attemptID && run.ArtifactVersion == version {
			out = append(out, run)
		}
	}
	return out, nil
}

func (f *checkRunFakeStore) RecordAttemptCheckObservation(_ context.Context, run ports.AttemptCheckRun) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := checkRunFakeKey(run.AttemptID, run.CheckID, run.ArtifactVersion)
	existing, ok := f.runs[key]
	if !ok {
		return ports.ErrCheckRunAlreadyReserved
	}
	if existing.State == ports.CheckRunObserved {
		// Write-once: a recorded observation is what proof is rebuilt from.
		return nil
	}
	run.State = ports.CheckRunObserved
	run.ReservedAt = existing.ReservedAt
	f.runs[key] = run
	return nil
}

func (f *checkRunFakeStore) MarkAttemptCheckRunUnknown(_ context.Context, attemptID domain.AttemptID, checkID domain.ApprovedCheckID, version string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := checkRunFakeKey(attemptID, checkID, version)
	run, ok := f.runs[key]
	if !ok || run.State != ports.CheckRunReserved {
		return nil
	}
	run.State = ports.CheckRunUnknown
	observed := at.UTC()
	run.ObservedAt = &observed
	f.runs[key] = run
	return nil
}

// forgetObservations drops every recorded observation while keeping the
// reservations, which is what a crash between reserving and recording leaves
// behind.
func (f *checkRunFakeStore) forgetObservations() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for key, run := range f.runs {
		run.State = ports.CheckRunReserved
		run.ObservedAt = nil
		f.runs[key] = run
	}
}

// countingCheckRunner records every invocation so a test can prove a command
// ran exactly once across repeated reconciliation.
type countingCheckRunner struct {
	mu sync.Mutex
	// invoked lists the check ids handed to each call, one entry per call.
	invoked [][]domain.ApprovedCheckID
	// observe shapes what each check reports; nil means it passes.
	observe func(domain.ApprovedCheck) ports.AttemptCheckObservation
	// observedVersion is what the workspace measures after the run. Empty
	// means unchanged, which the request fills in from the receipt.
	observedVersion string
}

func (r *countingCheckRunner) RunAttemptChecks(_ context.Context, req ports.AttemptCheckRequest) (ports.AttemptCheckResult, error) {
	r.mu.Lock()
	ids := make([]domain.ApprovedCheckID, 0, len(req.Checks))
	for _, check := range req.Checks {
		ids = append(ids, check.ID)
	}
	r.invoked = append(r.invoked, ids)
	observe := r.observe
	version := r.observedVersion
	r.mu.Unlock()

	result := ports.AttemptCheckResult{ObservedArtifactVersion: version}
	if version == "" {
		result.ObservedArtifactVersion = req.Receipt.ArtifactVersion
	}
	for _, check := range req.Checks {
		if observe != nil {
			result.Observations = append(result.Observations, observe(check))
			continue
		}
		// A default pass is a DISCRIMINATING pass: the command failed against a
		// workspace holding none of the work, which is what lets a zero exit
		// support the criterion at all.
		result.Observations = append(result.Observations, ports.AttemptCheckObservation{
			Check: check, ArtifactVersion: req.Receipt.ArtifactVersion,
			Ran: true, Passed: true, EnforcedBy: "test-enforcement",
			BaselineRan: true, BaselinePassed: false,
		})
	}
	return result, nil
}

func (r *countingCheckRunner) calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.invoked)
}

func (r *countingCheckRunner) totalChecksInvoked() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	total := 0
	for _, ids := range r.invoked {
		total += len(ids)
	}
	return total
}
