package domain

import (
	"fmt"
	"strings"
	"time"
)

// EvidenceItemID identifies one immutable EvidenceItem.
type EvidenceItemID string

// VerificationRunID identifies one immutable VerificationRun.
type VerificationRunID string

// AcceptanceDecisionID identifies one immutable AcceptanceDecision.
type AcceptanceDecisionID string

// OutcomeCorrectionID identifies one immutable OutcomeCorrection.
type OutcomeCorrectionID string

// ProofSubjectType names the exact lineage object whose revision is proved.
type ProofSubjectType string

// Proof subject types cover the current Outcome execution lineage.
const (
	ProofSubjectOutcome  ProofSubjectType = "outcome"
	ProofSubjectContract ProofSubjectType = "contract"
	ProofSubjectPlan     ProofSubjectType = "plan"
	ProofSubjectWorkUnit ProofSubjectType = "work_unit"
	ProofSubjectAttempt  ProofSubjectType = "attempt"
)

func (t ProofSubjectType) valid() bool {
	switch t {
	case ProofSubjectOutcome, ProofSubjectContract, ProofSubjectPlan, ProofSubjectWorkUnit, ProofSubjectAttempt:
		return true
	default:
		return false
	}
}

// EvidenceKind distinguishes support from contradiction.
type EvidenceKind string

// Evidence kinds remain factual and never imply acceptance.
const (
	EvidenceSupporting    EvidenceKind = "supporting"
	EvidenceContradicting EvidenceKind = "contradicting"
)

// EvidenceSourceType records how Evidence was obtained.
type EvidenceSourceType string

// Evidence source types preserve provenance without upgrading trust.
const (
	EvidenceSourceArtifact           EvidenceSourceType = "artifact"
	EvidenceSourceDeterministicCheck EvidenceSourceType = "deterministic_check"
	EvidenceSourceProviderOutput     EvidenceSourceType = "provider_output"
	EvidenceSourceOwnerWalkthrough   EvidenceSourceType = "owner_walkthrough"
)

// EvidenceProducerType records who or what produced Evidence.
type EvidenceProducerType string

// Evidence producer types deliberately exclude acceptance authority.
const (
	EvidenceProducerUser     EvidenceProducerType = "user"
	EvidenceProducerProvider EvidenceProducerType = "provider"
	EvidenceProducerTool     EvidenceProducerType = "tool"
)

// EvidenceItem is immutable provenance-bearing support or contradiction for
// one exact criterion and subject revision.
type EvidenceItem struct {
	ID                 EvidenceItemID
	OutcomeID          OutcomeID
	ContractRevisionID ContractRevisionID
	CriterionID        CriterionID
	SubjectType        ProofSubjectType
	SubjectID          string
	SubjectRevision    string
	Kind               EvidenceKind
	SourceType         EvidenceSourceType
	SourceRef          string
	ProducerType       EvidenceProducerType
	ProducerRef        string
	Summary            string
	ContentDigest      string
	RequestKey         string
	RequestFingerprint string
	CreatedAt          time.Time
}

// Validate enforces immutable Evidence identity, provenance, and binding.
func (e EvidenceItem) Validate() error {
	switch {
	case strings.TrimSpace(string(e.ID)) == "":
		return fmt.Errorf("evidence item id is required")
	case e.OutcomeID.IsZero():
		return fmt.Errorf("evidence outcome id is required")
	case e.ContractRevisionID.IsZero():
		return fmt.Errorf("evidence contract revision id is required")
	case e.CriterionID.IsZero():
		return fmt.Errorf("evidence criterion id is required")
	case !e.SubjectType.valid():
		return fmt.Errorf("evidence subject type is invalid")
	case strings.TrimSpace(e.SubjectID) == "":
		return fmt.Errorf("evidence subject id is required")
	case strings.TrimSpace(e.SubjectRevision) == "":
		return fmt.Errorf("evidence subject revision is required")
	case e.Kind != EvidenceSupporting && e.Kind != EvidenceContradicting:
		return fmt.Errorf("evidence kind is invalid")
	case strings.TrimSpace(string(e.SourceType)) == "":
		return fmt.Errorf("evidence source type is required")
	case e.SourceType != EvidenceSourceArtifact && e.SourceType != EvidenceSourceDeterministicCheck && e.SourceType != EvidenceSourceProviderOutput && e.SourceType != EvidenceSourceOwnerWalkthrough:
		return fmt.Errorf("evidence source type is invalid")
	case strings.TrimSpace(e.SourceRef) == "":
		return fmt.Errorf("evidence source provenance is required")
	case e.ProducerType != EvidenceProducerUser && e.ProducerType != EvidenceProducerProvider && e.ProducerType != EvidenceProducerTool:
		return fmt.Errorf("evidence producer type is invalid")
	case strings.TrimSpace(e.ProducerRef) == "":
		return fmt.Errorf("evidence producer reference is required")
	case strings.TrimSpace(e.Summary) == "":
		return fmt.Errorf("evidence summary is required")
	case !isSHA256Hex(e.ContentDigest):
		return fmt.Errorf("evidence content digest must be 64 hexadecimal characters")
	case strings.TrimSpace(e.RequestKey) == "":
		return fmt.Errorf("evidence request key is required")
	case !isSHA256Hex(e.RequestFingerprint):
		return fmt.Errorf("evidence request fingerprint must be 64 hexadecimal characters")
	}
	return nil
}

