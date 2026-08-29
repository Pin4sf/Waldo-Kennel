package outcome

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
)

// AcceptBatchInput is one owner sitting over a decomposed Outcome's ready
// contributors.
type AcceptBatchInput struct {
	// ExpectedContractRevision must name the parent's current revision.
	ExpectedContractRevision int64
	// OutcomeIDs narrows the sitting to specific contributors. Empty means
	// "every eligible contributor", which is the ordinary case.
	OutcomeIDs []domain.OutcomeID
	// Summary is the owner's words for this sitting. It is copied onto every
	// decision, which stay separate records.
	Summary             string
	ResourceDisposition domain.ResourceDisposition
	// AcceptParent also accepts the Project-level Outcome, permitted only when
	// accepting this batch leaves every parent criterion proved.
	AcceptParent bool
	RequestKey   string
}

// AcceptBatchView reports what one sitting decided and what it could not.
type AcceptBatchView struct {
	// Accepted is one immutable decision per contributing Outcome accepted.
	Accepted []domain.AcceptanceDecision
	// Excluded names every contributor the daemon withheld, with the reason
	// and the smallest remedy. Exclusion is escalation, not rejection.
	Excluded []domain.BatchEntryVerdict
	// ParentAccepted is the parent's own decision when the owner asked for it
	// and every criterion was proved; nil otherwise.
	ParentAccepted *domain.AcceptanceDecision
	// Parent is the parent's proof state after the sitting.
	Parent ProofView
}

