// Package home owns Quick Capture and canonical Open Loop lifecycle policy.
package home

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/google/uuid"
)

// Manager is the controller-facing Home service boundary.
type Manager interface {
	Capture(context.Context, CaptureInput) (CaptureResult, error)
	CreateOpenLoop(context.Context, CreateOpenLoopInput) (OpenLoopView, error)
	RecordDisposition(context.Context, domain.OpenLoopID, DispositionInput) (OpenLoopView, error)
}

type CaptureInput struct {
	SpaceID    domain.ResponsibilitySpaceID
	Text       string
	Kind       ports.QuickCaptureKind
	RequestKey string
}

type CaptureResult struct {
	Capture ports.QuickCapture
}

type CreateOpenLoopInput struct {
	SpaceID            domain.ResponsibilitySpaceID
	Meaning            string
	Owner              string
	Trigger            string
	RecheckCondition   string
	SourceRef          string
	Confirmation       domain.OpenLoopConfirmation
	ConfirmationReason string
	RequestKey         string
}

type DispositionInput struct {
	ExpectedRevision int64
	Kind             domain.LoopDispositionKind
	Actor            domain.LoopDispositionActor
	Reason           string
	TargetSpaceID    domain.ResponsibilitySpaceID
	TargetOwner      string
	SuccessorRef     string
	RequestKey       string
}

type OpenLoopView struct {
	OpenLoop     domain.OpenLoop
	Dispositions []domain.LoopDisposition
	Revision     int64
}

type Service struct {
	store ports.HomeStore
	clock func() time.Time
}

func New(store ports.HomeStore, clock func() time.Time) *Service {
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, clock: clock}
}

var _ Manager = (*Service)(nil)