// VerificationIndependenceClass declares the verifier-producer relationship.
type VerificationIndependenceClass string

// Verification independence classes describe actual, not inferred, independence.
const (
	VerificationDeterministic     VerificationIndependenceClass = "deterministic"
	VerificationProducerSelfCheck VerificationIndependenceClass = "producer_self_check"
	VerificationSeparateSession   VerificationIndependenceClass = "separate_session"
	VerificationCrossProvider     VerificationIndependenceClass = "cross_provider"
	VerificationOwnerWalkthrough  VerificationIndependenceClass = "owner_walkthrough"
)

// VerificationResult records the verifier's factual result.
type VerificationResult string

// Verification results do not themselves accept an Outcome.
const (
	VerificationPassed       VerificationResult = "passed"
	VerificationFailed       VerificationResult = "failed"
	VerificationInconclusive VerificationResult = "inconclusive"
	VerificationException    VerificationResult = "exception"
)

// VerificationRun records what was checked and the actual relationship
// between producer and verifier. It never accepts an Outcome.
type VerificationRun struct {
	ID                 VerificationRunID
	OutcomeID          OutcomeID
	ContractRevisionID ContractRevisionID
	CriterionID        CriterionID
	SubjectType        ProofSubjectType
	SubjectID          string
	SubjectRevision    string
	EvidenceItemIDs    []EvidenceItemID
	Method             string
	IndependenceClass  VerificationIndependenceClass
	Result             VerificationResult
	ProducerRef        string
	VerifierRef        string
	ProducerProvider   string
	VerifierProvider   string
	Detail             string
	RequestKey         string
	RequestFingerprint string
	CreatedAt          time.Time
}

// Validate enforces exact Evidence binding and truthful independence claims.
func (v VerificationRun) Validate() error {
	switch {
	case strings.TrimSpace(string(v.ID)) == "":
		return fmt.Errorf("verification run id is required")
	case v.OutcomeID.IsZero() || v.ContractRevisionID.IsZero() || v.CriterionID.IsZero():
		return fmt.Errorf("verification outcome, contract revision, and criterion are required")
	case !v.SubjectType.valid() || strings.TrimSpace(v.SubjectID) == "" || strings.TrimSpace(v.SubjectRevision) == "":
		return fmt.Errorf("verification exact subject revision is required")
	case len(v.EvidenceItemIDs) == 0:
		return fmt.Errorf("verification requires bound evidence")
	case strings.TrimSpace(v.Method) == "":
		return fmt.Errorf("verification method is required")
	case v.Result != VerificationPassed && v.Result != VerificationFailed && v.Result != VerificationInconclusive && v.Result != VerificationException:
		return fmt.Errorf("verification result is invalid")
	case strings.TrimSpace(v.VerifierRef) == "":
		return fmt.Errorf("verification verifier reference is required")
	case strings.TrimSpace(v.RequestKey) == "" || !isSHA256Hex(v.RequestFingerprint):
		return fmt.Errorf("verification idempotency identity is required")
	}
	switch v.IndependenceClass {
	case VerificationDeterministic, VerificationOwnerWalkthrough:
	case VerificationProducerSelfCheck:
		if strings.TrimSpace(v.ProducerRef) == "" || v.VerifierRef != v.ProducerRef {
			return fmt.Errorf("producer self-check must identify the producer as verifier")
		}
	case VerificationSeparateSession:
		if strings.TrimSpace(v.ProducerRef) == "" || v.VerifierRef == v.ProducerRef {
			return fmt.Errorf("separate-session review requires a different verifier session")
		}
	case VerificationCrossProvider:
		if strings.TrimSpace(v.ProducerProvider) == "" || strings.TrimSpace(v.VerifierProvider) == "" || v.ProducerProvider == v.VerifierProvider {
			return fmt.Errorf("cross-provider review requires a different provider")
		}
	default:
		return fmt.Errorf("verification independence class is invalid")
	}
	if v.Result == VerificationException && strings.TrimSpace(v.Detail) == "" {
		return fmt.Errorf("verification exception requires detail")
	}
	return nil
}

