package outcome

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// checkEvidenceSummaryLimit bounds what a check's log contributes to durable
// evidence. The full output is a runtime artifact, not proof: content
// integrity comes from the artifact version the check was bound to.
const checkEvidenceSummaryLimit = 2000

// runApprovedChecks executes an ended Attempt's approved checks and records
// what Kennel itself observed as criterion-bound proof.
//
// This is the difference between "the provider finished" and "the criterion is
// proved". The provider's own account of its work is a claim; a command this
// daemon launched under the Attempt's frozen policy, whose result is bound to
// the exact retained artifact version, is an independent observation.
//
// It reports whether any proof was written, so the caller knows to re-read the
// proof projection and generation before classifying.
func (s *Service) runApprovedChecks(
	ctx context.Context,
	attempt domain.Attempt,
	plan domain.PlanRevision,
	unit domain.WorkUnit,
	receipt domain.AttemptReceipt,
	contract domain.ContractRevision,
) (bool, error) {
	if s.checks == nil || len(unit.Checks) == 0 {
		return false, nil
	}
	// Proof binds to the current Contract revision. An Attempt admitted under
	// an older one cannot have its checks recorded against today's criteria.
	if attempt.ContractRevisionNumber != contract.Number {
		return false, nil
	}
	policy, err := domain.BuildAttemptExecutionPolicy(attempt.OutcomeID, plan, unit, plan.RunBriefCoreDigest)
	if err != nil {
		return false, fmt.Errorf("build check policy for %s: %w", attempt.ID, err)
	}
	result, err := s.checks.RunAttemptChecks(ctx, ports.AttemptCheckRequest{
		Attempt: attempt, Receipt: receipt, Policy: policy, Checks: unit.Checks,
	})
	if err != nil {
		return false, fmt.Errorf("run approved checks for %s: %w", attempt.ID, err)
	}
	// A check that rewrote the result invalidates any pass taken from the
	// bytes before it ran. Recording those passes against the retained version
	// would attach proof to content that no longer exists.
	changed := result.ArtifactChanged(receipt.ArtifactVersion)

	wrote := false
	for _, observation := range result.Observations {
		recorded, err := s.recordCheckObservation(ctx, attempt, receipt, contract, observation, changed, result.ObservedArtifactVersion)
		if err != nil {
			return wrote, err
		}
		wrote = wrote || recorded
	}
	return wrote, nil
}

// recordCheckObservation writes one check's evidence and its verification run.
//
// Both carry a deterministic request key over the producing Attempt, the
// artifact version and the check id, so a reconciliation tick that repeats is
// a replay rather than a second observation of the same run.
func (s *Service) recordCheckObservation(
	ctx context.Context,
	attempt domain.Attempt,
	receipt domain.AttemptReceipt,
	contract domain.ContractRevision,
	observation ports.AttemptCheckObservation,
	artifactChanged bool,
	observedVersion string,
) (bool, error) {
	key := checkRequestKey(attempt.ID, receipt.ArtifactVersion, observation.Check.ID)
	summary, detail := checkNarrative(observation, artifactChanged, observedVersion)

	kind, verdict := checkVerdict(observation, artifactChanged)

	if _, err := s.RecordEvidence(ctx, attempt.OutcomeID, RecordEvidenceInput{
		ExpectedContractRevision: contract.Number,
		ContractRevisionID:       contract.ID,
		CriterionID:              observation.Check.CriterionID,
		SubjectType:              domain.ProofSubjectAttempt,
		SubjectID:                string(attempt.ID),
		SubjectRevision:          receipt.ArtifactVersion,
		Kind:                     kind,
		SourceType:               domain.EvidenceSourceDeterministicCheck,
		SourceRef:                strings.Join(observation.Check.Argv, " "),
		ProducerType:             domain.EvidenceProducerTool,
		ProducerRef:              string(attempt.ID),
		Summary:                  summary,
		ContentDigest:            checkOutputDigest(observation),
		RequestKey:               "ev-" + key,
	}); err != nil {
		return false, fmt.Errorf("record check evidence for %s: %w", observation.Check.ID, err)
	}
	item, found, err := s.proof.FindEvidenceItemByRequestKey(ctx, "ev-"+key)
	if err != nil {
		return false, err
	}
	if !found {
		return false, fmt.Errorf("check evidence for %s was not persisted", observation.Check.ID)
	}
	if _, err := s.RecordVerification(ctx, attempt.OutcomeID, RecordVerificationInput{
		ExpectedContractRevision: contract.Number,
		ContractRevisionID:       contract.ID,
		CriterionID:              observation.Check.CriterionID,
		SubjectType:              domain.ProofSubjectAttempt,
		SubjectID:                string(attempt.ID),
		SubjectRevision:          receipt.ArtifactVersion,
		EvidenceItemIDs:          []domain.EvidenceItemID{item.ID},
		Method:                   strings.Join(observation.Check.Argv, " "),
		// Deterministic is the only class this earns: the daemon ran the
		// command itself. An agent's report of the same command would be a
		// producer self-check, and must never be recorded here.
		IndependenceClass: domain.VerificationDeterministic,
		Result:            verdict,
		ProducerRef:       string(attempt.ID),
		VerifierRef:       checkVerifierRef(observation),
		Detail:            detail,
		RequestKey:        "ver-" + key,
	}); err != nil {
		return false, fmt.Errorf("record check verification for %s: %w", observation.Check.ID, err)
	}
	return true, nil
}

