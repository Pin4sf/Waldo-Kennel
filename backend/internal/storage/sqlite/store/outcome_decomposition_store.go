package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/gen"
)

// AppendDecompositionRevision persists one proposed decomposition — its
// contributors, retained criteria, and dependencies — in a single transaction,
// assigning the revision number inside it.
//
// Nothing it writes is a responsibility yet. A proposed decomposition creates
// no Outcome, no contract, and no binding; it is the reviewable offer.
func (s *Store) AppendDecompositionRevision(ctx context.Context, revision domain.DecompositionRevision) (domain.DecompositionRevision, error) {
	if revision.Status != domain.DecompositionProposed {
		return domain.DecompositionRevision{}, fmt.Errorf("a decomposition is persisted as proposed, not %q", revision.Status)
	}
	if err := revision.Validate(); err != nil {
		return domain.DecompositionRevision{}, err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return domain.DecompositionRevision{}, fmt.Errorf("begin decomposition for %s: %w", revision.OutcomeID, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	highest, err := txq.MaxDecompositionRevisionNumber(ctx, revision.OutcomeID)
	if err != nil {
		return domain.DecompositionRevision{}, fmt.Errorf("resolve decomposition number for %s: %w", revision.OutcomeID, err)
	}
	revision.Number = highest + 1

	if err := txq.CreateDecompositionRevision(ctx, gen.CreateDecompositionRevisionParams{
		ID:                 revision.ID,
		OutcomeID:          revision.OutcomeID,
		Number:             revision.Number,
		ContractRevisionID: revision.ContractRevisionID,
		Status:             revision.Status,
		Rationale:          revision.Rationale,
		CreatedAt:          revision.CreatedAt,
	}); err != nil {
		return domain.DecompositionRevision{}, fmt.Errorf("create decomposition %s: %w", revision.ID, err)
	}

	for _, contributor := range revision.Contributors {
		criteria, err := json.Marshal(contributor.SuccessCriteria)
		if err != nil {
			return domain.DecompositionRevision{}, err
		}
		constraints, err := json.Marshal(nonNilStrings(contributor.Constraints))
		if err != nil {
			return domain.DecompositionRevision{}, err
		}
		nonGoals, err := json.Marshal(nonNilStrings(contributor.NonGoals))
		if err != nil {
			return domain.DecompositionRevision{}, err
		}
		authority, err := json.Marshal(contributor.Authority)
		if err != nil {
			return domain.DecompositionRevision{}, err
		}
		claimed, err := json.Marshal(contributor.ClaimedCriteria)
		if err != nil {
			return domain.DecompositionRevision{}, err
		}
		if err := txq.CreateDecompositionContribution(ctx, gen.CreateDecompositionContributionParams{
			ID:              contributionRowID(revision.ID, contributor.Ref),
			DecompositionID: revision.ID,
			Ref:             contributor.Ref,
			Position:        contributor.Position,
			Title:           contributor.Title,
			Goal:            contributor.Goal,
			SuccessCriteria: string(criteria),
			Review:          contributor.Review,
			Constraints:     string(constraints),
			NonGoals:        string(nonGoals),
			Authority:       string(authority),
			ClaimedCriteria: string(claimed),
		}); err != nil {
			return domain.DecompositionRevision{}, fmt.Errorf("create proposed contribution %s: %w", contributor.Ref, err)
		}
	}

	for _, criterion := range revision.RetainedCriteria {
		if err := txq.CreateDecompositionRetainedCriterion(ctx, gen.CreateDecompositionRetainedCriterionParams{
			ID:                string(revision.ID) + ":retained:" + string(criterion),
			DecompositionID:   revision.ID,
			ParentCriterionID: criterion,
		}); err != nil {
			return domain.DecompositionRevision{}, fmt.Errorf("retain criterion %s: %w", criterion, err)
		}
	}

	for _, dependency := range revision.Dependencies {
		if err := txq.CreateContributionDependency(ctx, gen.CreateContributionDependencyParams{
			ID:              dependency.ID,
			DecompositionID: revision.ID,
			FromRef:         dependency.FromRef,
			ToRef:           dependency.ToRef,
		}); err != nil {
			return domain.DecompositionRevision{}, fmt.Errorf("record dependency %s: %w", dependency.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.DecompositionRevision{}, fmt.Errorf("commit decomposition for %s: %w", revision.OutcomeID, err)
	}
	return revision, nil
}

// AuthorizeDecompositionRevision is the owner decision that turns a proposal
// into responsibilities.
//
// Everything happens in one transaction: each contributing Outcome, its
// ContractRevision 1, its criterion bindings, the proposal's resolution to
// real Outcome ids, and the one-way status move. A partial failure must leave
// no half-decomposed parent behind — some children existing and others not
// would be a decomposition nobody authorized.
func (s *Store) AuthorizeDecompositionRevision(
	ctx context.Context,
	outcomeID domain.OutcomeID,
	decompositionID domain.DecompositionRevisionID,
	contributions []ports.AuthorizedContribution,
	at time.Time,
) error {
	if len(contributions) == 0 {
		return fmt.Errorf("authorizing a decomposition requires at least one contribution")
	}
	for _, contribution := range contributions {
		if err := contribution.Outcome.Validate(); err != nil {
			return err
		}
		if err := domain.ValidateContributionLinkSet(contribution.Outcome.ID, contribution.Links); err != nil {
			return err
		}
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin authorize decomposition %s: %w", decompositionID, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	// Claim the status first: a lost race here rolls the whole thing back
	// rather than creating a second set of contributing Outcomes.
	rows, err := txq.AuthorizeDecompositionRevision(ctx, gen.AuthorizeDecompositionRevisionParams{
		AuthorizedAt: sql.NullTime{Time: at, Valid: true},
		ID:           decompositionID,
		OutcomeID:    outcomeID,
	})
	if err != nil {
		return fmt.Errorf("authorize decomposition %s: %w", decompositionID, err)
	}
	if rows != 1 {
		return fmt.Errorf("%w: decomposition %s", ports.ErrDecompositionNotProposed, decompositionID)
	}

	for _, contribution := range contributions {
		if err := txq.CreateOutcome(ctx, gen.CreateOutcomeParams{
			ID:              contribution.Outcome.ID,
			SpaceID:         contribution.Outcome.SpaceID,
			Title:           contribution.Outcome.Title,
			ParentOutcomeID: nullOutcomeID(contribution.Outcome.ParentID),
		}); err != nil {
			return fmt.Errorf("create contributing outcome %s: %w", contribution.Outcome.ID, err)
		}
		number, err := nextRevisionNumber(ctx, txq, contribution.Outcome.ID)
		if err != nil {
			return err
		}
		first := contribution.First
		first.Number = number
		if err := insertContractRevision(ctx, txq, first); err != nil {
			return err
		}
		moved, err := txq.AdvanceOutcomeCurrentRevision(ctx, gen.AdvanceOutcomeCurrentRevisionParams{
			CurrentRevisionNumber:   number,
			UpdatedAt:               first.CreatedAt,
			ID:                      contribution.Outcome.ID,
			CurrentRevisionNumber_2: 0,
		})
		if err != nil {
			return fmt.Errorf("point contributing outcome %s at revision 1: %w", contribution.Outcome.ID, err)
		}
		if moved != 1 {
			return fmt.Errorf("point contributing outcome %s at revision 1: pointer moved concurrently", contribution.Outcome.ID)
		}
		for _, link := range contribution.Links {
			if err := txq.CreateContributionLink(ctx, gen.CreateContributionLinkParams{
				ID:                       string(link.ID),
				ParentOutcomeID:          string(link.ParentOutcomeID),
				ChildOutcomeID:           string(link.ChildOutcomeID),
				ParentContractRevisionID: string(link.ParentContractRevisionID),
				ParentCriterionID:        string(link.ParentCriterionID),
			}); err != nil {
				return fmt.Errorf("bind contribution %s to criterion %s: %w", link.ChildOutcomeID, link.ParentCriterionID, err)
			}
		}
		childID := contribution.Outcome.ID
		bound, err := txq.BindDecompositionContributionOutcome(ctx, gen.BindDecompositionContributionOutcomeParams{
			ChildOutcomeID: &childID,
			ID:             contributionRowID(decompositionID, contribution.Ref),
		})
		if err != nil {
			return fmt.Errorf("resolve proposed contribution %s: %w", contribution.Ref, err)
		}
		if bound != 1 {
			return fmt.Errorf("resolve proposed contribution %s: already resolved or absent", contribution.Ref)
		}
	}
	return tx.Commit()
}

// GetDecompositionRevision reads one decomposition with its contributors,
// retained criteria, and dependencies; ok=false when absent for this Outcome.
func (s *Store) GetDecompositionRevision(ctx context.Context, outcomeID domain.OutcomeID, id domain.DecompositionRevisionID) (domain.DecompositionRevision, bool, error) {
	row, err := s.qr.GetDecompositionRevision(ctx, gen.GetDecompositionRevisionParams{
		ID: id, OutcomeID: outcomeID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.DecompositionRevision{}, false, nil
	}
	if err != nil {
		return domain.DecompositionRevision{}, false, fmt.Errorf("get decomposition %s: %w", id, err)
	}
	revision, err := s.hydrateDecomposition(ctx, decompositionFromRow(row))
	if err != nil {
		return domain.DecompositionRevision{}, false, err
	}
	return revision, true, nil
}

// LatestDecompositionRevision returns the newest decomposition of any status;
// ok=false when the Outcome has never been decomposed.
func (s *Store) LatestDecompositionRevision(ctx context.Context, outcomeID domain.OutcomeID) (domain.DecompositionRevision, bool, error) {
	row, err := s.qr.LatestDecompositionRevision(ctx, outcomeID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.DecompositionRevision{}, false, nil
	}
	if err != nil {
		return domain.DecompositionRevision{}, false, fmt.Errorf("latest decomposition for %s: %w", outcomeID, err)
	}
	revision, err := s.hydrateDecomposition(ctx, decompositionFromRow(row))
	if err != nil {
		return domain.DecompositionRevision{}, false, err
	}
	return revision, true, nil
}

func (s *Store) hydrateDecomposition(ctx context.Context, revision domain.DecompositionRevision) (domain.DecompositionRevision, error) {
	contributions, err := s.qr.ListDecompositionContributions(ctx, revision.ID)
	if err != nil {
		return domain.DecompositionRevision{}, fmt.Errorf("list contributions for %s: %w", revision.ID, err)
	}
	revision.Contributors = make([]domain.ProposedContribution, 0, len(contributions))
	for _, row := range contributions {
		contributor := domain.ProposedContribution{
			Ref: row.Ref, Position: row.Position, Title: row.Title, Goal: row.Goal,
			Review: row.Review, ChildOutcomeID: derefOutcomeID(row.ChildOutcomeID),
		}
		if err := json.Unmarshal([]byte(row.SuccessCriteria), &contributor.SuccessCriteria); err != nil {
			return domain.DecompositionRevision{}, err
		}
		if err := json.Unmarshal([]byte(row.Constraints), &contributor.Constraints); err != nil {
			return domain.DecompositionRevision{}, err
		}
		if err := json.Unmarshal([]byte(row.NonGoals), &contributor.NonGoals); err != nil {
			return domain.DecompositionRevision{}, err
		}
		if err := json.Unmarshal([]byte(row.Authority), &contributor.Authority); err != nil {
			return domain.DecompositionRevision{}, err
		}
		if err := json.Unmarshal([]byte(row.ClaimedCriteria), &contributor.ClaimedCriteria); err != nil {
			return domain.DecompositionRevision{}, err
		}
		revision.Contributors = append(revision.Contributors, contributor)
	}

	retained, err := s.qr.ListDecompositionRetainedCriteria(ctx, revision.ID)
	if err != nil {
		return domain.DecompositionRevision{}, fmt.Errorf("list retained criteria for %s: %w", revision.ID, err)
	}
	revision.RetainedCriteria = make([]domain.CriterionID, 0, len(retained))
	for _, row := range retained {
		revision.RetainedCriteria = append(revision.RetainedCriteria, row.ParentCriterionID)
	}

	dependencies, err := s.qr.ListContributionDependencies(ctx, revision.ID)
	if err != nil {
		return domain.DecompositionRevision{}, fmt.Errorf("list dependencies for %s: %w", revision.ID, err)
	}
	revision.Dependencies = make([]domain.ContributionDependency, 0, len(dependencies))
	for _, row := range dependencies {
		revision.Dependencies = append(revision.Dependencies, domain.ContributionDependency{
			ID: row.ID, FromRef: row.FromRef, ToRef: row.ToRef,
		})
	}
	return revision, nil
}

func decompositionFromRow(row gen.DecompositionRevision) domain.DecompositionRevision {
	revision := domain.DecompositionRevision{
		ID:                 row.ID,
		OutcomeID:          row.OutcomeID,
		Number:             row.Number,
		ContractRevisionID: row.ContractRevisionID,
		Status:             row.Status,
		Rationale:          row.Rationale,
		CreatedAt:          row.CreatedAt,
	}
	if row.AuthorizedAt.Valid {
		at := row.AuthorizedAt.Time
		revision.AuthorizedAt = &at
	}
	return revision
}

// contributionRowID derives a proposed contribution's row id from the
// decomposition and its ref, which are unique together by construction.
func contributionRowID(decomposition domain.DecompositionRevisionID, ref string) string {
	return string(decomposition) + ":" + ref
}

func derefOutcomeID(id *domain.OutcomeID) domain.OutcomeID {
	if id == nil {
		return ""
	}
	return *id
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// AppendContributionDependencyWaiver records the owner's override of a
// declared ordering. Storage refuses a waiver for a dependency nobody
// declared: consenting to nothing is not consent.
func (s *Store) AppendContributionDependencyWaiver(ctx context.Context, waiver domain.ContributionDependencyWaiver) error {
	if err := waiver.Validate(); err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.qw.CreateContributionDependencyWaiver(ctx, gen.CreateContributionDependencyWaiverParams{
		ID:              string(waiver.ID),
		DecompositionID: waiver.DecompositionID,
		FromRef:         waiver.FromRef,
		ToRef:           waiver.ToRef,
		Reason:          waiver.Reason,
		WaivedBy:        string(waiver.WaivedBy),
		CreatedAt:       waiver.CreatedAt,
	}); err != nil {
		return fmt.Errorf("waive dependency %s -> %s: %w", waiver.FromRef, waiver.ToRef, err)
	}
	return nil
}

// ListContributionDependencyWaivers returns every waiver recorded against one
// decomposition, oldest first.
func (s *Store) ListContributionDependencyWaivers(ctx context.Context, decompositionID domain.DecompositionRevisionID) ([]domain.ContributionDependencyWaiver, error) {
	rows, err := s.qr.ListContributionDependencyWaivers(ctx, decompositionID)
	if err != nil {
		return nil, fmt.Errorf("list waivers for %s: %w", decompositionID, err)
	}
	out := make([]domain.ContributionDependencyWaiver, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ContributionDependencyWaiver{
			ID:              domain.ContributionWaiverID(row.ID),
			DecompositionID: row.DecompositionID,
			FromRef:         row.FromRef,
			ToRef:           row.ToRef,
			Reason:          row.Reason,
			WaivedBy:        domain.AcceptanceActorType(row.WaivedBy),
			CreatedAt:       row.CreatedAt,
		})
	}
	return out, nil
}
