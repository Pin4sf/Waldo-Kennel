package outcome_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// attemptFakeStore extends the plan fake with a faithful in-memory
// implementation of the Act & Observe store seam: numbered attempts,
// exclusive open custody fences, guarded transitions, ordered observations,
// and append-only receipts. SQLite-level fidelity is proven in the storage
// suites; this fake exists so admission orchestration is tested against real
// state transitions instead of mocks.
type attemptFakeStore struct {
	*planFakeStore

	mu             sync.Mutex
	projectOfSpace map[domain.ResponsibilitySpaceID]domain.ProjectID
	attempts       map[domain.OutcomeID][]domain.Attempt
	attemptByKey   map[string]domain.AttemptID
	fences         map[string][]domain.AttemptFence
	refs           map[domain.AttemptID][]domain.AttemptSessionRef
	obs            map[domain.AttemptID][]domain.AttemptObservation
	receipts       map[domain.AttemptID][]domain.AttemptRecoveryReceipt

	// dropActivationOnce simulates losing the queued->running promotion race.
	dropActivationOnce bool

	// failBindOnce simulates losing the durable session-binding write AFTER a
	// live spawn — the exact post-spawn failure the round-3 review required
	// be injectable end to end.
	failBindOnce bool

	// renewals counts fence-lease refreshes the liveness loop performs.
	renewals int
}

func newAttemptFakeStore() *attemptFakeStore {
	return &attemptFakeStore{
		planFakeStore:  newPlanFakeStore(),
		projectOfSpace: map[domain.ResponsibilitySpaceID]domain.ProjectID{},
		attempts:       map[domain.OutcomeID][]domain.Attempt{},
		attemptByKey:   map[string]domain.AttemptID{},
		fences:         map[string][]domain.AttemptFence{},
		refs:           map[domain.AttemptID][]domain.AttemptSessionRef{},
		obs:            map[domain.AttemptID][]domain.AttemptObservation{},
		receipts:       map[domain.AttemptID][]domain.AttemptRecoveryReceipt{},
	}
}

