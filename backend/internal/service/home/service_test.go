package home

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

type replayedWrite struct {
	fingerprint string
	kind        string
	capture     ports.QuickCapture
	loop        ports.OpenLoopSnapshot
}

// fakeHomeStore implements the transactional guarantees expected from the
// future SQLite adapter. Service tests assert observable Home behavior; the
// fake keeps atomicity, idempotency, and optimistic concurrency honest.
type fakeHomeStore struct {
	mu sync.Mutex

	captures map[ports.QuickCaptureID]ports.QuickCapture
	loops    map[domain.OpenLoopID]ports.OpenLoopSnapshot
	replays  map[string]replayedWrite

	writes           int
	createCalls      int
	dispositionCalls int
	createErr        error
}

func newFakeHomeStore() *fakeHomeStore {
	return &fakeHomeStore{
		captures: make(map[ports.QuickCaptureID]ports.QuickCapture),
		loops:    make(map[domain.OpenLoopID]ports.OpenLoopSnapshot),
		replays:  make(map[string]replayedWrite),
	}
}

func cloneOpenLoopSnapshot(in ports.OpenLoopSnapshot) ports.OpenLoopSnapshot {
	out := in
	out.Dispositions = append([]domain.LoopDisposition(nil), in.Dispositions...)
	return out
}

func (f *fakeHomeStore) SaveQuickCapture(_ context.Context, capture ports.QuickCapture, request ports.HomeIdempotency) (ports.QuickCapture, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if replay, ok := f.replays[request.Key]; ok {
		if replay.fingerprint != request.Fingerprint || replay.kind != "capture" {
			return ports.QuickCapture{}, &ports.HomeIdempotencyConflictError{Key: request.Key}
		}
		return replay.capture, nil
	}
	f.captures[capture.ID] = capture
	f.replays[request.Key] = replayedWrite{fingerprint: request.Fingerprint, kind: "capture", capture: capture}
	f.writes++
	return capture, nil
}

func (f *fakeHomeStore) CreateOpenLoopWithConfirmation(_ context.Context, loop domain.OpenLoop, confirmation domain.LoopDisposition, request ports.HomeIdempotency) (ports.OpenLoopSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls++

	if replay, ok := f.replays[request.Key]; ok {
		if replay.fingerprint != request.Fingerprint || replay.kind != "create_open_loop" {
			return ports.OpenLoopSnapshot{}, &ports.HomeIdempotencyConflictError{Key: request.Key}
		}
		return cloneOpenLoopSnapshot(replay.loop), nil
	}
	if f.createErr != nil {
		return ports.OpenLoopSnapshot{}, f.createErr
	}

	snapshot := ports.OpenLoopSnapshot{
		OpenLoop:     loop,
		Dispositions: []domain.LoopDisposition{confirmation},
		Revision:     1,
	}
	f.loops[loop.ID] = cloneOpenLoopSnapshot(snapshot)
	f.replays[request.Key] = replayedWrite{
		fingerprint: request.Fingerprint,
		kind:        "create_open_loop",
		loop:        cloneOpenLoopSnapshot(snapshot),
	}
	f.writes++
	return cloneOpenLoopSnapshot(snapshot), nil
}

func (f *fakeHomeStore) AppendLoopDisposition(_ context.Context, loopID domain.OpenLoopID, expectedRevision int64, disposition domain.LoopDisposition, request ports.HomeIdempotency) (ports.OpenLoopSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dispositionCalls++

	if replay, ok := f.replays[request.Key]; ok {
		if replay.fingerprint != request.Fingerprint || replay.kind != "append_disposition" {
			return ports.OpenLoopSnapshot{}, &ports.HomeIdempotencyConflictError{Key: request.Key}
		}
		return cloneOpenLoopSnapshot(replay.loop), nil
	}
	snapshot, ok := f.loops[loopID]
	if !ok {
		return ports.OpenLoopSnapshot{}, errors.New("open loop not found")
	}
	if snapshot.Revision != expectedRevision {
		return ports.OpenLoopSnapshot{}, &ports.OpenLoopRevisionConflictError{
			OpenLoopID:       loopID,
			ExpectedRevision: expectedRevision,
			CurrentRevision:  snapshot.Revision,
		}
	}

	snapshot.Dispositions = append(snapshot.Dispositions, disposition)
	snapshot.Revision++
	f.loops[loopID] = cloneOpenLoopSnapshot(snapshot)
	f.replays[request.Key] = replayedWrite{
		fingerprint: request.Fingerprint,
		kind:        "append_disposition",
		loop:        cloneOpenLoopSnapshot(snapshot),
	}
	f.writes++
	return cloneOpenLoopSnapshot(snapshot), nil
}

func fixedNow() time.Time {
	return time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
}

