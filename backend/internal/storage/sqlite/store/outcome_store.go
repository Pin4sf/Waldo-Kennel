package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/gen"
)

// EnsureWorkResponsibilitySpace resolves the project-backed Work space,
// creating it on first use. The daemon serialises writes, so check-then-insert
// under writeMu is race-free; the partial unique index backstops it.
func (s *Store) EnsureWorkResponsibilitySpace(ctx context.Context, projectID domain.ProjectID) (domain.ResponsibilitySpace, error) {
	row, err := s.qr.FindWorkResponsibilitySpaceByProject(ctx, projectID)
	if err == nil {
		return responsibilitySpaceFromRow(row), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.ResponsibilitySpace{}, fmt.Errorf("find work space for %s: %w", projectID, err)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	space := domain.ResponsibilitySpace{
		ID:        domain.ResponsibilitySpaceID("rsp-" + uuid.NewString()),
		Kind:      domain.ResponsibilitySpaceWorkProject,
		ProjectID: projectID,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.qw.CreateResponsibilitySpace(ctx, gen.CreateResponsibilitySpaceParams{
		ID:        space.ID,
		ProjectID: space.ProjectID,
	}); err != nil {
		if isSQLiteUnique(err) {
			// Lost a concurrent first-use race; return the winner.
			row, err := s.qw.FindWorkResponsibilitySpaceByProject(ctx, projectID)
			if err != nil {
				return domain.ResponsibilitySpace{}, fmt.Errorf("re-find work space for %s: %w", projectID, err)
			}
			return responsibilitySpaceFromRow(row), nil
		}
		return domain.ResponsibilitySpace{}, fmt.Errorf("create work space for %s: %w", projectID, err)
	}
	return space, nil
}

// FindOutcomeByIdempotencyKey resolves a previously delivered create request.
func (s *Store) FindOutcomeByIdempotencyKey(ctx context.Context, key string) (domain.Outcome, bool, error) {
	row, err := s.qr.FindOutcomeByIdempotencyKey(ctx, sql.NullString{String: key, Valid: key != ""})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Outcome{}, false, nil
	}
	if err != nil {
		return domain.Outcome{}, false, fmt.Errorf("find outcome by idempotency key: %w", err)
	}
	return outcomeFromRow(row), true, nil
}

// CreateOutcomeWithContract atomically persists the Outcome with its first
// contract revision and points current at it.
func (s *Store) CreateOutcomeWithContract(ctx context.Context, outcome domain.Outcome, first domain.ContractRevision, requestKey string) error {
	if err := outcome.Validate(); err != nil {
		return err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create outcome %s: %w", outcome.ID, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	var key sql.NullString
	if requestKey != "" {
		key = sql.NullString{String: requestKey, Valid: true}
	}
	if err := txq.CreateOutcome(ctx, gen.CreateOutcomeParams{
		ID:             outcome.ID,
		SpaceID:        outcome.SpaceID,
		Title:          outcome.Title,
		IdempotencyKey: key,
	}); err != nil {
		return fmt.Errorf("create outcome %s: %w", outcome.ID, err)
	}

	number, err := nextRevisionNumber(ctx, txq, outcome.ID)
	if err != nil {
		return err
	}
	first.Number = number
	if err := insertContractRevision(ctx, txq, first); err != nil {
		return err
	}

	rows, err := txq.AdvanceOutcomeCurrentRevision(ctx, gen.AdvanceOutcomeCurrentRevisionParams{
		CurrentRevisionNumber:   number,
		UpdatedAt:               first.CreatedAt,
		ID:                      outcome.ID,
		CurrentRevisionNumber_2: 0,
	})
	if err != nil {
		return fmt.Errorf("point outcome %s at revision 1: %w", outcome.ID, err)
	}
	if rows != 1 {
		return fmt.Errorf("point outcome %s at revision 1: pointer moved concurrently", outcome.ID)
	}
	return tx.Commit()
}

// AppendContractRevision atomically appends one immutable revision and swings
// the current-revision pointer from expected to it. A stale expected rolls the
// whole transaction back and reports the conflict.
func (s *Store) AppendContractRevision(ctx context.Context, id domain.OutcomeID, expectedCurrent int64, revision domain.ContractRevision) (int64, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin append revision for %s: %w", id, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	number, err := nextRevisionNumber(ctx, txq, id)
	if err != nil {
		return 0, err
	}
	revision.Number = number
	if err := insertContractRevision(ctx, txq, revision); err != nil {
		return 0, err
	}

	rows, err := txq.AdvanceOutcomeCurrentRevision(ctx, gen.AdvanceOutcomeCurrentRevisionParams{
		CurrentRevisionNumber:   number,
		UpdatedAt:               time.Now().UTC(),
		ID:                      id,
		CurrentRevisionNumber_2: expectedCurrent,
	})
	if err != nil {
		return 0, fmt.Errorf("advance outcome %s to revision %d: %w", id, number, err)
	}
	if rows == 0 {
		currentNum := int64(-1)
		row, getErr := txq.GetOutcome(ctx, id)
		if getErr == nil {
			currentNum = row.CurrentRevisionNumber
		}
		return 0, &ports.OutcomeConflictError{OutcomeID: id, ExpectedRevisionNum: expectedCurrent, CurrentRevisionNum: currentNum}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit append revision for %s: %w", id, err)
	}
	return number, nil
}

// GetOutcome reads one Outcome by id; ok=false when absent.
func (s *Store) GetOutcome(ctx context.Context, id domain.OutcomeID) (domain.Outcome, bool, error) {
	row, err := s.qr.GetOutcome(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Outcome{}, false, nil
	}
	if err != nil {
		return domain.Outcome{}, false, fmt.Errorf("get outcome %s: %w", id, err)
	}
	return outcomeFromRow(row), true, nil
}

// ListOutcomesByProject reads canonical Outcomes through their Work
// responsibility space. It never creates a space as a side effect of reading.
func (s *Store) ListOutcomesByProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Outcome, error) {
	rows, err := s.qr.ListOutcomesByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list outcomes for project %s: %w", projectID, err)
	}
	out := make([]domain.Outcome, 0, len(rows))
	for _, row := range rows {
		out = append(out, outcomeFromRow(row))
	}
	return out, nil
}

