package outcome

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
)

// ProofManager is the controller-facing Work E boundary. Only
// DecideAcceptance can append owner acceptance; Evidence and Verification are
// deliberately separate facts.
type ProofManager interface {
	GetProof(context.Context, domain.OutcomeID) (ProofView, error)
	RecordEvidence(context.Context, domain.OutcomeID, RecordEvidenceInput) (ProofView, error)
	RecordVerification(context.Context, domain.OutcomeID, RecordVerificationInput) (ProofView, error)
	DecideAcceptance(context.Context, domain.OutcomeID, DecideAcceptanceInput) (ProofView, error)

	// BatchEligibility reports which contributing Outcomes could be accepted
	// together right now, and why the others could not. It decides nothing.
	BatchEligibility(context.Context, domain.OutcomeID) ([]domain.BatchEntryVerdict, error)

	// AcceptContributorBatch records one owner sitting as N separate
	// immutable decisions (ADR 0007). The daemon may withhold a contributor
	// from the batch; it may never accept one.
	AcceptContributorBatch(context.Context, domain.OutcomeID, AcceptBatchInput) (AcceptBatchView, error)
}

// ProofStatus is derived from durable proof facts at read time.
type ProofStatus string

// Proof statuses never store or infer provider completion as acceptance.
const (
	ProofStatusActive             ProofStatus = "active"
	ProofStatusReadyForAcceptance ProofStatus = "ready_for_acceptance"
	ProofStatusAccepted           ProofStatus = "accepted"
	ProofStatusReworkRequired     ProofStatus = "rework_required"
)

// CriterionProofView groups current-revision facts for one stable criterion.
type CriterionProofView struct {
	Criterion     domain.ContractCriterion
	Evidence      []domain.EvidenceItem
	Verifications []domain.VerificationRun
	Ready         bool
	Gap           string
	// Delegated reports that contributing Outcomes prove this criterion rather
	// than the parent's own Evidence (ADR 0007). ClaimedBy names them.
	Delegated bool               `json:"delegated,omitempty"`
	ClaimedBy []domain.OutcomeID `json:"claimedBy,omitempty"`
}

// ProofView is the daemon-derived Prove & Close read model.
type ProofView struct {
	OutcomeID    domain.OutcomeID
	Contract     domain.ContractRevision
	Status       ProofStatus
	NextAction   string
	Criteria     []CriterionProofView
	Decisions    []domain.AcceptanceDecision
	Corrections  []domain.OutcomeCorrection
	ProofHorizon time.Time
}

// RecordEvidenceInput binds immutable Evidence to an exact criterion and subject revision.
type RecordEvidenceInput struct {
	ExpectedContractRevision int64
	ContractRevisionID       domain.ContractRevisionID
	CriterionID              domain.CriterionID
	SubjectType              domain.ProofSubjectType
	SubjectID                string
	SubjectRevision          string
	Kind                     domain.EvidenceKind
	SourceType               domain.EvidenceSourceType
	SourceRef                string
	ProducerType             domain.EvidenceProducerType
	ProducerRef              string
	Summary                  string
	ContentDigest            string
	RequestKey               string
}

// RecordVerificationInput binds a declared verification method to exact Evidence.
type RecordVerificationInput struct {
	ExpectedContractRevision int64
	ContractRevisionID       domain.ContractRevisionID
	CriterionID              domain.CriterionID
	SubjectType              domain.ProofSubjectType
	SubjectID                string
	SubjectRevision          string
	EvidenceItemIDs          []domain.EvidenceItemID
	Method                   string
	IndependenceClass        domain.VerificationIndependenceClass
	Result                   domain.VerificationResult
	ProducerRef              string
	VerifierRef              string
	ProducerProvider         string
	VerifierProvider         string
	Detail                   string
	RequestKey               string
}

// DecideAcceptanceInput records the user's explicit acceptance or re-entry decision.
type DecideAcceptanceInput struct {
	ExpectedContractRevision int64
	ContractRevisionID       domain.ContractRevisionID
	Kind                     domain.AcceptanceDecisionKind
	Summary                  string
	ResourceDisposition      domain.ResourceDisposition
	ReentryTargetType        domain.ReentryTargetType
	ReentryTargetID          string
	RequestKey               string
}