// IsIndependent reports whether the declared class is outside producer self-check.
func (v VerificationRun) IsIndependent() bool {
	return v.IndependenceClass != VerificationProducerSelfCheck
}

// AcceptanceDecisionKind names the user's explicit closure or re-entry decision.
type AcceptanceDecisionKind string

// Acceptance decision kinds keep acceptance separate from rework and reopen.
const (
	AcceptanceAccept        AcceptanceDecisionKind = "accept"
	AcceptanceRequestRework AcceptanceDecisionKind = "request_rework"
	AcceptanceReopen        AcceptanceDecisionKind = "reopen"
)

// AcceptanceActorType is intentionally restricted to the user.
type AcceptanceActorType string

// AcceptanceActorUser is the only actor allowed to accept or reopen an Outcome.
const AcceptanceActorUser AcceptanceActorType = "user"

// ResourceDisposition records what should happen to execution resources.
type ResourceDisposition string

// Resource disposition choices are explicit owner instructions.
const (
	ResourceDispositionRetain        ResourceDisposition = "retain"
	ResourceDispositionCleanupLater  ResourceDisposition = "cleanup_later"
	ResourceDispositionNotApplicable ResourceDisposition = "not_applicable"
)

// AcceptanceDecision is an immutable explicit user decision. ActorType has no
// provider/model value by design.
type AcceptanceDecision struct {
	ID                  AcceptanceDecisionID
	OutcomeID           OutcomeID
	ContractRevisionID  ContractRevisionID
	Kind                AcceptanceDecisionKind
	ActorType           AcceptanceActorType
	Summary             string
	ResourceDisposition ResourceDisposition
	RequestKey          string
	RequestFingerprint  string
	CreatedAt           time.Time
}

// Validate enforces explicit user authority and decision identity.
func (d AcceptanceDecision) Validate() error {
	switch {
	case strings.TrimSpace(string(d.ID)) == "":
		return fmt.Errorf("acceptance decision id is required")
	case d.OutcomeID.IsZero() || d.ContractRevisionID.IsZero():
		return fmt.Errorf("acceptance outcome and contract revision are required")
	case d.Kind != AcceptanceAccept && d.Kind != AcceptanceRequestRework && d.Kind != AcceptanceReopen:
		return fmt.Errorf("acceptance decision kind is invalid")
	case d.ActorType != AcceptanceActorUser:
		return fmt.Errorf("only the user may create an acceptance decision")
	case strings.TrimSpace(d.Summary) == "":
		return fmt.Errorf("acceptance decision summary is required")
	case d.ResourceDisposition != ResourceDispositionRetain && d.ResourceDisposition != ResourceDispositionCleanupLater && d.ResourceDisposition != ResourceDispositionNotApplicable:
		return fmt.Errorf("acceptance resource disposition is invalid")
	case strings.TrimSpace(d.RequestKey) == "" || !isSHA256Hex(d.RequestFingerprint):
		return fmt.Errorf("acceptance idempotency identity is required")
	}
	return nil
}

// ReentryTargetType names the lineage seam that must change next.
type ReentryTargetType string

// Re-entry targets cover the bounded Work lineage owned by this slice.
const (
	ReentryTargetAttempt  ReentryTargetType = "attempt"
	ReentryTargetWorkUnit ReentryTargetType = "work_unit"
	ReentryTargetPlan     ReentryTargetType = "plan"
	ReentryTargetContract ReentryTargetType = "contract"
)

// OutcomeCorrection is the explicit re-entry lineage created by request
// rework or reopen. It points at the seam that must change next.
type OutcomeCorrection struct {
	ID                 OutcomeCorrectionID
	DecisionID         AcceptanceDecisionID
	OutcomeID          OutcomeID
	ContractRevisionID ContractRevisionID
	Feedback           string
	TargetType         ReentryTargetType
	TargetID           string
	CreatedAt          time.Time
}

// Validate enforces explicit correction feedback and a concrete re-entry target.
func (c OutcomeCorrection) Validate() error {
	switch {
	case strings.TrimSpace(string(c.ID)) == "" || strings.TrimSpace(string(c.DecisionID)) == "":
		return fmt.Errorf("correction and decision ids are required")
	case c.OutcomeID.IsZero() || c.ContractRevisionID.IsZero():
		return fmt.Errorf("correction outcome and contract revision are required")
	case strings.TrimSpace(c.Feedback) == "":
		return fmt.Errorf("correction feedback is required")
	case c.TargetType != ReentryTargetAttempt && c.TargetType != ReentryTargetWorkUnit && c.TargetType != ReentryTargetPlan && c.TargetType != ReentryTargetContract:
		return fmt.Errorf("correction re-entry target type is invalid")
	case strings.TrimSpace(c.TargetID) == "":
		return fmt.Errorf("correction re-entry target id is required")
	}
	return nil
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}