// ListContractRevisions returns full immutable history ordered by number.
func (s *Store) ListContractRevisions(ctx context.Context, id domain.OutcomeID) ([]domain.ContractRevision, error) {
	rows, err := s.qr.ListContractRevisions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list revisions for %s: %w", id, err)
	}
	out := make([]domain.ContractRevision, 0, len(rows))
	for _, row := range rows {
		rev, err := contractRevisionFromRow(row)
		if err != nil {
			return nil, err
		}
		criterionRows, err := s.qr.ListContractCriteriaForRevision(ctx, string(rev.ID))
		if err != nil {
			return nil, fmt.Errorf("list stable criteria for revision %s: %w", rev.ID, err)
		}
		rev.Criteria = make([]domain.ContractCriterion, 0, len(criterionRows))
		for _, criterion := range criterionRows {
			rev.Criteria = append(rev.Criteria, domain.ContractCriterion{
				ID:                 domain.CriterionID(criterion.ID),
				ContractRevisionID: domain.ContractRevisionID(criterion.ContractRevisionID),
				Position:           criterion.Position,
				Text:               criterion.Text,
			})
		}
		out = append(out, rev)
	}
	return out, nil
}

func nextRevisionNumber(ctx context.Context, q *gen.Queries, id domain.OutcomeID) (int64, error) {
	maxNum, err := q.MaxContractRevisionNumber(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("max revision number for %s: %w", id, err)
	}
	switch v := maxNum.(type) {
	case int64:
		return v + 1, nil
	default:
		return 0, fmt.Errorf("max revision number for %s: unexpected type %T", id, maxNum)
	}
}