var _ ProofManager = (*Service)(nil)

// GetProof derives the current Prove & Close state from immutable records.
func (s *Service) GetProof(ctx context.Context, outcomeID domain.OutcomeID) (ProofView, error) {
	if s.proof == nil {
		return ProofView{}, apierr.Internal("OUTCOME_PROOF_UNAVAILABLE", "Outcome proof storage is unavailable")
	}
	outcomeView, err := s.Get(ctx, outcomeID)
	if err != nil {
		return ProofView{}, err
	}
	evidence, err := s.proof.ListEvidenceItems(ctx, outcomeID)
	if err != nil {
		return ProofView{}, err
	}
	verifications, err := s.proof.ListVerificationRuns(ctx, outcomeID)
	if err != nil {
		return ProofView{}, err
	}
	decisions, err := s.proof.ListAcceptanceDecisions(ctx, outcomeID)
	if err != nil {
		return ProofView{}, err
	}
	corrections, err := s.proof.ListOutcomeCorrections(ctx, outcomeID)
	if err != nil {
		return ProofView{}, err
	}
	delegated, err := s.delegatedCriteria(ctx, outcomeView)
	if err != nil {
		return ProofView{}, err
	}
	return deriveProof(outcomeView, evidence, verifications, decisions, corrections, delegated), nil
}

// RecordEvidence appends provenance-bearing Evidence for the current contract.
func (s *Service) RecordEvidence(ctx context.Context, outcomeID domain.OutcomeID, in RecordEvidenceInput) (ProofView, error) {
	if s.proof == nil {
		return ProofView{}, apierr.Internal("OUTCOME_PROOF_UNAVAILABLE", "Outcome proof storage is unavailable")
	}
	fingerprint, err := requestFingerprint("evidence", outcomeID, in)
	if err != nil {
		return ProofView{}, err
	}
	if replay, ok, err := s.proof.FindEvidenceItemByRequestKey(ctx, strings.TrimSpace(in.RequestKey)); err != nil {
		return ProofView{}, err
	} else if ok {
		if replay.OutcomeID != outcomeID || replay.RequestFingerprint != fingerprint {
			return ProofView{}, replayConflict("EVIDENCE_REQUEST_CONFLICT", in.RequestKey)
		}
		return s.GetProof(ctx, outcomeID)
	}
	if err := s.validateProofTarget(ctx, outcomeID, in.ExpectedContractRevision, in.ContractRevisionID, in.CriterionID, in.SubjectType, in.SubjectID, in.SubjectRevision); err != nil {
		return ProofView{}, err
	}
	item := domain.EvidenceItem{
		ID:                 domain.EvidenceItemID("ev-" + uuid.NewString()),
		OutcomeID:          outcomeID,
		ContractRevisionID: in.ContractRevisionID,
		CriterionID:        in.CriterionID,
		SubjectType:        in.SubjectType,
		SubjectID:          strings.TrimSpace(in.SubjectID),
		SubjectRevision:    strings.TrimSpace(in.SubjectRevision),
		Kind:               in.Kind,
		SourceType:         in.SourceType,
		SourceRef:          strings.TrimSpace(in.SourceRef),
		ProducerType:       in.ProducerType,
		ProducerRef:        strings.TrimSpace(in.ProducerRef),
		Summary:            strings.TrimSpace(in.Summary),
		ContentDigest:      strings.ToLower(strings.TrimSpace(in.ContentDigest)),
		RequestKey:         strings.TrimSpace(in.RequestKey),
		RequestFingerprint: fingerprint,
		CreatedAt:          s.clock(),
	}
	if err := item.Validate(); err != nil {
		return ProofView{}, apierr.Invalid("EVIDENCE_INVALID", err.Error(), nil)
	}
	if err := s.proof.CreateEvidenceItem(ctx, item); err != nil {
		if replay, ok, findErr := s.proof.FindEvidenceItemByRequestKey(ctx, item.RequestKey); findErr == nil && ok {
			if replay.OutcomeID == outcomeID && replay.RequestFingerprint == fingerprint {
				return s.GetProof(ctx, outcomeID)
			}
			return ProofView{}, replayConflict("EVIDENCE_REQUEST_CONFLICT", item.RequestKey)
		}
		return ProofView{}, err
	}
	return s.GetProof(ctx, outcomeID)
}