// checkVerdict maps one observation to the proof it may support.
//
// The distinctions matter because they lead the owner somewhere different. A
// check that never launched says nothing about the result — recording it as
// contradicting would blame the work for the host's missing sandbox, and
// recording it as supporting would be a lie. A check whose own run changed the
// result it examined proves nothing about the retained bytes either. Only a
// command that actually ran to a confirmed non-zero exit is evidence against
// the result.
func checkVerdict(observation ports.AttemptCheckObservation, artifactChanged bool) (domain.EvidenceKind, domain.VerificationResult) {
	switch {
	case !observation.Ran, artifactChanged, observation.TerminationUnknown:
		return domain.EvidenceSupporting, domain.VerificationInconclusive
	case !observation.Passed:
		return domain.EvidenceContradicting, domain.VerificationFailed
	default:
		return domain.EvidenceSupporting, domain.VerificationPassed
	}
}

// checkRequestKey is the replay identity of one check observation.
func checkRequestKey(attemptID domain.AttemptID, artifactVersion string, checkID domain.ApprovedCheckID) string {
	return fmt.Sprintf("chk:%s:%s:%s", attemptID, artifactVersion, checkID)
}

// checkVerifierRef names what actually held the boundary, so evidence can
// never imply a confinement that did not exist.
func checkVerifierRef(observation ports.AttemptCheckObservation) string {
	if observation.EnforcedBy == "" {
		return "kennel-governed-check/unenforced"
	}
	return "kennel-governed-check/" + observation.EnforcedBy
}

func checkOutputDigest(observation ports.AttemptCheckObservation) string {
	sum := sha256.Sum256([]byte(observation.Output))
	return hex.EncodeToString(sum[:])
}

func checkNarrative(observation ports.AttemptCheckObservation, artifactChanged bool, observedVersion string) (summary, detail string) {
	command := strings.Join(observation.Check.Argv, " ")
	switch {
	case !observation.Ran:
		summary = fmt.Sprintf("%s did not run: %s", command, observation.Unavailable)
	case artifactChanged:
		summary = fmt.Sprintf("%s exited %d, but the result changed while checking", command, observation.ExitCode)
	case observation.TerminationUnknown:
		summary = fmt.Sprintf("%s could not be confirmed stopped", command)
	case observation.TimedOut:
		summary = fmt.Sprintf("%s timed out after %ds", command, observation.Check.TimeoutSeconds)
	case observation.Passed:
		summary = fmt.Sprintf("%s exited 0", command)
	default:
		summary = fmt.Sprintf("%s exited %d", command, observation.ExitCode)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "enforcedBy=%s exit=%d timedOut=%t cancelled=%t terminationUnknown=%t truncated=%t",
		observation.EnforcedBy, observation.ExitCode, observation.TimedOut, observation.Cancelled,
		observation.TerminationUnknown, observation.OutputTruncated)
	if !observation.StartedAt.IsZero() && !observation.EndedAt.IsZero() {
		fmt.Fprintf(&b, " durationMs=%d", observation.EndedAt.Sub(observation.StartedAt).Milliseconds())
	}
	if artifactChanged {
		changedTo := observedVersion
		if changedTo == "" {
			changedTo = "unmeasurable"
		}
		fmt.Fprintf(&b, " artifactChanged=%s->%s", observation.ArtifactVersion, changedTo)
	}
	if output := strings.TrimSpace(observation.Output); output != "" {
		if len(output) > checkEvidenceSummaryLimit {
			output = output[:checkEvidenceSummaryLimit] + "…"
		}
		b.WriteString("\n")
		b.WriteString(output)
	}
	return summary, b.String()
}
