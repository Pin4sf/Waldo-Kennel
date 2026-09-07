package domain

import (
	"fmt"
	"strings"
	"time"
)

// IntakeSessionID identifies one shared Home/Work understanding flow.
type IntakeSessionID string

// IsZero reports whether the intake identity is blank.
func (id IntakeSessionID) IsZero() bool   { return strings.TrimSpace(string(id)) == "" }
func (id IntakeSessionID) String() string { return string(id) }

// IntakeSourceSurface records where the same shared state machine was opened.
// It changes presentation and provenance, never canonical transition rules.
type IntakeSourceSurface string

const (
	// IntakeSourceHome identifies intake opened from the Home surface.
	IntakeSourceHome IntakeSourceSurface = "home"
	// IntakeSourceWork identifies intake opened from the Work surface.
	IntakeSourceWork IntakeSourceSurface = "work"
)

// Valid reports whether the source surface belongs to the shared contract.
func (source IntakeSourceSurface) Valid() bool {
	return source == IntakeSourceHome || source == IntakeSourceWork
}

// IntakePurpose names the non-canonical responsibility proposal being built.
type IntakePurpose string

const (
	// IntakePurposeNote proposes preserving the statement as a note.
	IntakePurposeNote IntakePurpose = "note"
	// IntakePurposeOpenLoop proposes creating a Home Open Loop.
	IntakePurposeOpenLoop IntakePurpose = "open_loop"
	// IntakePurposeOutcome proposes creating a Work Outcome.
	IntakePurposeOutcome IntakePurpose = "outcome"
	// IntakePurposeResponsibilityLink proposes explicit Home-to-Work lineage.
	IntakePurposeResponsibilityLink IntakePurpose = "responsibility_link"
	// IntakePurposeDismiss proposes consciously releasing the statement.
	IntakePurposeDismiss IntakePurpose = "dismiss"
)

// Valid reports whether the purpose is supported by shared intake.
func (purpose IntakePurpose) Valid() bool {
	switch purpose {
	case IntakePurposeNote, IntakePurposeOpenLoop, IntakePurposeOutcome,
		IntakePurposeResponsibilityLink, IntakePurposeDismiss:
		return true
	default:
		return false
	}
}

// IntakeStatus is the durable state of a shared adaptive intake. Confirmed and
// cancelled are terminal. AnalysisFailed preserves the original statement and
// may be retried or manually revised.
type IntakeStatus string

const (
	// IntakeStatusCaptured is the initial durable state.
	IntakeStatusCaptured IntakeStatus = "captured"
	// IntakeStatusAnalyzing records bounded proposal analysis in progress.
	IntakeStatusAnalyzing IntakeStatus = "analyzing"
	// IntakeStatusNeedsUser records the single material-question boundary.
	IntakeStatusNeedsUser IntakeStatus = "needs_user"
	// IntakeStatusReady records an editable proposal awaiting confirmation.
	IntakeStatusReady IntakeStatus = "ready"
	// IntakeStatusConfirmed records canonical Outcome creation.
	IntakeStatusConfirmed IntakeStatus = "confirmed"
	// IntakeStatusAnalysisFailed preserves a retryable analysis failure.
	IntakeStatusAnalysisFailed IntakeStatus = "analysis_failed"
	// IntakeStatusCancelled records conscious release without responsibility creation.
	IntakeStatusCancelled IntakeStatus = "cancelled"
)

// Valid reports whether the status belongs to the intake state machine.
func (status IntakeStatus) Valid() bool {
	switch status {
	case IntakeStatusCaptured, IntakeStatusAnalyzing, IntakeStatusNeedsUser,
		IntakeStatusReady, IntakeStatusConfirmed, IntakeStatusAnalysisFailed,
		IntakeStatusCancelled:
		return true
	default:
		return false
	}
}

// CanTransitionIntake is the single Home/Work state-transition contract.
func CanTransitionIntake(from, to IntakeStatus) bool {
	switch from {
	case IntakeStatusCaptured:
		return to == IntakeStatusAnalyzing || to == IntakeStatusCancelled
	case IntakeStatusAnalyzing:
		return to == IntakeStatusNeedsUser || to == IntakeStatusReady || to == IntakeStatusAnalysisFailed
	case IntakeStatusNeedsUser:
		return to == IntakeStatusAnalyzing || to == IntakeStatusCancelled
	case IntakeStatusReady:
		return to == IntakeStatusAnalyzing || to == IntakeStatusConfirmed || to == IntakeStatusCancelled
	case IntakeStatusAnalysisFailed:
		return to == IntakeStatusAnalyzing || to == IntakeStatusCancelled
	default:
		return false
	}
}