func newTestService() (*Service, *fakeHomeStore) {
	store := newFakeHomeStore()
	return New(store, fixedNow), store
}

func validCreateOpenLoopInput() CreateOpenLoopInput {
	return CreateOpenLoopInput{
		SpaceID:            domain.ResponsibilitySpaceID("home-1"),
		Meaning:            "Renew passport",
		Owner:              "owner",
		Trigger:            "Passport expires within six months",
		RecheckCondition:   "Recheck after the renewal appointment",
		SourceRef:          "quick-capture:cap-1",
		Confirmation:       domain.OpenLoopConfirmationExplicit,
		ConfirmationReason: "I want Waldo to preserve this responsibility",
		RequestKey:         "home-create-1",
	}
}

func TestCapturePreservesNoteOrCandidateWithoutCreatingCanonicalResponsibility(t *testing.T) {
	for _, kind := range []ports.QuickCaptureKind{
		ports.QuickCaptureKindNote,
		ports.QuickCaptureKindOpenLoopCandidate,
	} {
		t.Run(string(kind), func(t *testing.T) {
			svc, store := newTestService()
			result, err := svc.Capture(context.Background(), CaptureInput{
				SpaceID:    domain.ResponsibilitySpaceID("home-1"),
				Text:       "Renew passport",
				Kind:       kind,
				RequestKey: "capture-" + string(kind),
			})
			if err != nil {
				t.Fatalf("Capture() error = %v", err)
			}
			if result.Capture.Kind != kind || result.Capture.Text != "Renew passport" {
				t.Fatalf("Capture() = %+v, want preserved %s", result.Capture, kind)
			}
			if len(store.loops) != 0 {
				t.Fatalf("Capture() created %d canonical Open Loops", len(store.loops))
			}
		})
	}
}

func TestCreateOpenLoopPersistsConfirmationAtomically(t *testing.T) {
	for _, confirmation := range []domain.OpenLoopConfirmation{
		domain.OpenLoopConfirmationExplicit,
		domain.OpenLoopConfirmationDirectCommand,
	} {
		t.Run(string(confirmation), func(t *testing.T) {
			svc, store := newTestService()
			in := validCreateOpenLoopInput()
			in.Confirmation = confirmation
			in.RequestKey += "-" + string(confirmation)

			view, err := svc.CreateOpenLoop(context.Background(), in)
			if err != nil {
				t.Fatalf("CreateOpenLoop() error = %v", err)
			}
			if !view.OpenLoop.IsCanonical() {
				t.Fatalf("CreateOpenLoop() returned non-canonical loop: %+v", view.OpenLoop)
			}
			if view.Revision != 1 || len(view.Dispositions) != 1 {
				t.Fatalf("CreateOpenLoop() revision/history = %d/%d, want 1/1", view.Revision, len(view.Dispositions))
			}
			initial := view.Dispositions[0]
			if initial.Kind != domain.LoopDispositionConfirm || initial.Actor != domain.LoopDispositionActorOwner {
				t.Fatalf("initial disposition = %+v, want owner confirmation", initial)
			}
			if initial.OpenLoopID != view.OpenLoop.ID {
				t.Fatalf("confirmation loop = %s, want %s", initial.OpenLoopID, view.OpenLoop.ID)
			}
			if store.createCalls != 1 || store.writes != 1 {
				t.Fatalf("port calls/writes = %d/%d, want one atomic write", store.createCalls, store.writes)
			}
		})
	}
}

func TestRecordDispositionRejectsProviderSessionCompletion(t *testing.T) {
	for _, kind := range []domain.LoopDispositionKind{
		domain.LoopDispositionClose,
		domain.LoopDispositionRelease,
		domain.LoopDispositionReopen,
		domain.LoopDispositionTransfer,
		domain.LoopDispositionSupersede,
	} {
		t.Run(string(kind), func(t *testing.T) {
			svc, store := newTestService()
			created, err := svc.CreateOpenLoop(context.Background(), validCreateOpenLoopInput())
			if err != nil {
				t.Fatalf("CreateOpenLoop() error = %v", err)
			}
			beforeWrites := store.writes
			in := DispositionInput{
				ExpectedRevision: created.Revision,
				Kind:             kind,
				Actor:            domain.LoopDispositionActorProvider,
				Reason:           "provider session completed",
				RequestKey:       "provider-" + string(kind),
			}
			if kind == domain.LoopDispositionTransfer {
				in.TargetSpaceID = domain.ResponsibilitySpaceID("home-2")
				in.TargetOwner = "other-owner"
			}
			if kind == domain.LoopDispositionSupersede {
				in.SuccessorRef = "open-loop:loop-2"
			}

			_, err = svc.RecordDisposition(context.Background(), created.OpenLoop.ID, in)
			var apiErr *apierr.Error
			if !errors.As(err, &apiErr) || apiErr.Code != "OPEN_LOOP_OWNER_DECISION_REQUIRED" {
				t.Fatalf("RecordDisposition() error = %v, want owner-decision rejection", err)
			}
			if store.dispositionCalls != 0 || store.writes != beforeWrites {
				t.Fatalf("provider completion reached port: calls=%d writes=%d", store.dispositionCalls, store.writes)
			}
		})
	}
}