func insertContractRevision(ctx context.Context, q *gen.Queries, revision domain.ContractRevision) error {
	if len(revision.Criteria) == 0 {
		revision.Criteria = make([]domain.ContractCriterion, 0, len(revision.SuccessCriteria))
		for i, text := range revision.SuccessCriteria {
			revision.Criteria = append(revision.Criteria, domain.ContractCriterion{
				ID:                 domain.CriterionID(fmt.Sprintf("crit-%s-%04d", revision.ID, i+1)),
				ContractRevisionID: revision.ID,
				Position:           int64(i + 1),
				Text:               text,
			})
		}
	}
	criteria, err := marshalJSONStrings(revision.SuccessCriteria)
	if err != nil {
		return fmt.Errorf("revision criteria: %w", err)
	}
	constraints, err := marshalJSONStrings(revision.Constraints)
	if err != nil {
		return fmt.Errorf("revision constraints: %w", err)
	}
	nonGoals, err := marshalJSONStrings(revision.NonGoals)
	if err != nil {
		return fmt.Errorf("revision non-goals: %w", err)
	}
	if err := revision.Validate(); err != nil {
		return err
	}
	if err := q.CreateContractRevision(ctx, gen.CreateContractRevisionParams{
		ID:              revision.ID,
		OutcomeID:       revision.OutcomeID,
		Number:          revision.Number,
		Goal:            revision.Goal,
		SuccessCriteria: criteria,
		Review:          revision.Review,
		Constraints:     constraints,
		NonGoals:        nonGoals,
		Clarification:   revision.Clarification,
	}); err != nil {
		return fmt.Errorf("create contract revision %s: %w", revision.ID, err)
	}
	for _, criterion := range revision.Criteria {
		if err := q.CreateContractCriterion(ctx, gen.CreateContractCriterionParams{
			ID:                 string(criterion.ID),
			ContractRevisionID: string(criterion.ContractRevisionID),
			Position:           criterion.Position,
			Text:               criterion.Text,
		}); err != nil {
			return fmt.Errorf("create contract criterion %s: %w", criterion.ID, err)
		}
	}
	return nil
}