// IntakeSession preserves exact user intent and proposal lineage. Conversation
// bodies deliberately do not belong here; IntakeConversationRef stores only
// provenance identifiers.
type IntakeSession struct {
	ID                      IntakeSessionID
	SourceSurface           IntakeSourceSurface
	Purpose                 IntakePurpose
	ProjectID               ProjectID
	SourceOpenLoopID        OpenLoopID
	Statement               string
	Status                  IntakeStatus
	CurrentProposalRevision int64
	ClarificationCount      int64
	ConfirmedOutcomeID      OutcomeID
	FailureCode             string
	CancellationReason      string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// Validate checks the durable invariants for an intake session.
func (session IntakeSession) Validate() error {
	if session.ID.IsZero() {
		return fmt.Errorf("intake session id is required")
	}
	if !session.SourceSurface.Valid() {
		return fmt.Errorf("unsupported intake source surface %q", session.SourceSurface)
	}
	if !session.Purpose.Valid() {
		return fmt.Errorf("unsupported intake purpose %q", session.Purpose)
	}
	if session.Purpose == IntakePurposeOutcome && strings.TrimSpace(string(session.ProjectID)) == "" {
		return fmt.Errorf("outcome intake project id is required")
	}
	if strings.TrimSpace(session.Statement) == "" {
		return fmt.Errorf("intake statement is required")
	}
	if !session.Status.Valid() {
		return fmt.Errorf("unsupported intake status %q", session.Status)
	}
	if session.CurrentProposalRevision < 0 {
		return fmt.Errorf("intake proposal revision must not be negative")
	}
	if session.ClarificationCount < 0 || session.ClarificationCount > 1 {
		return fmt.Errorf("intake permits at most one material clarification")
	}
	if session.Status == IntakeStatusConfirmed && session.ConfirmedOutcomeID.IsZero() {
		return fmt.Errorf("confirmed outcome intake requires an outcome id")
	}
	if session.Status != IntakeStatusConfirmed && !session.ConfirmedOutcomeID.IsZero() {
		return fmt.Errorf("unconfirmed intake cannot reference a confirmed outcome")
	}
	if session.CreatedAt.IsZero() || session.UpdatedAt.IsZero() {
		return fmt.Errorf("intake timestamps are required")
	}
	return nil
}

// CanAskMaterialClarification enforces the product-wide one-question bound.
func (session IntakeSession) CanAskMaterialClarification() bool {
	return session.ClarificationCount == 0 && session.Status != IntakeStatusConfirmed && session.Status != IntakeStatusCancelled
}

// IntakeConversationRef references a durable conversation episode/turn for
// provenance. It intentionally has no transcript, message, or content field.
type IntakeConversationRef struct {
	EpisodeID string
	TurnID    string
	Position  int64
}

// Validate checks that a provenance reference contains identifiers only.
func (ref IntakeConversationRef) Validate() error {
	if strings.TrimSpace(ref.EpisodeID) == "" {
		return fmt.Errorf("conversation episode id is required")
	}
	if strings.TrimSpace(ref.TurnID) == "" {
		return fmt.Errorf("conversation turn id is required")
	}
	if ref.Position < 1 {
		return fmt.Errorf("conversation reference position must be at least 1")
	}
	return nil
}

// ClarificationRequestID identifies the single material clarification.
type ClarificationRequestID string

// ClarificationRequest is one genuinely material question. Alternatives are
// bounded structured choices; free-text remains available at the UI/API seam.
type ClarificationRequest struct {
	ID                  ClarificationRequestID
	IntakeID            IntakeSessionID
	Question            string
	Reason              string
	Recommendation      string
	Alternatives        []string
	DeferralConsequence string
	Answer              string
	CreatedAt           time.Time
	AnsweredAt          *time.Time
}

// Validate checks the bounded clarification contract.
func (request ClarificationRequest) Validate() error {
	if strings.TrimSpace(string(request.ID)) == "" || request.IntakeID.IsZero() {
		return fmt.Errorf("clarification identity is required")
	}
	if strings.TrimSpace(request.Question) == "" || strings.TrimSpace(request.Reason) == "" {
		return fmt.Errorf("clarification question and reason are required")
	}
	if len(request.Alternatives) > 3 {
		return fmt.Errorf("clarification permits at most three alternatives")
	}
	if request.CreatedAt.IsZero() {
		return fmt.Errorf("clarification created time is required")
	}
	if request.AnsweredAt != nil && strings.TrimSpace(request.Answer) == "" {
		return fmt.Errorf("answered clarification requires an answer")
	}
	return nil
}

// ProposalRevisionID identifies one immutable proposal revision.
type ProposalRevisionID string

// ProposedCriterionID preserves criterion identity across proposal edits.
type ProposedCriterionID string

// ProposedCriterion is a stable criterion plus its expected evidence.
type ProposedCriterion struct {
	ID               ProposedCriterionID
	Text             string
	EvidenceExpected []string
}

// Validate checks the proposed criterion and evidence contract.
func (criterion ProposedCriterion) Validate() error {
	if strings.TrimSpace(string(criterion.ID)) == "" || strings.TrimSpace(criterion.Text) == "" {
		return fmt.Errorf("proposal criterion identity and text are required")
	}
	if len(criterion.EvidenceExpected) == 0 {
		return fmt.Errorf("proposal criterion requires expected evidence")
	}
	for _, expected := range criterion.EvidenceExpected {
		if strings.TrimSpace(expected) == "" {
			return fmt.Errorf("proposal expected evidence is blank")
		}
	}
	return nil
}

// ProposedAuthority is a typed least-privilege ceiling. It is a proposal only
// until the Contract and later Plan/CapabilityGrant boundaries are confirmed.
type ProposedAuthority struct {
	ReadWorkspace  bool
	WriteWorkspace bool
	ExecuteLocal   bool
	UseNetwork     bool
	CommitLocal    bool
	CreatePR       bool
	Deploy         bool
	ExternalEffect bool
}

// ContractFacetKind names an adaptive typed facet of the stable core.
type ContractFacetKind string

const (
	// ContractFacetSoftware identifies software delivery requirements.
	ContractFacetSoftware ContractFacetKind = "software"
	// ContractFacetResearch identifies research requirements.
	ContractFacetResearch ContractFacetKind = "research"
	// ContractFacetDesign identifies design requirements.
	ContractFacetDesign ContractFacetKind = "design"
	// ContractFacetDocumentation identifies documentation requirements.
	ContractFacetDocumentation ContractFacetKind = "documentation"
	// ContractFacetInvestigation identifies investigation requirements.
	ContractFacetInvestigation ContractFacetKind = "investigation"
	// ContractFacetEvaluation identifies evaluation requirements.
	ContractFacetEvaluation ContractFacetKind = "evaluation"
	// ContractFacetOperations identifies operational requirements.
	ContractFacetOperations ContractFacetKind = "operations"
)

// Valid reports whether the adaptive facet kind is supported.
func (kind ContractFacetKind) Valid() bool {
	switch kind {
	case ContractFacetSoftware, ContractFacetResearch, ContractFacetDesign,
		ContractFacetDocumentation, ContractFacetInvestigation,
		ContractFacetEvaluation, ContractFacetOperations:
		return true
	default:
		return false
	}
}

// ContractFacet is adaptive in kind but stable and typed in shape.
type ContractFacet struct {
	Kind         ContractFacetKind
	Summary      string
	Requirements []string
}

// Validate checks the typed facet contract.
func (facet ContractFacet) Validate() error {
	if !facet.Kind.Valid() {
		return fmt.Errorf("unsupported contract facet kind %q", facet.Kind)
	}
	if strings.TrimSpace(facet.Summary) == "" {
		return fmt.Errorf("contract facet summary is required")
	}
	return nil
}

// OutcomeContractProposal is immutable, non-canonical analysis output. User
// confirmation compiles its stable core into one Outcome/ContractRevision.
type OutcomeContractProposal struct {
	ID                 ProposalRevisionID
	IntakeID           IntakeSessionID
	Revision           int64
	Title              string
	DesiredState       string
	Criteria           []ProposedCriterion
	ReviewMethod       string
	Constraints        []string
	NonGoals           []string
	AuthorityCeiling   ProposedAuthority
	StopConditions     []string
	ClarificationNotes []string
	TemporalCondition  *string
	Facets             []ContractFacet
	CreatedAt          time.Time
}

// Validate checks the immutable proposal and its stable criterion identities.
func (proposal OutcomeContractProposal) Validate() error {
	if strings.TrimSpace(string(proposal.ID)) == "" || proposal.IntakeID.IsZero() {
		return fmt.Errorf("proposal identity is required")
	}
	if proposal.Revision < 1 {
		return fmt.Errorf("proposal revision must be at least 1")
	}
	if strings.TrimSpace(proposal.Title) == "" || strings.TrimSpace(proposal.DesiredState) == "" {
		return fmt.Errorf("proposal title and desired state are required")
	}
	if len(proposal.Criteria) == 0 {
		return fmt.Errorf("proposal requires at least one success criterion")
	}
	seen := make(map[ProposedCriterionID]struct{}, len(proposal.Criteria))
	for _, criterion := range proposal.Criteria {
		if err := criterion.Validate(); err != nil {
			return err
		}
		if _, ok := seen[criterion.ID]; ok {
			return fmt.Errorf("proposal criterion id %s is duplicated", criterion.ID)
		}
		seen[criterion.ID] = struct{}{}
	}
	if strings.TrimSpace(proposal.ReviewMethod) == "" {
		return fmt.Errorf("proposal review method is required")
	}
	if len(proposal.StopConditions) == 0 {
		return fmt.Errorf("proposal requires at least one stop condition")
	}
	for _, facet := range proposal.Facets {
		if err := facet.Validate(); err != nil {
			return err
		}
	}
	if proposal.TemporalCondition != nil && strings.TrimSpace(*proposal.TemporalCondition) == "" {
		return fmt.Errorf("proposal temporal condition is blank")
	}
	if proposal.CreatedAt.IsZero() {
		return fmt.Errorf("proposal created time is required")
	}
	return nil
}