// RecordVerification appends the actual verification method and independence class.
func (s *Service) RecordVerification(ctx context.Context, outcomeID domain.OutcomeID, in RecordVerificationInput) (ProofView, error) {
	if s.proof == nil {
		return ProofView{}, apierr.Internal("OUTCOME_PROOF_UNAVAILABLE", "Outcome proof storage is unavailable")
	}
	fingerprint, err := requestFingerprint("verification", outcomeID, in)
	if err != nil {
		return ProofView{}, err
	}
	if replay, ok, err := s.proof.FindVerificationRunByRequestKey(ctx, strings.TrimSpace(in.RequestKey)); err != nil {
		return ProofView{}, err
	} else if ok {
		if replay.OutcomeID != outcomeID || replay.RequestFingerprint != fingerprint {
			return ProofView{}, replayConflict("VERIFICATION_REQUEST_CONFLICT", in.RequestKey)
		}
		return s.GetProof(ctx, outcomeID)
	}
	if err := s.validateProofTarget(ctx, outcomeID, in.ExpectedContractRevision, in.ContractRevisionID, in.CriterionID, in.SubjectType, in.SubjectID, in.SubjectRevision); err != nil {
		return ProofView{}, err
	}
	for _, id := range in.EvidenceItemIDs {
		item, ok, err := s.proof.GetEvidenceItem(ctx, outcomeID, id)
		if err != nil {
			return ProofView{}, err
		}
		if !ok {
			return ProofView{}, apierr.NotFound("EVIDENCE_NOT_FOUND", "One of the Evidence items does not exist")
		}
		if item.ContractRevisionID != in.ContractRevisionID || item.CriterionID != in.CriterionID || item.SubjectType != in.SubjectType || item.SubjectID != strings.TrimSpace(in.SubjectID) || item.SubjectRevision != strings.TrimSpace(in.SubjectRevision) {
			return ProofView{}, apierr.Invalid("EVIDENCE_BINDING_MISMATCH", "Verification Evidence must bind the exact same criterion and subject revision", map[string]any{"evidenceItemId": id})
		}
	}
	run := domain.VerificationRun{
		ID:                 domain.VerificationRunID("ver-" + uuid.NewString()),
		OutcomeID:          outcomeID,
		ContractRevisionID: in.ContractRevisionID,
		CriterionID:        in.CriterionID,
		SubjectType:        in.SubjectType,
		SubjectID:          strings.TrimSpace(in.SubjectID),
		SubjectRevision:    strings.TrimSpace(in.SubjectRevision),
		EvidenceItemIDs:    append([]domain.EvidenceItemID(nil), in.EvidenceItemIDs...),
		Method:             strings.TrimSpace(in.Method),
		IndependenceClass:  in.IndependenceClass,
		Result:             in.Result,
		ProducerRef:        strings.TrimSpace(in.ProducerRef),
		VerifierRef:        strings.TrimSpace(in.VerifierRef),
		ProducerProvider:   strings.TrimSpace(in.ProducerProvider),
		VerifierProvider:   strings.TrimSpace(in.VerifierProvider),
		Detail:             strings.TrimSpace(in.Detail),
		RequestKey:         strings.TrimSpace(in.RequestKey),
		RequestFingerprint: fingerprint,
		CreatedAt:          s.clock(),
	}
	if err := run.Validate(); err != nil {
		return ProofView{}, apierr.Invalid("VERIFICATION_INVALID", err.Error(), nil)
	}
	if err := s.proof.CreateVerificationRun(ctx, run); err != nil {
		if replay, ok, findErr := s.proof.FindVerificationRunByRequestKey(ctx, run.RequestKey); findErr == nil && ok {
			if replay.OutcomeID == outcomeID && replay.RequestFingerprint == fingerprint {
				return s.GetProof(ctx, outcomeID)
			}
			return ProofView{}, replayConflict("VERIFICATION_REQUEST_CONFLICT", run.RequestKey)
		}
		return ProofView{}, err
	}
	return s.GetProof(ctx, outcomeID)
}