func (f *attemptFakeStore) EnsureWorkResponsibilitySpace(ctx context.Context, projectID domain.ProjectID) (domain.ResponsibilitySpace, error) {
	space, err := f.planFakeStore.EnsureWorkResponsibilitySpace(ctx, projectID)
	if err != nil {
		return space, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.projectOfSpace[space.ID] = projectID
	return space, nil
}

// GetOutcomeProjectID resolves the project through the Outcome's space,
// mirroring the SQL join.
func (f *attemptFakeStore) GetOutcomeProjectID(_ context.Context, outcomeID domain.OutcomeID) (domain.ProjectID, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	outcome, ok := f.outcomes[outcomeID]
	if !ok {
		return "", false, nil
	}
	id, ok := f.projectOfSpace[outcome.SpaceID]
	return id, ok, nil
}

func (f *attemptFakeStore) FindAttemptByIdempotencyKey(_ context.Context, key string) (domain.Attempt, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.attemptByKey[key]
	if !ok {
		return domain.Attempt{}, false, nil
	}
	for _, list := range f.attempts {
		for _, attempt := range list {
			if attempt.ID == id {
				return attempt, true, nil
			}
		}
	}
	return domain.Attempt{}, false, nil
}

func (f *attemptFakeStore) CreateAttemptWithFence(_ context.Context, outcomeID domain.OutcomeID, plan domain.PlanRevision, requestKey string, subject string, at time.Time) (domain.Attempt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, held := f.openFenceLocked(subject); held {
		holder, _ := f.openFenceLocked(subject)
		return domain.Attempt{}, &ports.AttemptFenceHeldError{Subject: subject, Holder: holder.AttemptID, OutcomeID: outcomeID}
	}
	if requestKey != "" {
		if existingKey, taken := f.attemptByKey[requestKey]; taken {
			for _, list := range f.attempts {
				for _, prior := range list {
					if prior.ID == existingKey {
						return prior, &ports.AttemptReplayError{Attempt: prior}
					}
				}
			}
		}
	}
	fakeAttemptCounter++
	attempt := domain.Attempt{
		ID:                     domain.AttemptID("att-" + string(rune('a'+fakeAttemptCounter%26)) + timeToSuffix(at)),
		OutcomeID:              outcomeID,
		PlanRevisionID:         plan.ID,
		WorkUnitID:             plan.WorkUnits[0].ID,
		Number:                 int64(len(f.attempts[outcomeID]) + 1),
		Status:                 domain.AttemptQueued,
		ContractRevisionNumber: plan.ContractRevisionNumber,
		RequestKey:             requestKey,
		CreatedAt:              at,
		UpdatedAt:              at,
	}
	f.attempts[outcomeID] = append(f.attempts[outcomeID], attempt)
	if requestKey != "" {
		f.attemptByKey[requestKey] = attempt.ID
	}
	f.fences[subject] = append(f.fences[subject], domain.AttemptFence{
		ID: "fence-" + string(attempt.ID), Subject: subject, AttemptID: attempt.ID, IssuedAt: at,
	})
	return attempt, nil
}

// openFenceLocked resolves the open fence over a subject. ok=false when free.
func (f *attemptFakeStore) openFenceLocked(subject string) (domain.AttemptFence, bool) {
	for _, fence := range f.fences[subject] {
		if fence.Open() {
			return fence, true
		}
	}
	return domain.AttemptFence{}, false
}

func (f *attemptFakeStore) GetAttempt(_ context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (domain.Attempt, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, attempt := range f.attempts[outcomeID] {
		if attempt.ID == attemptID {
			return attempt, true, nil
		}
	}
	return domain.Attempt{}, false, nil
}

func (f *attemptFakeStore) ListAttempts(_ context.Context, outcomeID domain.OutcomeID) ([]domain.Attempt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.Attempt, len(f.attempts[outcomeID]))
	copy(out, f.attempts[outcomeID])
	return out, nil
}

func (f *attemptFakeStore) TransitionAttemptStatus(_ context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, expected, next domain.AttemptStatus, at time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.dropActivationOnce && next == domain.AttemptRunning {
		f.dropActivationOnce = false
		return 0, nil // simulate losing the promotion race
	}
	for i, attempt := range f.attempts[outcomeID] {
		if attempt.ID != attemptID || attempt.Status != expected {
			continue
		}
		if !domain.AttemptTransitionLegal(expected, next) {
			return 0, errors.New("illegal attempt status transition")
		}
		f.attempts[outcomeID][i].Status = next
		f.attempts[outcomeID][i].UpdatedAt = at
		return 1, nil
	}
	return 0, nil
}

func (f *attemptFakeStore) ListAttemptsByStatus(_ context.Context, status domain.AttemptStatus) ([]domain.Attempt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Attempt
	for _, list := range f.attempts {
		for _, attempt := range list {
			if attempt.Status == status {
				out = append(out, attempt)
			}
		}
	}
	return out, nil
}

func (f *attemptFakeStore) BindAttemptSession(_ context.Context, ref domain.AttemptSessionRef) (domain.AttemptSessionRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failBindOnce {
		f.failBindOnce = false
		return domain.AttemptSessionRef{}, errors.New("injected binding write failure")
	}
	ref.Seq = int64(len(f.refs[ref.AttemptID]) + 1)
	ref.ID = domain.AttemptSessionRefID("asr-" + string(ref.AttemptID) + "-" + strconv.FormatInt(ref.Seq, 10))
	f.refs[ref.AttemptID] = append(f.refs[ref.AttemptID], ref)
	return ref, nil
}

func (f *attemptFakeStore) LatestAttemptSessionRef(_ context.Context, attemptID domain.AttemptID) (domain.AttemptSessionRef, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.refs[attemptID]
	if len(list) == 0 {
		return domain.AttemptSessionRef{}, false, nil
	}
	return list[len(list)-1], true, nil
}

func (f *attemptFakeStore) ListAttemptSessionRefs(_ context.Context, attemptID domain.AttemptID) ([]domain.AttemptSessionRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.AttemptSessionRef, len(f.refs[attemptID]))
	copy(out, f.refs[attemptID])
	return out, nil
}

func (f *attemptFakeStore) AppendAttemptObservation(_ context.Context, attemptID domain.AttemptID, kind string, payload string, at time.Time) (domain.AttemptObservation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	observation := domain.AttemptObservation{
		ID:        "obs-" + strings.ToLower(kind) + "-" + strconv.FormatInt(int64(len(f.obs[attemptID])+1), 10),
		AttemptID: attemptID,
		Seq:       int64(len(f.obs[attemptID]) + 1),
		Kind:      kind,
		Payload:   payload,
		CreatedAt: at,
	}
	if err := observation.Validate(); err != nil {
		return domain.AttemptObservation{}, err
	}
	f.obs[attemptID] = append(f.obs[attemptID], observation)
	return observation, nil
}

func (f *attemptFakeStore) ListAttemptObservations(_ context.Context, attemptID domain.AttemptID) ([]domain.AttemptObservation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.AttemptObservation, len(f.obs[attemptID]))
	copy(out, f.obs[attemptID])
	return out, nil
}