func TestCreateOpenLoopIdempotencyReplayReturnsOriginalWithoutDuplicateWrite(t *testing.T) {
	svc, store := newTestService()
	in := validCreateOpenLoopInput()

	first, err := svc.CreateOpenLoop(context.Background(), in)
	if err != nil {
		t.Fatalf("first CreateOpenLoop() error = %v", err)
	}
	replayed, err := svc.CreateOpenLoop(context.Background(), in)
	if err != nil {
		t.Fatalf("replayed CreateOpenLoop() error = %v", err)
	}
	if replayed.OpenLoop.ID != first.OpenLoop.ID || replayed.Dispositions[0].ID != first.Dispositions[0].ID {
		t.Fatalf("replay IDs = %s/%s, want originals %s/%s",
			replayed.OpenLoop.ID, replayed.Dispositions[0].ID, first.OpenLoop.ID, first.Dispositions[0].ID)
	}
	if store.writes != 1 {
		t.Fatalf("idempotent replay persisted %d writes, want 1", store.writes)
	}
}

func TestCreateOpenLoopIdempotencyKeyReuseWithDifferentInputFailsClosed(t *testing.T) {
	svc, store := newTestService()
	first := validCreateOpenLoopInput()
	if _, err := svc.CreateOpenLoop(context.Background(), first); err != nil {
		t.Fatalf("first CreateOpenLoop() error = %v", err)
	}

	different := first
	different.Meaning = "Renew driving licence"
	_, err := svc.CreateOpenLoop(context.Background(), different)
	var apiErr *apierr.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "HOME_IDEMPOTENCY_CONFLICT" {
		t.Fatalf("CreateOpenLoop() error = %v, want HOME_IDEMPOTENCY_CONFLICT", err)
	}
	if store.writes != 1 || len(store.loops) != 1 {
		t.Fatalf("conflicting key changed state: writes=%d loops=%d", store.writes, len(store.loops))
	}
}

func TestRecordDispositionStaleExpectedRevisionConflictsWithoutWrite(t *testing.T) {
	svc, store := newTestService()
	created, err := svc.CreateOpenLoop(context.Background(), validCreateOpenLoopInput())
	if err != nil {
		t.Fatalf("CreateOpenLoop() error = %v", err)
	}

	closed, err := svc.RecordDisposition(context.Background(), created.OpenLoop.ID, DispositionInput{
		ExpectedRevision: 1,
		Kind:             domain.LoopDispositionClose,
		Actor:            domain.LoopDispositionActorOwner,
		Reason:           "The passport was renewed",
		RequestKey:       "close-1",
	})
	if err != nil {
		t.Fatalf("close RecordDisposition() error = %v", err)
	}
	if closed.Revision != 2 || len(closed.Dispositions) != 2 {
		t.Fatalf("closed revision/history = %d/%d, want 2/2", closed.Revision, len(closed.Dispositions))
	}
	beforeWrites := store.writes

	_, err = svc.RecordDisposition(context.Background(), created.OpenLoop.ID, DispositionInput{
		ExpectedRevision: 1,
		Kind:             domain.LoopDispositionReopen,
		Actor:            domain.LoopDispositionActorOwner,
		Reason:           "A correction is still needed",
		RequestKey:       "reopen-stale",
	})
	var apiErr *apierr.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "OPEN_LOOP_REVISION_CONFLICT" {
		t.Fatalf("stale RecordDisposition() error = %v, want revision conflict", err)
	}
	if store.writes != beforeWrites {
		t.Fatalf("stale disposition persisted a write: before=%d after=%d", beforeWrites, store.writes)
	}
}

func TestCreateOpenLoopStoreFailureCannotLeavePartialConfirmation(t *testing.T) {
	svc, store := newTestService()
	store.createErr = errors.New("transaction commit failed")

	_, err := svc.CreateOpenLoop(context.Background(), validCreateOpenLoopInput())
	if !errors.Is(err, store.createErr) {
		t.Fatalf("CreateOpenLoop() error = %v, want store failure", err)
	}
	if len(store.loops) != 0 || store.writes != 0 {
		t.Fatalf("failed transaction left partial state: loops=%d writes=%d", len(store.loops), store.writes)
	}
}