// DecideAcceptance appends the sole user-authoritative closure or re-entry fact.
func (s *Service) DecideAcceptance(ctx context.Context, outcomeID domain.OutcomeID, in DecideAcceptanceInput) (ProofView, error) {
	if s.proof == nil {
		return ProofView{}, apierr.Internal("OUTCOME_PROOF_UNAVAILABLE", "Outcome proof storage is unavailable")
	}
	fingerprint, err := requestFingerprint("acceptance", outcomeID, in)
	if err != nil {
		return ProofView{}, err
	}
	if replay, ok, err := s.proof.FindAcceptanceDecisionByRequestKey(ctx, strings.TrimSpace(in.RequestKey)); err != nil {
		return ProofView{}, err
	} else if ok {
		if replay.OutcomeID != outcomeID || replay.RequestFingerprint != fingerprint {
			return ProofView{}, replayConflict("ACCEPTANCE_REQUEST_CONFLICT", in.RequestKey)
		}
		return s.GetProof(ctx, outcomeID)
	}
	current, err := s.requireCurrentContract(ctx, outcomeID, in.ExpectedContractRevision, in.ContractRevisionID)
	if err != nil {
		return ProofView{}, err
	}
	proof, err := s.GetProof(ctx, outcomeID)
	if err != nil {
		return ProofView{}, err
	}
	switch in.Kind {
	case domain.AcceptanceAccept:
		if proof.Status != ProofStatusReadyForAcceptance {
			return ProofView{}, apierr.Conflict("OUTCOME_NOT_READY_FOR_ACCEPTANCE", "Every current criterion needs supporting Evidence and passing Verification before acceptance", map[string]any{"status": proof.Status})
		}
		if in.ReentryTargetType != "" || strings.TrimSpace(in.ReentryTargetID) != "" {
			return ProofView{}, apierr.Invalid("ACCEPTANCE_REENTRY_INVALID", "Acceptance does not create re-entry lineage", nil)
		}
	case domain.AcceptanceRequestRework:
		if proof.Status == ProofStatusAccepted {
			return ProofView{}, apierr.Conflict("OUTCOME_REOPEN_REQUIRED", "Reopen an accepted Outcome before requesting more work", nil)
		}
		if err := s.validateReentryTarget(ctx, outcomeID, current, in.ReentryTargetType, in.ReentryTargetID); err != nil {
			return ProofView{}, err
		}
	case domain.AcceptanceReopen:
		if proof.Status != ProofStatusAccepted {
			return ProofView{}, apierr.Conflict("OUTCOME_NOT_ACCEPTED", "Only an accepted Outcome can be reopened", map[string]any{"status": proof.Status})
		}
		if err := s.validateReentryTarget(ctx, outcomeID, current, in.ReentryTargetType, in.ReentryTargetID); err != nil {
			return ProofView{}, err
		}
	default:
		return ProofView{}, apierr.Invalid("ACCEPTANCE_INVALID", "Choose accept, request_rework, or reopen", nil)
	}
	decision := domain.AcceptanceDecision{
		ID:                  domain.AcceptanceDecisionID("acc-" + uuid.NewString()),
		OutcomeID:           outcomeID,
		ContractRevisionID:  current.ID,
		Kind:                in.Kind,
		ActorType:           domain.AcceptanceActorUser,
		Summary:             strings.TrimSpace(in.Summary),
		ResourceDisposition: in.ResourceDisposition,
		RequestKey:          strings.TrimSpace(in.RequestKey),
		RequestFingerprint:  fingerprint,
		CreatedAt:           s.clock(),
	}
	if err := decision.Validate(); err != nil {
		return ProofView{}, apierr.Invalid("ACCEPTANCE_INVALID", err.Error(), nil)
	}
	var correction *domain.OutcomeCorrection
	if in.Kind == domain.AcceptanceRequestRework || in.Kind == domain.AcceptanceReopen {
		correction = &domain.OutcomeCorrection{
			ID:                 domain.OutcomeCorrectionID("corr-" + uuid.NewString()),
			DecisionID:         decision.ID,
			OutcomeID:          outcomeID,
			ContractRevisionID: current.ID,
			Feedback:           decision.Summary,
			TargetType:         in.ReentryTargetType,
			TargetID:           strings.TrimSpace(in.ReentryTargetID),
			CreatedAt:          decision.CreatedAt,
		}
	}
	if err := s.proof.CreateAcceptanceDecision(ctx, decision, correction); err != nil {
		if replay, ok, findErr := s.proof.FindAcceptanceDecisionByRequestKey(ctx, decision.RequestKey); findErr == nil && ok {
			if replay.OutcomeID == outcomeID && replay.RequestFingerprint == fingerprint {
				return s.GetProof(ctx, outcomeID)
			}
			return ProofView{}, replayConflict("ACCEPTANCE_REQUEST_CONFLICT", decision.RequestKey)
		}
		return ProofView{}, err
	}
	return s.GetProof(ctx, outcomeID)
}