func (f *attemptFakeStore) OpenFenceForSubject(_ context.Context, subject string) (domain.AttemptFence, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fence, ok := f.openFenceLocked(subject)
	return fence, ok, nil
}

func (f *attemptFakeStore) ReleaseFenceForAttempt(_ context.Context, attemptID domain.AttemptID, reason string, at time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if reason == "" {
		return 0, errors.New("release reason required")
	}
	for subject, history := range f.fences {
		for i, fence := range history {
			if fence.AttemptID == attemptID && fence.Open() {
				f.fences[subject][i].ReleasedAt = at
				f.fences[subject][i].ReleaseReason = reason
				return 1, nil
			}
		}
	}
	return 0, nil
}

func (f *attemptFakeStore) RenewFenceForAttempt(_ context.Context, attemptID domain.AttemptID, at time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for subject, history := range f.fences {
		for i, fence := range history {
			if fence.AttemptID == attemptID && fence.Open() {
				f.fences[subject][i].LastRenewedAt = at
				f.renewals++
				return 1, nil
			}
		}
	}
	return 0, nil
}

func (f *attemptFakeStore) CreateRecoveryReceipt(_ context.Context, receipt domain.AttemptRecoveryReceipt) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := receipt.Validate(); err != nil {
		return err
	}
	f.receipts[receipt.AttemptID] = append(f.receipts[receipt.AttemptID], receipt)
	return nil
}

func (f *attemptFakeStore) ListRecoveryReceipts(_ context.Context, attemptID domain.AttemptID) ([]domain.AttemptRecoveryReceipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.AttemptRecoveryReceipt, len(f.receipts[attemptID]))
	copy(out, f.receipts[attemptID])
	return out, nil
}

var fakeAttemptCounter int

func timeToSuffix(t time.Time) string {
	return strconv.FormatInt(t.UnixNano()%100000, 10)
}

// fakeSpawner is the execution seam double: it records what admission asked
// for and answers with whatever failure the current scenario injects. NO
// fallback: the requested harness is echoed verbatim.
type fakeSpawner struct {
	mu           sync.Mutex
	readiness    ports.AgentProfileReadiness
	readinessErr error
	spawnErr     error
	terminateErr error
	// terminateResult shapes the next successful Terminate answer; nil means
	// a clean proven stop whose workspace was freed. Tests inject the
	// dirty-preserved shape {ProviderStopped:true, WorkspaceFreed:false} to
	// prove workspace preservation is not provider liveness.
	terminateResult *ports.TerminationResult
	terminated      []string
	spawned         []ports.AttemptSpawnRequest
	sessionN        int
}

func (f *fakeSpawner) ProfileReadiness(_ context.Context, _ domain.ProjectID, harness domain.AgentHarness) (ports.AgentProfileReadiness, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.readinessErr != nil {
		return ports.AgentProfileReadiness{}, f.readinessErr
	}
	return f.readiness, nil
}

func (f *fakeSpawner) Spawn(_ context.Context, req ports.AttemptSpawnRequest) (domain.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.spawned = append(f.spawned, req)
	if f.spawnErr != nil {
		return domain.Session{}, f.spawnErr
	}
	f.sessionN++
	rec := domain.SessionRecord{
		ID:      domain.SessionID("sess-provider-" + string(rune('a'+f.sessionN))),
		Mode:    domain.SessionModeTUI,
		Harness: req.Harness,
		Activity: domain.Activity{
			State:          domain.ActivityActive,
			LastActivityAt: time.Now(),
		},
	}
	return domain.Session{SessionRecord: rec}, nil
}

// Terminate records the request; failures AND result shapes are injectable
// per scenario. Tests pair a successful Terminate with heartbeats.terminate(...)
// to mirror the real flow, where Kill writes the durable is_terminated fact.
func (f *fakeSpawner) Terminate(_ context.Context, _ domain.ProjectID, sessionID string) (ports.TerminationResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.terminateErr != nil {
		return ports.TerminationResult{}, f.terminateErr
	}
	f.terminated = append(f.terminated, sessionID)
	if f.terminateResult != nil {
		return *f.terminateResult, nil
	}
	return ports.TerminationResult{ProviderStopped: true, WorkspaceFreed: true}, nil
}

func (f *fakeSpawner) spawnCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.spawned)
}

func (f *fakeSpawner) failNextSpawn(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.spawnErr = err
}

func (f *fakeSpawner) failNextTerminate(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.terminateErr = err
}

// setTerminateResult injects the two-fact answer a successful stop reports.
func (f *fakeSpawner) setTerminateResult(res ports.TerminationResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.terminateResult = &res
}