func TestCaptureOpenLoopAndDispositionValidationRejectsBeforeStore(t *testing.T) {
	t.Run("capture", func(t *testing.T) {
		valid := CaptureInput{
			SpaceID:    domain.ResponsibilitySpaceID("home-1"),
			Text:       "Renew passport",
			Kind:       ports.QuickCaptureKindNote,
			RequestKey: "capture-valid",
		}
		tests := []struct {
			name     string
			mutate   func(*CaptureInput)
			wantCode string
		}{
			{name: "missing space", mutate: func(in *CaptureInput) { in.SpaceID = "" }, wantCode: "HOME_SPACE_REQUIRED"},
			{name: "missing text", mutate: func(in *CaptureInput) { in.Text = " " }, wantCode: "HOME_CAPTURE_TEXT_REQUIRED"},
			{name: "invalid kind", mutate: func(in *CaptureInput) { in.Kind = "task" }, wantCode: "HOME_CAPTURE_KIND_INVALID"},
			{name: "missing request key", mutate: func(in *CaptureInput) { in.RequestKey = "" }, wantCode: "REQUEST_KEY_REQUIRED"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, store := newTestService()
				in := valid
				tt.mutate(&in)
				_, err := svc.Capture(context.Background(), in)
				assertAPIErrorCode(t, err, tt.wantCode)
				if store.writes != 0 {
					t.Fatalf("invalid capture persisted %d writes", store.writes)
				}
			})
		}
	})

	t.Run("create Open Loop", func(t *testing.T) {
		tests := []struct {
			name     string
			mutate   func(*CreateOpenLoopInput)
			wantCode string
		}{
			{name: "missing request key", mutate: func(in *CreateOpenLoopInput) { in.RequestKey = " " }, wantCode: "REQUEST_KEY_REQUIRED"},
			{name: "missing confirmation reason", mutate: func(in *CreateOpenLoopInput) { in.ConfirmationReason = "" }, wantCode: "OPEN_LOOP_CONFIRMATION_REASON_REQUIRED"},
			{name: "missing confirmation", mutate: func(in *CreateOpenLoopInput) { in.Confirmation = "" }, wantCode: "OPEN_LOOP_INVALID"},
			{name: "missing owner", mutate: func(in *CreateOpenLoopInput) { in.Owner = "" }, wantCode: "OPEN_LOOP_INVALID"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, store := newTestService()
				in := validCreateOpenLoopInput()
				tt.mutate(&in)
				_, err := svc.CreateOpenLoop(context.Background(), in)
				assertAPIErrorCode(t, err, tt.wantCode)
				if store.createCalls != 0 || store.writes != 0 {
					t.Fatalf("invalid create reached store: calls=%d writes=%d", store.createCalls, store.writes)
				}
			})
		}
	})

	t.Run("record disposition", func(t *testing.T) {
		tests := []struct {
			name     string
			loopID   domain.OpenLoopID
			mutate   func(*DispositionInput)
			wantCode string
		}{
			{name: "missing loop", loopID: "", mutate: func(*DispositionInput) {}, wantCode: "OPEN_LOOP_ID_REQUIRED"},
			{name: "missing revision", loopID: "created", mutate: func(in *DispositionInput) { in.ExpectedRevision = 0 }, wantCode: "EXPECTED_REVISION_REQUIRED"},
			{name: "missing request key", loopID: "created", mutate: func(in *DispositionInput) { in.RequestKey = "" }, wantCode: "REQUEST_KEY_REQUIRED"},
			{name: "invalid kind", loopID: "created", mutate: func(in *DispositionInput) { in.Kind = "done" }, wantCode: "OPEN_LOOP_DISPOSITION_INVALID"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, store := newTestService()
				created, err := svc.CreateOpenLoop(context.Background(), validCreateOpenLoopInput())
				if err != nil {
					t.Fatalf("CreateOpenLoop() error = %v", err)
				}
				loopID := tt.loopID
				if loopID == "created" {
					loopID = created.OpenLoop.ID
				}
				in := DispositionInput{
					ExpectedRevision: created.Revision,
					Kind:             domain.LoopDispositionClose,
					Actor:            domain.LoopDispositionActorOwner,
					Reason:           "The responsibility became true",
					RequestKey:       "close-valid",
				}
				tt.mutate(&in)
				beforeWrites := store.writes
				_, err = svc.RecordDisposition(context.Background(), loopID, in)
				assertAPIErrorCode(t, err, tt.wantCode)
				if store.dispositionCalls != 0 || store.writes != beforeWrites {
					t.Fatalf("invalid disposition reached store: calls=%d writes=%d", store.dispositionCalls, store.writes)
				}
			})
		}
	})
}

func assertAPIErrorCode(t *testing.T, err error, want string) {
	t.Helper()
	var apiErr *apierr.Error
	if !errors.As(err, &apiErr) || apiErr.Code != want {
		t.Fatalf("error = %v, want apierr code %s", err, want)
	}
}
