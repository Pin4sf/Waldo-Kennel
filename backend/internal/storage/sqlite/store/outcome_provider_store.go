package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
)

const insertWorkUnitProviderBindingSQL = `
INSERT INTO work_unit_provider_bindings (work_unit_id, provider)
VALUES (?, ?)`

const getWorkUnitProviderBindingSQL = `
SELECT provider
FROM work_unit_provider_bindings
WHERE work_unit_id = ?`

// AppendPlanRevisionWithProvider is the provider-bound counterpart of
// AppendPlanRevision. It keeps the existing sqlc plan/work-unit/grant writes,
// adding only the one-to-one provider binding inside the SAME transaction.
// That prevents a crash from leaving a newly proposed plan half-authorized.
//
// This method is intentionally an additive storage capability rather than a
// widening of ports.OutcomeStore: non-SQL test stores keep carrying Provider
// directly in their in-memory domain.PlanRevision values.
func (s *Store) AppendPlanRevisionWithProvider(ctx context.Context, outcomeID domain.OutcomeID, plan domain.PlanRevision) (domain.PlanRevision, error) {
	if plan.Status != domain.PlanStatusProposed {
		return domain.PlanRevision{}, fmt.Errorf("append provider-bound plan for %s: only proposed plans are created", outcomeID)
	}
	plan.OutcomeID = outcomeID
	if len(plan.WorkUnits) != 1 || strings.TrimSpace(string(plan.WorkUnits[0].Provider)) == "" {
		return domain.PlanRevision{}, fmt.Errorf("append provider-bound plan for %s: work unit provider is required", outcomeID)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return domain.PlanRevision{}, fmt.Errorf("begin append provider-bound plan for %s: %w", outcomeID, err)
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

	unit := plan.WorkUnits[0]
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
	if _, err := tx.ExecContext(ctx, insertWorkUnitProviderBindingSQL, unit.ID, string(unit.Provider)); err != nil {
		return domain.PlanRevision{}, fmt.Errorf("bind work unit %s to provider %s: %w", unit.ID, unit.Provider, err)
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
		return domain.PlanRevision{}, fmt.Errorf("commit append provider-bound plan for %s: %w", outcomeID, err)
	}
	return plan, nil
}

// GetWorkUnitProvider returns the immutable provider binding for one WorkUnit.
// ok=false is meaningful legacy state: the WorkUnit predates provider binding
// and must remain readable but cannot be admitted for execution.
func (s *Store) GetWorkUnitProvider(ctx context.Context, workUnitID domain.WorkUnitID) (domain.AgentHarness, bool, error) {
	var provider string
	err := s.readDB.QueryRowContext(ctx, getWorkUnitProviderBindingSQL, workUnitID).Scan(&provider)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get provider binding for work unit %s: %w", workUnitID, err)
	}
	return domain.AgentHarness(provider), true, nil
}