func (s *Service) requireCurrentContract(ctx context.Context, outcomeID domain.OutcomeID, expected int64, revisionID domain.ContractRevisionID) (domain.ContractRevision, error) {
	view, err := s.Get(ctx, outcomeID)
	if err != nil {
		return domain.ContractRevision{}, err
	}
	if expected < 1 || expected != view.Current.Number || revisionID != view.Current.ID {
		return domain.ContractRevision{}, apierr.Conflict("OUTCOME_PROOF_CONTRACT_CONFLICT", "Proof must bind the Outcome's current immutable contract revision", map[string]any{
			"expectedRevision": expected, "currentRevision": view.Current.Number, "contractRevisionId": revisionID, "currentContractRevisionId": view.Current.ID,
		})
	}
	return view.Current, nil
}

func (s *Service) validateProofTarget(ctx context.Context, outcomeID domain.OutcomeID, expected int64, revisionID domain.ContractRevisionID, criterionID domain.CriterionID, subjectType domain.ProofSubjectType, subjectID, subjectRevision string) error {
	current, err := s.requireCurrentContract(ctx, outcomeID, expected, revisionID)
	if err != nil {
		return err
	}
	var criterion domain.ContractCriterion
	for _, candidate := range current.Criteria {
		if candidate.ID == criterionID {
			criterion = candidate
			break
		}
	}
	if criterion.ID.IsZero() {
		return apierr.Invalid("CRITERION_BINDING_INVALID", "Evidence and Verification must name a criterion in the current contract revision", nil)
	}
	subjectID = strings.TrimSpace(subjectID)
	subjectRevision = strings.TrimSpace(subjectRevision)
	switch subjectType {
	case domain.ProofSubjectOutcome:
		if subjectID != string(outcomeID) || subjectRevision != string(current.ID) {
			return subjectMismatch()
		}
	case domain.ProofSubjectContract:
		if subjectID != string(current.ID) || subjectRevision != string(current.ID) {
			return subjectMismatch()
		}
	case domain.ProofSubjectPlan:
		plan, ok, err := s.store.GetPlanRevision(ctx, outcomeID, domain.PlanRevisionID(subjectID))
		if err != nil {
			return err
		}
		if !ok || plan.ContractRevisionNumber != current.Number || subjectRevision != string(plan.ID) {
			return subjectMismatch()
		}
	case domain.ProofSubjectWorkUnit:
		plan, ok, err := s.store.GetLatestPlanRevision(ctx, outcomeID)
		if err != nil {
			return err
		}
		if !ok || plan.ContractRevisionNumber != current.Number || subjectRevision != string(plan.ID) || len(plan.WorkUnits) != 1 || subjectID != string(plan.WorkUnits[0].ID) {
			return subjectMismatch()
		}
	case domain.ProofSubjectAttempt:
		attempt, ok, err := s.store.GetAttempt(ctx, outcomeID, domain.AttemptID(subjectID))
		if err != nil {
			return err
		}
		if !ok || attempt.ContractRevisionNumber != current.Number || subjectRevision != string(attempt.ID) {
			return subjectMismatch()
		}
	default:
		return subjectMismatch()
	}
	return nil
}

