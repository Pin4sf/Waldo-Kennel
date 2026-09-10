package outcome

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

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
	if s.checks == nil || s.checkRuns == nil || len(unit.Checks) == 0 {
		return false, nil
	}
	// Proof binds to the current Contract revision. An Attempt admitted under
	// an older one cannot have its checks recorded against today's criteria.
	if attempt.ContractRevisionNumber != contract.Number {
		return false, nil
	}

	pending, recorded, err := s.partitionCheckRuns(ctx, attempt, receipt, unit.Checks)
	if err != nil {
		return false, err
	}
	if len(pending) > 0 {
		newlyRecorded, err := s.invokeReservedChecks(ctx, attempt, plan, unit, receipt, pending)
		if err != nil {
			return false, err
		}
		recorded = append(recorded, newlyRecorded...)
	}

	wrote := false
	for _, run := range recorded {
		written, err := s.recordCheckObservation(ctx, attempt, receipt, contract, run)
		if err != nil {
			return wrote, err
		}
		wrote = wrote || written
	}
	return wrote, nil
}

// partitionCheckRuns separates checks this Attempt still owes from ones whose
// observation is already durable.
//
// The reservation is taken here, before any command is invoked. A row that
// survives as "reserved" from an earlier process is the interrupted case: the
// command may have run and had effects, so it is recorded as unknown rather
// than launched again. Silently re-running it is exactly the behaviour that
// makes a deterministic check unsafe to own.
func (s *Service) partitionCheckRuns(
	ctx context.Context,
	attempt domain.Attempt,
	receipt domain.AttemptReceipt,
	checks []domain.ApprovedCheck,
) (pending []domain.ApprovedCheck, recorded []ports.AttemptCheckRun, err error) {
	for _, check := range checks {
		existing, found, err := s.checkRuns.GetAttemptCheckRun(ctx, attempt.ID, check.ID, receipt.ArtifactVersion)
		if err != nil {
			return nil, nil, err
		}
		if found {
			run, err := s.resolveExistingCheckRun(ctx, attempt, receipt, check, existing)
			if err != nil {
				return nil, nil, err
			}
			recorded = append(recorded, run)
			continue
		}
		reserveErr := s.checkRuns.ReserveAttemptCheckRun(ctx, ports.AttemptCheckRun{
			ID: "chkrun-" + uuid.NewString(), AttemptID: attempt.ID, CheckID: check.ID,
			ArtifactVersion: receipt.ArtifactVersion, ReservedAt: s.clock(),
		})
		switch {
		case reserveErr == nil:
			pending = append(pending, check)
		case errors.Is(reserveErr, ports.ErrCheckRunAlreadyReserved):
			// Another reconciler owns this invocation. Leaving it to them is
			// the whole point of reserving before running.
		default:
			return nil, nil, reserveErr
		}
	}
	return pending, recorded, nil
}

// resolveExistingCheckRun turns a stored row into the observation proof is
// built from, closing out an interrupted reservation on the way.
func (s *Service) resolveExistingCheckRun(
	ctx context.Context,
	attempt domain.Attempt,
	receipt domain.AttemptReceipt,
	check domain.ApprovedCheck,
	existing ports.AttemptCheckRun,
) (ports.AttemptCheckRun, error) {
	existing.Observation.Check = check
	if existing.State == ports.CheckRunReserved {
		// The process that reserved this is gone. Whether the command ran is
		// unknown, and an unknown run is not a failed one and not a retry.
		if err := s.checkRuns.MarkAttemptCheckRunUnknown(ctx, attempt.ID, check.ID, receipt.ArtifactVersion, s.clock()); err != nil {
			return ports.AttemptCheckRun{}, err
		}
		existing.State = ports.CheckRunUnknown
	}
	if existing.State == ports.CheckRunUnknown {
		// Discard whatever the incomplete row happens to hold. A partially
		// written observation is not a weaker observation, it is none: it
		// must not be able to read as a failing check and blame the work for
		// an interruption.
		existing.Observation = ports.AttemptCheckObservation{
			Check: check, ArtifactVersion: receipt.ArtifactVersion,
			Unavailable: "the check was interrupted; whether the command ran is unknown",
			StartedAt:   existing.ReservedAt,
		}
		existing.ArtifactChanged, existing.ObservedArtifactVersion = false, ""
	}
	return existing, nil
}

// invokeReservedChecks runs only the checks this process reserved.
func (s *Service) invokeReservedChecks(
	ctx context.Context,
	attempt domain.Attempt,
	plan domain.PlanRevision,
	unit domain.WorkUnit,
	receipt domain.AttemptReceipt,
	pending []domain.ApprovedCheck,
) ([]ports.AttemptCheckRun, error) {
	policy, err := domain.BuildAttemptExecutionPolicy(attempt.OutcomeID, plan, unit, plan.RunBriefCoreDigest)
	if err != nil {
		return nil, fmt.Errorf("build check policy for %s: %w", attempt.ID, err)
	}
	result, err := s.checks.RunAttemptChecks(ctx, ports.AttemptCheckRequest{
		Attempt: attempt, Receipt: receipt, Policy: policy, Checks: pending,
	})
	if err != nil {
		return nil, fmt.Errorf("run approved checks for %s: %w", attempt.ID, err)
	}
	// A check that rewrote the result invalidates any pass taken from the
	// bytes before it ran, so the fact is stored with the observation rather
	// than recomputed later from a workspace that has since moved on.
	changed := result.ArtifactChanged(receipt.ArtifactVersion)

	runs := make([]ports.AttemptCheckRun, 0, len(result.Observations))
	for _, observation := range result.Observations {
		run := ports.AttemptCheckRun{
			AttemptID: attempt.ID, CheckID: observation.Check.ID, ArtifactVersion: receipt.ArtifactVersion,
			State: ports.CheckRunObserved, Observation: observation,
			ArtifactChanged: changed, ObservedArtifactVersion: result.ObservedArtifactVersion,
		}
		if err := s.checkRuns.RecordAttemptCheckObservation(ctx, run); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
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
	run ports.AttemptCheckRun,
) (bool, error) {
	observation := run.Observation
	key := checkRequestKey(attempt.ID, receipt.ArtifactVersion, observation.Check.ID)
	// Both writes derive entirely from the stored observation, so a restart
	// between them rebuilds byte-identical content. That is what lets the
	// request key mean "this observation" rather than "this attempt at
	// writing it": a differing fingerprint under the same key is a replay
	// conflict, and re-running the command to regenerate the text would both
	// re-execute it and change the text.
	summary, detail := checkNarrative(observation, run.ArtifactChanged, run.ObservedArtifactVersion)
	kind, verdict := checkVerdict(observation, run.ArtifactChanged)

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

// checkRequestKey is the replay identity of one check observation. It is
// stable across restarts because the observation it names is durable.
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
