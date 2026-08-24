package domain

import (
	"fmt"
	"strings"
	"time"
)

// OpenLoopID identifies one confirmed unresolved Home responsibility.
type OpenLoopID string

// IsZero reports whether the id is unset or blank.
func (id OpenLoopID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// String returns the raw identifier value.
func (id OpenLoopID) String() string {
	return string(id)
}

// OpenLoopConfirmation records the user action that made a candidate a
// canonical responsibility. Inference and provider completion are deliberately
// not valid confirmation modes.
type OpenLoopConfirmation string

const (
	// OpenLoopConfirmationExplicit is a separate confirm action on a proposal.
	OpenLoopConfirmationExplicit OpenLoopConfirmation = "explicit"
	// OpenLoopConfirmationDirectCommand is an unambiguous user command whose
	// requested effect is to create the Open Loop.
	OpenLoopConfirmationDirectCommand OpenLoopConfirmation = "direct_command"
)

// Valid reports whether the confirmation is an admitted user action.
func (confirmation OpenLoopConfirmation) Valid() bool {
	switch confirmation {
	case OpenLoopConfirmationExplicit, OpenLoopConfirmationDirectCommand:
		return true
	}
	return false
}

// OpenLoopCandidate is proposed responsibility, not canonical responsibility.
// It may come from Quick Capture, intake, or a source observation and remains a
// candidate until Confirm receives an admitted user confirmation.
type OpenLoopCandidate struct {
	Meaning string
}

// NewOpenLoopCandidate creates a non-canonical proposal. Candidate validation
// happens at confirmation so intake can preserve an incomplete draft.
func NewOpenLoopCandidate(meaning string) OpenLoopCandidate {
	return OpenLoopCandidate{Meaning: meaning}
}

// IsCanonical is always false for a candidate.
func (OpenLoopCandidate) IsCanonical() bool {
	return false
}

// ConfirmOpenLoopInput supplies the durable identity and provenance needed to
// turn one candidate into a confirmed Open Loop.
type ConfirmOpenLoopInput struct {
	ID               OpenLoopID
	SpaceID          ResponsibilitySpaceID
	Owner            string
	Trigger          string
	RecheckCondition string
	SourceRef        string
	Confirmation     OpenLoopConfirmation
	ConfirmedAt      time.Time
}

// Confirm creates a canonical Open Loop only from explicit confirmation or an
// unambiguous direct command.
func (candidate OpenLoopCandidate) Confirm(input ConfirmOpenLoopInput) (OpenLoop, error) {
	if input.Confirmation == "" {
		return OpenLoop{}, fmt.Errorf("open loop confirmation is required")
	}
	if !input.Confirmation.Valid() {
		return OpenLoop{}, fmt.Errorf("unsupported open loop confirmation %q", input.Confirmation)
	}

	loop := OpenLoop{
		ID:               input.ID,
		SpaceID:          input.SpaceID,
		Meaning:          candidate.Meaning,
		Owner:            input.Owner,
		Trigger:          input.Trigger,
		RecheckCondition: input.RecheckCondition,
		SourceRef:        input.SourceRef,
		Confirmation:     input.Confirmation,
		ConfirmedAt:      input.ConfirmedAt,
	}
	if err := loop.Validate(); err != nil {
		return OpenLoop{}, err
	}
	return loop, nil
}

// OpenLoop is one confirmed unresolved responsibility in Personal Home. It has
// no stored lifecycle status: the current view is derived from immutable
// LoopDispositions.
type OpenLoop struct {
	ID               OpenLoopID
	SpaceID          ResponsibilitySpaceID
	Meaning          string
	Owner            string
	Trigger          string
	RecheckCondition string
	SourceRef        string
	Confirmation     OpenLoopConfirmation
	ConfirmedAt      time.Time
}

// IsCanonical reports whether the Open Loop carries all intrinsic facts needed
// for canonical responsibility.
func (loop OpenLoop) IsCanonical() bool {
	return loop.Validate() == nil
}

// Validate checks intrinsic Open Loop invariants. Space ownership, idempotency,
// and revision conflicts are enforced by service/storage transactions.
func (loop OpenLoop) Validate() error {
	if loop.ID.IsZero() {
		return fmt.Errorf("open loop id is required")
	}
	if loop.SpaceID.IsZero() {
		return fmt.Errorf("open loop responsibility space id is required")
	}
	if strings.TrimSpace(loop.Meaning) == "" {
		return fmt.Errorf("open loop meaning is required")
	}
	if strings.TrimSpace(loop.Owner) == "" {
		return fmt.Errorf("open loop owner is required")
	}
	if strings.TrimSpace(loop.Trigger) == "" {
		return fmt.Errorf("open loop trigger is required")
	}
	if strings.TrimSpace(loop.RecheckCondition) == "" {
		return fmt.Errorf("open loop recheck condition is required")
	}
	if strings.TrimSpace(loop.SourceRef) == "" {
		return fmt.Errorf("open loop source reference is required")
	}
	if !loop.Confirmation.Valid() {
		return fmt.Errorf("unsupported open loop confirmation %q", loop.Confirmation)
	}
	if loop.ConfirmedAt.IsZero() {
		return fmt.Errorf("open loop confirmation time is required")
	}
	return nil
}

// LoopDispositionID identifies one immutable owner decision about an Open Loop.
type LoopDispositionID string

// IsZero reports whether the id is unset or blank.
func (id LoopDispositionID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// LoopDispositionKind names the native Home lifecycle decision.
type LoopDispositionKind string

const (
	LoopDispositionConfirm   LoopDispositionKind = "confirm"
	LoopDispositionClose     LoopDispositionKind = "close"
	LoopDispositionRelease   LoopDispositionKind = "release"
	LoopDispositionReopen    LoopDispositionKind = "reopen"
	LoopDispositionTransfer  LoopDispositionKind = "transfer"
	LoopDispositionSupersede LoopDispositionKind = "supersede"
)

// Valid reports whether kind is a supported native disposition.
func (kind LoopDispositionKind) Valid() bool {
	switch kind {
	case LoopDispositionConfirm, LoopDispositionClose, LoopDispositionRelease,
		LoopDispositionReopen, LoopDispositionTransfer, LoopDispositionSupersede:
		return true
	}
	return false
}

// LoopDispositionActor records the authority that created a disposition. Only
// the owner is admitted. Provider exists as an explicit invalid value so
// adapters cannot reinterpret provider completion as a user decision.
type LoopDispositionActor string

const (
	LoopDispositionActorOwner    LoopDispositionActor = "owner"
	LoopDispositionActorProvider LoopDispositionActor = "provider"
)

// LoopDisposition is one immutable owner decision. Transfer and supersede keep
// their target lineage on the disposition rather than rewriting the Open Loop.
type LoopDisposition struct {
	ID            LoopDispositionID
	OpenLoopID    OpenLoopID
	Kind          LoopDispositionKind
	Actor         LoopDispositionActor
	Reason        string
	TargetSpaceID ResponsibilitySpaceID
	TargetOwner   string
	SuccessorRef  string
	OccurredAt    time.Time
}

// Validate checks intrinsic disposition invariants. Append-only behavior and
// lifecycle transition ordering are enforced by storage and the Home service.
func (disposition LoopDisposition) Validate() error {
	if disposition.ID.IsZero() {
		return fmt.Errorf("loop disposition id is required")
	}
	if disposition.OpenLoopID.IsZero() {
		return fmt.Errorf("loop disposition open loop id is required")
	}
	if !disposition.Kind.Valid() {
		return fmt.Errorf("unsupported loop disposition %q", disposition.Kind)
	}
	if disposition.Actor != LoopDispositionActorOwner {
		return fmt.Errorf("loop disposition requires an owner decision")
	}
	if strings.TrimSpace(disposition.Reason) == "" {
		return fmt.Errorf("loop disposition reason is required")
	}
	if disposition.OccurredAt.IsZero() {
		return fmt.Errorf("loop disposition occurred time is required")
	}

	if disposition.Kind == LoopDispositionTransfer {
		if disposition.TargetSpaceID.IsZero() {
			return fmt.Errorf("transfer target space is required")
		}
		if strings.TrimSpace(disposition.TargetOwner) == "" {
			return fmt.Errorf("transfer target owner is required")
		}
	} else if !disposition.TargetSpaceID.IsZero() || strings.TrimSpace(disposition.TargetOwner) != "" {
		return fmt.Errorf("transfer target is valid only for transfer disposition")
	}

	if disposition.Kind == LoopDispositionSupersede {
		if strings.TrimSpace(disposition.SuccessorRef) == "" {
			return fmt.Errorf("supersede successor reference is required")
		}
	} else if strings.TrimSpace(disposition.SuccessorRef) != "" {
		return fmt.Errorf("successor reference is valid only for supersede disposition")
	}
	return nil
}