// deriveProof computes the read model. delegated carries the parent criteria
// a decomposition hands to contributing Outcomes: those are proved by their
// contributors' acceptance rather than by the parent's own Evidence, so one
// definition of "ready" serves a direct and a decomposed Outcome alike and
// nothing downstream has to ask which shape it is looking at. It is nil for
// every direct Outcome, which is every Outcome that predates composition.
func deriveProof(view OutcomeView, allEvidence []domain.EvidenceItem, allVerifications []domain.VerificationRun, allDecisions []domain.AcceptanceDecision, corrections []domain.OutcomeCorrection, delegated map[domain.CriterionID]domain.DelegatedCriterion) ProofView {
	currentDecisions := make([]domain.AcceptanceDecision, 0)
	var horizon time.Time
	for _, decision := range allDecisions {
		if decision.ContractRevisionID != view.Current.ID {
			continue
		}
		currentDecisions = append(currentDecisions, decision)
		if (decision.Kind == domain.AcceptanceRequestRework || decision.Kind == domain.AcceptanceReopen) && decision.CreatedAt.After(horizon) {
			horizon = decision.CreatedAt
		}
	}
	currentCorrections := make([]domain.OutcomeCorrection, 0)
	for _, correction := range corrections {
		if correction.ContractRevisionID == view.Current.ID {
			currentCorrections = append(currentCorrections, correction)
		}
	}

	proof := ProofView{OutcomeID: view.Outcome.ID, Contract: view.Current, Status: ProofStatusActive, Decisions: currentDecisions, Corrections: currentCorrections, ProofHorizon: horizon}
	allReady := len(view.Current.Criteria) > 0
	for _, criterion := range view.Current.Criteria {
		criterionView := CriterionProofView{Criterion: criterion, Gap: "Add supporting Evidence for this criterion."}
		for _, item := range allEvidence {
			if item.ContractRevisionID == view.Current.ID && item.CriterionID == criterion.ID {
				criterionView.Evidence = append(criterionView.Evidence, item)
			}
		}
		for _, run := range allVerifications {
			if run.ContractRevisionID == view.Current.ID && run.CriterionID == criterion.ID {
				criterionView.Verifications = append(criterionView.Verifications, run)
			}
		}
		if entry, isDelegated := delegated[criterion.ID]; isDelegated {
			// A delegated criterion is proved by its contributors, never by
			// the parent's own Evidence.
			criterionView.Delegated = true
			criterionView.ClaimedBy = entry.ClaimedBy
			criterionView.Ready, criterionView.Gap = entry.Proved, entry.Gap
		} else {
			criterionView.Ready, criterionView.Gap = criterionReady(criterionView, horizon)
		}
		allReady = allReady && criterionView.Ready
		proof.Criteria = append(proof.Criteria, criterionView)
	}
	latestDecision := domain.AcceptanceDecision{}
	if len(currentDecisions) > 0 {
		latestDecision = currentDecisions[len(currentDecisions)-1]
	}
	switch {
	case latestDecision.Kind == domain.AcceptanceAccept:
		proof.Status = ProofStatusAccepted
		proof.NextAction = "Accepted. Reopen explicitly if the result needs more work."
	case allReady:
		proof.Status = ProofStatusReadyForAcceptance
		proof.NextAction = "Review the current proof and explicitly accept or request rework."
	case !horizon.IsZero():
		proof.Status = ProofStatusReworkRequired
		proof.NextAction = "Follow the recorded correction, then add fresh Evidence and Verification."
	default:
		proof.NextAction = "Complete current criterion Evidence and Verification."
	}
	return proof
}

func criterionReady(view CriterionProofView, horizon time.Time) (bool, string) {
	var latestEvidence domain.EvidenceItem
	for _, item := range view.Evidence {
		if !horizon.IsZero() && !item.CreatedAt.After(horizon) {
			continue
		}
		if latestEvidence.ID == "" || item.CreatedAt.After(latestEvidence.CreatedAt) {
			latestEvidence = item
		}
	}
	if latestEvidence.ID == "" || latestEvidence.Kind != domain.EvidenceSupporting {
		if latestEvidence.Kind == domain.EvidenceContradicting {
			return false, "Resolve the latest contradicting Evidence."
		}
		return false, "Add supporting Evidence for this criterion."
	}
	var latestVerification domain.VerificationRun
	for _, run := range view.Verifications {
		if (!horizon.IsZero() && !run.CreatedAt.After(horizon)) || run.CreatedAt.Before(latestEvidence.CreatedAt) || !containsEvidence(run.EvidenceItemIDs, latestEvidence.ID) {
			continue
		}
		if latestVerification.ID == "" || run.CreatedAt.After(latestVerification.CreatedAt) {
			latestVerification = run
		}
	}
	if latestVerification.ID == "" {
		return false, "Verify the latest supporting Evidence."
	}
	if latestVerification.Result != domain.VerificationPassed && latestVerification.Result != domain.VerificationException {
		return false, "Resolve the latest failed or inconclusive Verification."
	}
	return true, ""
}