func (s *Service) Capture(ctx context.Context, in CaptureInput) (CaptureResult, error) {
	in.SpaceID = domain.ResponsibilitySpaceID(strings.TrimSpace(in.SpaceID.String()))
	in.Text = strings.TrimSpace(in.Text)
	in.RequestKey = strings.TrimSpace(in.RequestKey)
	switch {
	case in.SpaceID.IsZero():
		return CaptureResult{}, apierr.Invalid("HOME_SPACE_REQUIRED", "Choose the Home space for this capture", nil)
	case in.Text == "":
		return CaptureResult{}, apierr.Invalid("HOME_CAPTURE_TEXT_REQUIRED", "Enter something to preserve", nil)
	case !in.Kind.Valid():
		return CaptureResult{}, apierr.Invalid("HOME_CAPTURE_KIND_INVALID", "Preserve this capture as a note or candidate", nil)
	case in.RequestKey == "":
		return CaptureResult{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this capture", nil)
	}

	capture := ports.QuickCapture{
		ID:        ports.QuickCaptureID("cap-" + uuid.NewString()),
		SpaceID:   in.SpaceID,
		Text:      in.Text,
		Kind:      in.Kind,
		CreatedAt: s.clock(),
	}
	request := ports.HomeIdempotency{
		Key:         in.RequestKey,
		Fingerprint: fingerprint("capture", in.SpaceID.String(), in.Text, string(in.Kind)),
	}
	stored, err := s.store.SaveQuickCapture(ctx, capture, request)
	if err != nil {
		return CaptureResult{}, mapStoreError(err)
	}
	return CaptureResult{Capture: stored}, nil
}

func (s *Service) CreateOpenLoop(ctx context.Context, in CreateOpenLoopInput) (OpenLoopView, error) {
	in = normalizeCreateOpenLoopInput(in)
	if in.RequestKey == "" {
		return OpenLoopView{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this Open Loop", nil)
	}
	if in.ConfirmationReason == "" {
		return OpenLoopView{}, apierr.Invalid("OPEN_LOOP_CONFIRMATION_REASON_REQUIRED", "Record why this responsibility was confirmed", nil)
	}

	now := s.clock()
	loopID := domain.OpenLoopID("loop-" + uuid.NewString())
	loop, err := domain.NewOpenLoopCandidate(in.Meaning).Confirm(domain.ConfirmOpenLoopInput{
		ID:               loopID,
		SpaceID:          in.SpaceID,
		Owner:            in.Owner,
		Trigger:          in.Trigger,
		RecheckCondition: in.RecheckCondition,
		SourceRef:        in.SourceRef,
		Confirmation:     in.Confirmation,
		ConfirmedAt:      now,
	})
	if err != nil {
		return OpenLoopView{}, apierr.Invalid("OPEN_LOOP_INVALID", err.Error(), nil)
	}
	confirmation := domain.LoopDisposition{
		ID:         domain.LoopDispositionID("disp-" + uuid.NewString()),
		OpenLoopID: loop.ID,
		Kind:       domain.LoopDispositionConfirm,
		Actor:      domain.LoopDispositionActorOwner,
		Reason:     in.ConfirmationReason,
		OccurredAt: now,
	}
	if err := confirmation.Validate(); err != nil {
		return OpenLoopView{}, apierr.Invalid("OPEN_LOOP_CONFIRMATION_INVALID", err.Error(), nil)
	}

	request := ports.HomeIdempotency{
		Key: in.RequestKey,
		Fingerprint: fingerprint(
			"create_open_loop",
			in.SpaceID.String(), in.Meaning, in.Owner, in.Trigger,
			in.RecheckCondition, in.SourceRef, string(in.Confirmation),
			in.ConfirmationReason,
		),
	}
	snapshot, err := s.store.CreateOpenLoopWithConfirmation(ctx, loop, confirmation, request)
	if err != nil {
		return OpenLoopView{}, mapStoreError(err)
	}
	return viewFromSnapshot(snapshot), nil
}

func (s *Service) RecordDisposition(ctx context.Context, loopID domain.OpenLoopID, in DispositionInput) (OpenLoopView, error) {
	loopID = domain.OpenLoopID(strings.TrimSpace(loopID.String()))
	in = normalizeDispositionInput(in)
	switch {
	case loopID.IsZero():
		return OpenLoopView{}, apierr.Invalid("OPEN_LOOP_ID_REQUIRED", "Choose the Open Loop to update", nil)
	case in.ExpectedRevision < 1:
		return OpenLoopView{}, apierr.Invalid("EXPECTED_REVISION_REQUIRED", "State which Open Loop revision this decision follows", nil)
	case in.RequestKey == "":
		return OpenLoopView{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this disposition", nil)
	case in.Actor != domain.LoopDispositionActorOwner:
		return OpenLoopView{}, apierr.Forbidden("OPEN_LOOP_OWNER_DECISION_REQUIRED", "Only the owner can change an Open Loop's lifecycle")
	}

	disposition := domain.LoopDisposition{
		ID:            domain.LoopDispositionID("disp-" + uuid.NewString()),
		OpenLoopID:    loopID,
		Kind:          in.Kind,
		Actor:         in.Actor,
		Reason:        in.Reason,
		TargetSpaceID: in.TargetSpaceID,
		TargetOwner:   in.TargetOwner,
		SuccessorRef:  in.SuccessorRef,
		OccurredAt:    s.clock(),
	}
	if err := disposition.Validate(); err != nil {
		return OpenLoopView{}, apierr.Invalid("OPEN_LOOP_DISPOSITION_INVALID", err.Error(), nil)
	}

	request := ports.HomeIdempotency{
		Key: in.RequestKey,
		Fingerprint: fingerprint(
			"append_disposition", loopID.String(), strconv.FormatInt(in.ExpectedRevision, 10),
			string(in.Kind), string(in.Actor), in.Reason, in.TargetSpaceID.String(),
			in.TargetOwner, in.SuccessorRef,
		),
	}
	snapshot, err := s.store.AppendLoopDisposition(ctx, loopID, in.ExpectedRevision, disposition, request)
	if err != nil {
		return OpenLoopView{}, mapStoreError(err)
	}
	return viewFromSnapshot(snapshot), nil
}

func normalizeCreateOpenLoopInput(in CreateOpenLoopInput) CreateOpenLoopInput {
	in.SpaceID = domain.ResponsibilitySpaceID(strings.TrimSpace(in.SpaceID.String()))
	in.Meaning = strings.TrimSpace(in.Meaning)
	in.Owner = strings.TrimSpace(in.Owner)
	in.Trigger = strings.TrimSpace(in.Trigger)
	in.RecheckCondition = strings.TrimSpace(in.RecheckCondition)
	in.SourceRef = strings.TrimSpace(in.SourceRef)
	in.ConfirmationReason = strings.TrimSpace(in.ConfirmationReason)
	in.RequestKey = strings.TrimSpace(in.RequestKey)
	return in
}

func normalizeDispositionInput(in DispositionInput) DispositionInput {
	in.Reason = strings.TrimSpace(in.Reason)
	in.TargetSpaceID = domain.ResponsibilitySpaceID(strings.TrimSpace(in.TargetSpaceID.String()))
	in.TargetOwner = strings.TrimSpace(in.TargetOwner)
	in.SuccessorRef = strings.TrimSpace(in.SuccessorRef)
	in.RequestKey = strings.TrimSpace(in.RequestKey)
	return in
}

func fingerprint(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = fmt.Fprintf(hash, "%d:%s;", len(part), part)
	}
	return fmt.Sprintf("v1:%x", hash.Sum(nil))
}

func viewFromSnapshot(snapshot ports.OpenLoopSnapshot) OpenLoopView {
	return OpenLoopView{
		OpenLoop:     snapshot.OpenLoop,
		Dispositions: append([]domain.LoopDisposition(nil), snapshot.Dispositions...),
		Revision:     snapshot.Revision,
	}
}

func mapStoreError(err error) error {
	var idempotencyConflict *ports.HomeIdempotencyConflictError
	if errors.As(err, &idempotencyConflict) {
		return apierr.Conflict("HOME_IDEMPOTENCY_CONFLICT", "That idempotency key is already associated with different Home input", nil)
	}

	var revisionConflict *ports.OpenLoopRevisionConflictError
	if errors.As(err, &revisionConflict) {
		return apierr.Conflict(
			"OPEN_LOOP_REVISION_CONFLICT",
			fmt.Sprintf("Open Loop moved to revision %s; reload and retry against it", strconv.FormatInt(revisionConflict.CurrentRevision, 10)),
			map[string]any{
				"openLoopId":       revisionConflict.OpenLoopID.String(),
				"expectedRevision": revisionConflict.ExpectedRevision,
				"currentRevision":  revisionConflict.CurrentRevision,
			},
		)
	}
	return err
}