// fakeHeartbeats stands in for the sessions table: tests mutate it to model
// signalled, silent, GC'd, and terminated provider sessions.
type fakeHeartbeats struct {
	mu       sync.Mutex
	sessions map[domain.SessionID]domain.SessionRecord
}

func newFakeHeartbeats() *fakeHeartbeats {
	return &fakeHeartbeats{sessions: map[domain.SessionID]domain.SessionRecord{}}
}

func (f *fakeHeartbeats) GetSession(_ context.Context, id domain.SessionID) (domain.SessionRecord, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rec, ok := f.sessions[id]
	return rec, ok, nil
}

func (f *fakeHeartbeats) signal(id domain.SessionID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rec := f.sessions[id]
	rec.FirstSignalAt = time.Now().UTC()
	rec.Activity = domain.Activity{State: domain.ActivityActive, LastActivityAt: time.Now().UTC()}
	f.sessions[id] = rec
}

func (f *fakeHeartbeats) terminate(id domain.SessionID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rec := f.sessions[id]
	rec.IsTerminated = true
	f.sessions[id] = rec
}

// backdate ages the session's durable activity WITHOUT terminating it — the
// stale-heartbeat shape the lease gate must refuse to renew.
func (f *fakeHeartbeats) backdate(id domain.SessionID, age time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rec := f.sessions[id]
	rec.Activity = domain.Activity{State: domain.ActivityActive, LastActivityAt: time.Now().Add(-age)}
	f.sessions[id] = rec
}

func (f *fakeHeartbeats) forget(id domain.SessionID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, id)
}

// The bare contract/plan fakes satisfy the widened OutcomeStore interface
// with inert stubs; attemptFakeStore above shadows every one of them with the
// real in-memory behavior the execution tests exercise.
func (f *fakeStore) GetOutcomeProjectID(context.Context, domain.OutcomeID) (domain.ProjectID, bool, error) {
	return "", false, nil
}
func (f *fakeStore) FindAttemptByIdempotencyKey(context.Context, string) (domain.Attempt, bool, error) {
	return domain.Attempt{}, false, nil
}
func (f *fakeStore) CreateAttemptWithFence(context.Context, domain.OutcomeID, domain.PlanRevision, string, string, time.Time) (domain.Attempt, error) {
	return domain.Attempt{}, nil
}
func (f *fakeStore) GetAttempt(context.Context, domain.OutcomeID, domain.AttemptID) (domain.Attempt, bool, error) {
	return domain.Attempt{}, false, nil
}
func (f *fakeStore) ListAttempts(context.Context, domain.OutcomeID) ([]domain.Attempt, error) {
	return nil, nil
}
func (f *fakeStore) TransitionAttemptStatus(context.Context, domain.OutcomeID, domain.AttemptID, domain.AttemptStatus, domain.AttemptStatus, time.Time) (int64, error) {
	return 0, nil
}
func (f *fakeStore) ListAttemptsByStatus(context.Context, domain.AttemptStatus) ([]domain.Attempt, error) {
	return nil, nil
}
func (f *fakeStore) BindAttemptSession(context.Context, domain.AttemptSessionRef) (domain.AttemptSessionRef, error) {
	return domain.AttemptSessionRef{}, nil
}
func (f *fakeStore) LatestAttemptSessionRef(context.Context, domain.AttemptID) (domain.AttemptSessionRef, bool, error) {
	return domain.AttemptSessionRef{}, false, nil
}
func (f *fakeStore) ListAttemptSessionRefs(context.Context, domain.AttemptID) ([]domain.AttemptSessionRef, error) {
	return nil, nil
}
func (f *fakeStore) AppendAttemptObservation(context.Context, domain.AttemptID, string, string, time.Time) (domain.AttemptObservation, error) {
	return domain.AttemptObservation{}, nil
}
func (f *fakeStore) ListAttemptObservations(context.Context, domain.AttemptID) ([]domain.AttemptObservation, error) {
	return nil, nil
}
func (f *fakeStore) OpenFenceForSubject(context.Context, string) (domain.AttemptFence, bool, error) {
	return domain.AttemptFence{}, false, nil
}
func (f *fakeStore) ReleaseFenceForAttempt(context.Context, domain.AttemptID, string, time.Time) (int64, error) {
	return 0, nil
}
func (f *fakeStore) CreateRecoveryReceipt(context.Context, domain.AttemptRecoveryReceipt) error {
	return nil
}
func (f *fakeStore) ListRecoveryReceipts(context.Context, domain.AttemptID) ([]domain.AttemptRecoveryReceipt, error) {
	return nil, nil
}

func (f *fakeStore) RenewFenceForAttempt(context.Context, domain.AttemptID, time.Time) (int64, error) {
	return 0, nil
}