// BatchEligibility reports, without deciding anything, which contributors
// could enter a batched acceptance right now and why the others could not.
func (s *Service) BatchEligibility(ctx context.Context, parentID domain.OutcomeID) ([]domain.BatchEntryVerdict, error) {
	if s.proof == nil {
		return nil, apierr.Internal("OUTCOME_PROOF_UNAVAILABLE", "Outcome proof storage is unavailable")
	}
	composition, err := s.Composition(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if composition.Shape != domain.OutcomeShapeDecomposed {
		return nil, apierr.Invalid("OUTCOME_NOT_DECOMPOSED",
			"Only a decomposed Outcome has contributors to accept together", nil)
	}
	verdicts := make([]domain.BatchEntryVerdict, 0, len(composition.Contributors))
	for _, contributor := range composition.Contributors {
		facts, err := s.contributorProofFacts(ctx, contributor)
		if err != nil {
			return nil, err
		}
		verdicts = append(verdicts, domain.EligibleForAcceptanceBatch(facts, domain.MinimumBatchIndependence))
	}
	sort.SliceStable(verdicts, func(i, j int) bool { return verdicts[i].OutcomeID < verdicts[j].OutcomeID })
	return verdicts, nil
}

// AcceptContributorBatch records the owner's decision over every eligible
// contributor in one sitting.
//
// It writes N separate immutable AcceptanceDecisions — one per Outcome, each
// with its own contract revision and idempotency identity — plus the parent's
// when asked and earned. It never merges them into one decision, and it never
// lets an ineligible contributor through: the daemon's only power over the
// batch is EXCLUSION.
//
// This is the whole answer to composition's cost. Acceptance multiplies with
// contributors, which attacks the supervision-cost the product is judged on,
// so the sitting is batched while the authority stays unbatched.
func (s *Service) AcceptContributorBatch(ctx context.Context, parentID domain.OutcomeID, in AcceptBatchInput) (AcceptBatchView, error) {
	if s.proof == nil {
		return AcceptBatchView{}, apierr.Internal("OUTCOME_PROOF_UNAVAILABLE", "Outcome proof storage is unavailable")
	}
	if strings.TrimSpace(in.RequestKey) == "" {
		return AcceptBatchView{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this sitting", nil)
	}
	if strings.TrimSpace(in.Summary) == "" {
		return AcceptBatchView{}, apierr.Invalid("ACCEPTANCE_SUMMARY_REQUIRED",
			"Say what you are accepting; the summary is kept on every decision", nil)
	}
	parent, ok, err := s.store.GetOutcome(ctx, parentID)
	if err != nil {
		return AcceptBatchView{}, err
	}
	if !ok {
		return AcceptBatchView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome no longer exists")
	}
	parentContract, err := s.currentRevision(ctx, parent)
	if err != nil {
		return AcceptBatchView{}, err
	}
	if in.ExpectedContractRevision != parentContract.Number {
		return AcceptBatchView{}, apierr.Conflict("OUTCOME_CONTRACT_CONFLICT",
			"That is no longer the parent's current contract revision",
			map[string]any{"expected": in.ExpectedContractRevision, "current": parentContract.Number})
	}

	composition, err := s.Composition(ctx, parentID)
	if err != nil {
		return AcceptBatchView{}, err
	}
	if composition.Shape != domain.OutcomeShapeDecomposed {
		return AcceptBatchView{}, apierr.Invalid("OUTCOME_NOT_DECOMPOSED",
			"Only a decomposed Outcome has contributors to accept together", nil)
	}

	// Replay FIRST, like every other delivered write. Without this, a
	// re-sent sitting finds every contributor already accepted, filters them
	// all out, and reports "nothing is eligible" — an error for a request
	// that already succeeded.
	if replayed, isReplay, err := s.replayedSitting(ctx, parentID, composition, in); err != nil {
		return AcceptBatchView{}, err
	} else if isReplay {
		return replayed, nil
	}

	requested := make(map[domain.OutcomeID]struct{}, len(in.OutcomeIDs))
	for _, id := range in.OutcomeIDs {
		requested[id] = struct{}{}
	}
	now := s.clock()
	view := AcceptBatchView{
		Accepted: make([]domain.AcceptanceDecision, 0, len(composition.Contributors)),
		Excluded: make([]domain.BatchEntryVerdict, 0),
	}
	decisions := make([]domain.AcceptanceDecision, 0, len(composition.Contributors)+1)

	for _, contributor := range composition.Contributors {
		if len(requested) > 0 {
			if _, asked := requested[contributor.Outcome.ID]; !asked {
				continue
			}
		}
		facts, err := s.contributorProofFacts(ctx, contributor)
		if err != nil {
			return AcceptBatchView{}, err
		}
		verdict := domain.EligibleForAcceptanceBatch(facts, domain.MinimumBatchIndependence)
		if !verdict.Eligible {
			// An already-accepted contributor is not an exclusion worth
			// escalating; it simply has nothing left to decide.
			if !facts.Accepted {
				view.Excluded = append(view.Excluded, verdict)
			}
			continue
		}
		contract, err := s.currentRevision(ctx, contributor.Outcome)
		if err != nil {
			return AcceptBatchView{}, err
		}
		decisions = append(decisions, s.batchDecision(contributor.Outcome.ID, contract.ID, in, now))
	}

	// The parent is accepted only when accepting this batch actually leaves
	// every criterion proved — delegated ones by their contributors, retained
	// ones by the owner's own Evidence. All contributors accepted makes a
	// parent READY, never accepted.
	if in.AcceptParent {
		ready, gap, err := s.parentReadyAfterBatch(ctx, parentID, decisions)
		if err != nil {
			return AcceptBatchView{}, err
		}
		if !ready {
			return AcceptBatchView{}, apierr.Conflict("OUTCOME_NOT_READY_FOR_ACCEPTANCE",
				"The Project-level Outcome is not fully proved: "+gap, map[string]any{"outcomeId": string(parentID)})
		}
		parentDecision := s.batchDecision(parentID, parentContract.ID, in, now)
		decisions = append(decisions, parentDecision)
		view.ParentAccepted = &parentDecision
	}

	if len(decisions) == 0 {
		return AcceptBatchView{}, apierr.Conflict("ACCEPTANCE_BATCH_EMPTY",
			"No contributor is eligible for acceptance right now",
			map[string]any{"excluded": len(view.Excluded)})
	}
	if err := s.proof.CreateAcceptanceDecisionBatch(ctx, decisions); err != nil {
		// Lost a race with an identical sitting; serve the winner's result
		// rather than writing a second set of decisions.
		if replayed, isReplay, findErr := s.replayedSitting(ctx, parentID, composition, in); findErr == nil && isReplay {
			return replayed, nil
		}
		return AcceptBatchView{}, err
	}
	for _, decision := range decisions {
		if decision.OutcomeID != parentID {
			view.Accepted = append(view.Accepted, decision)
		}
	}
	view.Parent, err = s.GetProof(ctx, parentID)
	if err != nil {
		return AcceptBatchView{}, err
	}
	return view, nil
}

// batchDecision builds one contributor's decision. The idempotency key is
// derived per Outcome from the sitting's key, so a replayed sitting resolves
// to the same decisions instead of writing a second set. The fingerprint
// covers the sitting's terms, so a replay that changed them is a conflict
// rather than a silent overwrite.
func (s *Service) batchDecision(outcomeID domain.OutcomeID, contractRevisionID domain.ContractRevisionID, in AcceptBatchInput, at time.Time) domain.AcceptanceDecision {
	key := strings.TrimSpace(in.RequestKey) + ":" + string(outcomeID)
	digest := sha256.Sum256([]byte(key + "\x00" + strings.TrimSpace(in.Summary) + "\x00" + string(in.ResourceDisposition)))
	return domain.AcceptanceDecision{
		ID:                  domain.AcceptanceDecisionID("acc-" + uuid.NewString()),
		OutcomeID:           outcomeID,
		ContractRevisionID:  contractRevisionID,
		Kind:                domain.AcceptanceAccept,
		ActorType:           domain.AcceptanceActorUser,
		Summary:             strings.TrimSpace(in.Summary),
		ResourceDisposition: in.ResourceDisposition,
		RequestKey:          key,
		RequestFingerprint:  hex.EncodeToString(digest[:]),
		CreatedAt:           at,
	}
}

// parentReadyAfterBatch asks whether the parent would be fully proved once
// this batch's contributors are accepted, without writing anything.
func (s *Service) parentReadyAfterBatch(ctx context.Context, parentID domain.OutcomeID, pending []domain.AcceptanceDecision) (bool, string, error) {
	accepting := make(map[domain.OutcomeID]bool, len(pending))
	for _, decision := range pending {
		accepting[decision.OutcomeID] = true
	}
	proof, err := s.GetProof(ctx, parentID)
	if err != nil {
		return false, "", err
	}
	gaps := make([]string, 0)
	for _, criterion := range proof.Criteria {
		if criterion.Ready {
			continue
		}
		if criterion.Delegated {
			outstanding := false
			for _, child := range criterion.ClaimedBy {
				if !accepting[child] {
					outstanding = true
					break
				}
			}
			if !outstanding {
				continue // this batch proves it
			}
			gaps = append(gaps, criterion.Gap)
			continue
		}
		// A retained criterion is the owner's to prove directly. Retention
		// decides who proves it, never whether it is proved.
		gaps = append(gaps, "you retained \""+criterion.Criterion.Text+"\" and it has no proof yet")
	}
	if len(gaps) == 0 {
		return true, "", nil
	}
	return false, strings.Join(gaps, "; "), nil
}

// replayedSitting resolves a delivered sitting from durable facts.
//
// Every decision in a sitting carries a key derived from the sitting's key and
// its own Outcome, so finding ANY of them proves the sitting landed. A key
// that exists with a different fingerprint means the owner re-sent the same
// sitting with different terms, which is a conflict rather than a replay.
func (s *Service) replayedSitting(ctx context.Context, parentID domain.OutcomeID, composition CompositionView, in AcceptBatchInput) (AcceptBatchView, bool, error) {
	candidates := make([]domain.OutcomeID, 0, len(composition.Contributors)+1)
	for _, contributor := range composition.Contributors {
		candidates = append(candidates, contributor.Outcome.ID)
	}
	candidates = append(candidates, parentID)

	view := AcceptBatchView{
		Accepted: make([]domain.AcceptanceDecision, 0, len(candidates)),
		Excluded: make([]domain.BatchEntryVerdict, 0),
	}
	found := false
	for _, id := range candidates {
		key := strings.TrimSpace(in.RequestKey) + ":" + string(id)
		decision, ok, err := s.proof.FindAcceptanceDecisionByRequestKey(ctx, key)
		if err != nil {
			return AcceptBatchView{}, false, err
		}
		if !ok {
			continue
		}
		expected := s.batchDecision(id, decision.ContractRevisionID, in, decision.CreatedAt)
		if decision.RequestFingerprint != expected.RequestFingerprint {
			return AcceptBatchView{}, false, replayConflict("ACCEPTANCE_REQUEST_CONFLICT", key)
		}
		found = true
		if id == parentID {
			replayed := decision
			view.ParentAccepted = &replayed
			continue
		}
		view.Accepted = append(view.Accepted, decision)
	}
	if !found {
		return AcceptBatchView{}, false, nil
	}
	proof, err := s.GetProof(ctx, parentID)
	if err != nil {
		return AcceptBatchView{}, false, err
	}
	view.Parent = proof
	return view, true, nil
}

// contributorProofFacts derives one contributor's batch-entry facts from its
// durable proof records.
func (s *Service) contributorProofFacts(ctx context.Context, contributor ContributorView) (domain.ContributorProofFacts, error) {
	facts := domain.ContributorProofFacts{
		OutcomeID: contributor.Outcome.ID,
		Title:     contributor.Outcome.Title,
		Stale:     contributor.Stale,
	}
	proof, err := s.GetProof(ctx, contributor.Outcome.ID)
	if err != nil {
		return domain.ContributorProofFacts{}, err
	}
	facts.Ready = proof.Status == ProofStatusReadyForAcceptance
	facts.Accepted = proof.Status == ProofStatusAccepted

	classes := make([]domain.VerificationIndependenceClass, 0, len(proof.Criteria))
	for _, criterion := range proof.Criteria {
		for _, item := range criterion.Evidence {
			if item.Kind == domain.EvidenceContradicting {
				facts.Contradicted = true
			}
		}
		// The class that actually backs this criterion is the newest passing
		// verification on it; anything else did not make it ready.
		var backing domain.VerificationRun
		for _, run := range criterion.Verifications {
			if run.Result != domain.VerificationPassed {
				continue
			}
			if backing.ID == "" || run.CreatedAt.After(backing.CreatedAt) {
				backing = run
			}
		}
		if backing.ID != "" {
			classes = append(classes, backing.IndependenceClass)
		}
	}
	facts.BackingIndependence = domain.WeakestIndependence(classes)
	return facts, nil
}