func outcomeFromRow(row gen.Outcome) domain.Outcome {
	return domain.Outcome{
		ID:                    row.ID,
		SpaceID:               row.SpaceID,
		Title:                 row.Title,
		CurrentRevisionNumber: row.CurrentRevisionNumber,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}

func responsibilitySpaceFromRow(row gen.ResponsibilitySpace) domain.ResponsibilitySpace {
	return domain.ResponsibilitySpace{
		ID:        row.ID,
		Kind:      row.Kind,
		ProjectID: row.ProjectID,
		CreatedAt: row.CreatedAt,
	}
}

func contractRevisionFromRow(row gen.ContractRevision) (domain.ContractRevision, error) {
	criteria, err := unmarshalJSONStrings(row.SuccessCriteria)
	if err != nil {
		return domain.ContractRevision{}, fmt.Errorf("revision %s criteria: %w", row.ID, err)
	}
	constraints, err := unmarshalJSONStrings(row.Constraints)
	if err != nil {
		return domain.ContractRevision{}, fmt.Errorf("revision %s constraints: %w", row.ID, err)
	}
	nonGoals, err := unmarshalJSONStrings(row.NonGoals)
	if err != nil {
		return domain.ContractRevision{}, fmt.Errorf("revision %s non-goals: %w", row.ID, err)
	}
	return domain.ContractRevision{
		ID:              row.ID,
		OutcomeID:       row.OutcomeID,
		Number:          row.Number,
		Goal:            row.Goal,
		SuccessCriteria: criteria,
		Review:          row.Review,
		Constraints:     constraints,
		NonGoals:        nonGoals,
		Clarification:   row.Clarification,
		CreatedAt:       row.CreatedAt,
	}, nil
}

func marshalJSONStrings(in []string) (string, error) {
	if in == nil {
		in = []string{}
	}
	data, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalJSONStrings(data string) ([]string, error) {
	if data == "" {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}

// AppendPlanRevision atomically persists one proposed plan with its single
// Work Unit and grants, assigning the plan number inside the transaction.
func (s *Store) AppendPlanRevision(ctx context.Context, outcomeID domain.OutcomeID, plan domain.PlanRevision) (domain.PlanRevision, error) {
	if plan.Status != domain.PlanStatusProposed {
		return domain.PlanRevision{}, fmt.Errorf("append plan for %s: only proposed plans are created", outcomeID)
	}
	plan.OutcomeID = outcomeID

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return domain.PlanRevision{}, fmt.Errorf("begin append plan for %s: %w", outcomeID, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	maxNum, err := txq.MaxPlanRevisionNumber(ctx, outcomeID)
	if err != nil {
		return domain.PlanRevision{}, fmt.Errorf("max plan number for %s: %w", outcomeID, err)
	}
	switch v := maxNum.(type) {
	case int64:
		plan.Number = v + 1
	default:
		return domain.PlanRevision{}, fmt.Errorf("max plan number for %s: unexpected type %T", outcomeID, maxNum)
	}
	if err := plan.Validate(); err != nil {
		return domain.PlanRevision{}, err
	}
	if err := txq.CreatePlanRevision(ctx, gen.CreatePlanRevisionParams{
		ID:                     plan.ID,
		OutcomeID:              plan.OutcomeID,
		Number:                 plan.Number,
		ContractRevisionNumber: plan.ContractRevisionNumber,
		Status:                 string(plan.Status),
		Summary:                plan.Summary,
		RunBriefCoreDigest:     plan.RunBriefCoreDigest,
	}); err != nil {
		return domain.PlanRevision{}, fmt.Errorf("create plan revision %s: %w", plan.ID, err)
	}
	for i := range plan.WorkUnits {
		unit := plan.WorkUnits[i]
		checks, err := marshalJSONStrings(unit.EvidenceChecks)
		if err != nil {
			return domain.PlanRevision{}, fmt.Errorf("plan %s evidence checks: %w", plan.ID, err)
		}
		stops, err := marshalJSONStrings(unit.StopConditions)
		if err != nil {
			return domain.PlanRevision{}, fmt.Errorf("plan %s stop conditions: %w", plan.ID, err)
		}
		if err := txq.CreateWorkUnit(ctx, gen.CreateWorkUnitParams{
			ID:                      unit.ID,
			PlanRevisionID:          plan.ID,
			Kind:                    string(unit.Kind),
			Title:                   unit.Title,
			ContractRevisionNumber:  unit.ContractRevisionNumber,
			OutputSummary:           unit.OutputSummary,
			EvidenceChecks:          checks,
			VerificationRequirement: unit.VerificationRequirement,
			StopConditions:          stops,
		}); err != nil {
			return domain.PlanRevision{}, fmt.Errorf("create work unit %s: %w", unit.ID, err)
		}
	}
	for _, grant := range plan.Grants {
		if err := txq.CreateCapabilityGrant(ctx, gen.CreateCapabilityGrantParams{
			ID:             grant.ID,
			PlanRevisionID: plan.ID,
			Name:           grant.Name,
			Scope:          grant.Scope,
		}); err != nil {
			return domain.PlanRevision{}, fmt.Errorf("create capability grant %s: %w", grant.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.PlanRevision{}, fmt.Errorf("commit append plan for %s: %w", outcomeID, err)
	}
	return plan, nil
}

// LatestProposedPlanRevision resolves the highest-numbered proposed plan bound
// to the named contract revision.
func (s *Store) LatestProposedPlanRevision(ctx context.Context, outcomeID domain.OutcomeID, contractRevision int64) (domain.PlanRevision, bool, error) {
	row, err := s.qr.LatestProposedPlanRevision(ctx, gen.LatestProposedPlanRevisionParams{
		OutcomeID:              outcomeID,
		ContractRevisionNumber: contractRevision,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PlanRevision{}, false, nil
	}
	if err != nil {
		return domain.PlanRevision{}, false, fmt.Errorf("latest proposed plan for %s at r%d: %w", outcomeID, contractRevision, err)
	}
	return s.planFromRow(ctx, row)
}

// GetPlanRevision reads one plan with its work unit and grants.
func (s *Store) GetPlanRevision(ctx context.Context, outcomeID domain.OutcomeID, planID domain.PlanRevisionID) (domain.PlanRevision, bool, error) {
	row, err := s.qr.GetPlanRevision(ctx, gen.GetPlanRevisionParams{ID: planID, OutcomeID: outcomeID})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PlanRevision{}, false, nil
	}
	if err != nil {
		return domain.PlanRevision{}, false, fmt.Errorf("get plan %s: %w", planID, err)
	}
	return s.planFromRow(ctx, row)
}

// GetLatestPlanRevision returns the newest plan of any status.
func (s *Store) GetLatestPlanRevision(ctx context.Context, outcomeID domain.OutcomeID) (domain.PlanRevision, bool, error) {
	row, err := s.qr.GetLatestPlanRevision(ctx, outcomeID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PlanRevision{}, false, nil
	}
	if err != nil {
		return domain.PlanRevision{}, false, fmt.Errorf("latest plan for %s: %w", outcomeID, err)
	}
	return s.planFromRow(ctx, row)
}

// ApprovePlanRevision moves a proposed plan to approved under an optimistic
// guard and is idempotent for already-approved plans.
func (s *Store) ApprovePlanRevision(ctx context.Context, outcomeID domain.OutcomeID, planID domain.PlanRevisionID) (domain.PlanRevision, bool, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return domain.PlanRevision{}, false, fmt.Errorf("begin approve plan %s: %w", planID, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	rows, err := txq.ApprovePlanRevision(ctx, gen.ApprovePlanRevisionParams{
		ID:        planID,
		OutcomeID: outcomeID,
	})
	if err != nil {
		return domain.PlanRevision{}, false, fmt.Errorf("approve plan %s: %w", planID, err)
	}
	if rows == 0 {
		row, getErr := txq.GetPlanRevision(ctx, gen.GetPlanRevisionParams{ID: planID, OutcomeID: outcomeID})
		if errors.Is(getErr, sql.ErrNoRows) {
			return domain.PlanRevision{}, false, nil
		}
		if getErr != nil {
			return domain.PlanRevision{}, false, fmt.Errorf("re-read plan %s after guard miss: %w", planID, getErr)
		}
		plan, err := approveReadback(ctx, txq, row)
		if err != nil {
			return domain.PlanRevision{}, true, err
		}
		// Already approved by a concurrent winner; idempotent success.
		return plan, true, tx.Commit()
	}
	row, err := txq.GetPlanRevision(ctx, gen.GetPlanRevisionParams{ID: planID, OutcomeID: outcomeID})
	if err != nil {
		return domain.PlanRevision{}, false, fmt.Errorf("read approved plan %s: %w", planID, err)
	}
	plan, err := approveReadback(ctx, txq, row)
	if err != nil {
		return domain.PlanRevision{}, true, err
	}
	return plan, true, tx.Commit()
}

func approveReadback(ctx context.Context, q *gen.Queries, row gen.PlanRevision) (domain.PlanRevision, error) {
	grants, err := q.ListCapabilityGrantsForPlan(ctx, row.ID)
	if err != nil {
		return domain.PlanRevision{}, fmt.Errorf("list grants for %s: %w", row.ID, err)
	}
	units, err := q.ListWorkUnitsForPlan(ctx, row.ID)
	if err != nil {
		return domain.PlanRevision{}, fmt.Errorf("list work units for %s: %w", row.ID, err)
	}
	return planFromParts(row, units, grants)
}

func (s *Store) planFromRow(ctx context.Context, row gen.PlanRevision) (domain.PlanRevision, bool, error) {
	grants, err := s.qr.ListCapabilityGrantsForPlan(ctx, row.ID)
	if err != nil {
		return domain.PlanRevision{}, true, fmt.Errorf("list grants for %s: %w", row.ID, err)
	}
	units, err := s.qr.ListWorkUnitsForPlan(ctx, row.ID)
	if err != nil {
		return domain.PlanRevision{}, true, fmt.Errorf("list work units for %s: %w", row.ID, err)
	}
	plan, err := planFromParts(row, units, grants)
	return plan, true, err
}

func planFromParts(row gen.PlanRevision, units []gen.WorkUnit, grants []gen.CapabilityGrant) (domain.PlanRevision, error) {
	plan := domain.PlanRevision{
		ID:                     row.ID,
		OutcomeID:              row.OutcomeID,
		Number:                 row.Number,
		ContractRevisionNumber: row.ContractRevisionNumber,
		Status:                 domain.PlanStatus(row.Status),
		Summary:                row.Summary,
		RunBriefCoreDigest:     row.RunBriefCoreDigest,
		RunBriefCompiledDigest: row.RunBriefCompiledDigest,
		CreatedAt:              row.CreatedAt,
	}
	for _, u := range units {
		checks, err := unmarshalJSONStrings(u.EvidenceChecks)
		if err != nil {
			return domain.PlanRevision{}, fmt.Errorf("work unit %s evidence checks: %w", u.ID, err)
		}
		stops, err := unmarshalJSONStrings(u.StopConditions)
		if err != nil {
			return domain.PlanRevision{}, fmt.Errorf("work unit %s stop conditions: %w", u.ID, err)
		}
		plan.WorkUnits = append(plan.WorkUnits, domain.WorkUnit{
			ID:                      u.ID,
			Kind:                    domain.WorkUnitKind(u.Kind),
			Title:                   u.Title,
			ContractRevisionNumber:  u.ContractRevisionNumber,
			OutputSummary:           u.OutputSummary,
			EvidenceChecks:          checks,
			VerificationRequirement: u.VerificationRequirement,
			StopConditions:          stops,
		})
	}
	for _, g := range grants {
		plan.Grants = append(plan.Grants, domain.CapabilityGrant{
			ID:    g.ID,
			Name:  g.Name,
			Scope: g.Scope,
		})
	}
	if err := plan.Validate(); err != nil {
		return domain.PlanRevision{}, fmt.Errorf("plan %s failed readback validation: %w", row.ID, err)
	}
	return plan, nil
}