func containsEvidence(ids []domain.EvidenceItemID, want domain.EvidenceItemID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func requestFingerprint(kind string, outcomeID domain.OutcomeID, input any) (string, error) {
	payload, err := json.Marshal(struct {
		Kind      string
		OutcomeID domain.OutcomeID
		Input     any
	}{kind, outcomeID, input})
	if err != nil {
		return "", apierr.Internal("PROOF_REQUEST_FINGERPRINT_FAILED", "Could not fingerprint the proof request")
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func replayConflict(code, key string) error {
	return apierr.Conflict(code, "That request key was already used for different proof content", map[string]any{"requestKey": strings.TrimSpace(key)})
}

func subjectMismatch() error {
	return apierr.Invalid("PROOF_SUBJECT_MISMATCH", "The proof subject must name an exact current Outcome lineage revision", nil)
}

func (s *Service) validateReentryTarget(ctx context.Context, outcomeID domain.OutcomeID, current domain.ContractRevision, targetType domain.ReentryTargetType, targetID string) error {
	targetID = strings.TrimSpace(targetID)
	correction := domain.OutcomeCorrection{
		ID: "validation", DecisionID: "validation", OutcomeID: "validation", ContractRevisionID: "validation",
		Feedback: "validation", TargetType: targetType, TargetID: targetID, CreatedAt: time.Now(),
	}
	if err := correction.Validate(); err != nil {
		return apierr.Invalid("REENTRY_TARGET_REQUIRED", err.Error(), nil)
	}
	valid := false
	switch targetType {
	case domain.ReentryTargetContract:
		valid = targetID == string(current.ID)
	case domain.ReentryTargetPlan:
		plan, ok, err := s.store.GetPlanRevision(ctx, outcomeID, domain.PlanRevisionID(targetID))
		if err != nil {
			return err
		}
		valid = ok && plan.ContractRevisionNumber == current.Number
	case domain.ReentryTargetWorkUnit:
		plan, ok, err := s.store.GetLatestPlanRevision(ctx, outcomeID)
		if err != nil {
			return err
		}
		if ok && plan.ContractRevisionNumber == current.Number {
			for _, unit := range plan.WorkUnits {
				if targetID == string(unit.ID) {
					valid = true
					break
				}
			}
		}
	case domain.ReentryTargetAttempt:
		attempt, ok, err := s.store.GetAttempt(ctx, outcomeID, domain.AttemptID(targetID))
		if err != nil {
			return err
		}
		valid = ok && attempt.ContractRevisionNumber == current.Number
	}
	if !valid {
		return apierr.Invalid("REENTRY_TARGET_MISMATCH", "The re-entry target must belong to the Outcome's current contract lineage", map[string]any{"targetType": targetType, "targetId": targetID})
	}
	return nil
}

// delegatedCriteria resolves which of an Outcome's current criteria are proved
// by contributing Outcomes. A direct Outcome returns nil, so its proof
// derivation is byte-for-byte what it was before composition existed.
func (s *Service) delegatedCriteria(ctx context.Context, view OutcomeView) (map[domain.CriterionID]domain.DelegatedCriterion, error) {
	children, err := s.store.ListContributingOutcomes(ctx, view.Outcome.ID)
	if err != nil {
		return nil, err
	}
	if len(children) == 0 {
		return nil, nil
	}
	links, err := s.store.ListContributionLinksForParent(ctx, view.Outcome.ID)
	if err != nil {
		return nil, err
	}
	accepted := make(map[domain.OutcomeID]bool, len(children))
	titles := make(map[domain.OutcomeID]string, len(children))
	for _, child := range children {
		titles[child.ID] = child.Title
		decisions, err := s.proof.ListAcceptanceDecisions(ctx, child.ID)
		if err != nil {
			return nil, err
		}
		accepted[child.ID] = domain.LatestDecisionAccepts(decisions)
	}
	return domain.DelegatedCriteria(view.Current, links, accepted, titles), nil
}
