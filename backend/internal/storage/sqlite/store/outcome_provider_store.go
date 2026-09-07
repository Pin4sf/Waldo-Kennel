package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
)

const insertWorkUnitExecutionBindingSQL = `
INSERT INTO work_unit_provider_bindings (work_unit_id, provider, model_selection, model)
VALUES (?, ?, ?, ?)`

const insertLegacyWorkUnitProviderBindingSQL = `
INSERT INTO work_unit_provider_bindings (work_unit_id, provider)
VALUES (?, ?)`

const getWorkUnitExecutionBindingSQL = `
SELECT provider, model_selection, model
FROM work_unit_provider_bindings
WHERE work_unit_id = ?`

// AppendPlanRevisionWithExecution is the WT3 append path. Plan routing
// provenance and the exact WorkUnit provider/model binding land in the SAME
// transaction as the immutable proposal.
func (s *Store) AppendPlanRevisionWithExecution(ctx context.Context, outcomeID domain.OutcomeID, plan domain.PlanRevision) (domain.PlanRevision, error) {
	if plan.Status != domain.PlanStatusProposed {
		return domain.PlanRevision{}, fmt.Errorf("append execution-bound plan for %s: only proposed plans are created", outcomeID)
	}
	plan.OutcomeID = outcomeID
	if len(plan.WorkUnits) != 1 {
		return domain.PlanRevision{}, fmt.Errorf("append execution-bound plan for %s: exactly one work unit is required", outcomeID)
	}
	binding, err := plan.WorkUnits[0].ExecutionBindingForNewWork()
	if err != nil {
		return domain.PlanRevision{}, fmt.Errorf("append execution-bound plan for %s: %w", outcomeID, err)
	}
	if plan.RoutingDecision == nil || plan.RoutingDecision.Status != domain.RoutingDecisionRecommended {
		return domain.PlanRevision{}, fmt.Errorf("append execution-bound plan for %s: routing recommendation is required", outcomeID)
	}
	routingJSON, err := json.Marshal(plan.RoutingDecision)
	if err != nil {
		return domain.PlanRevision{}, fmt.Errorf("encode routing decision for %s: %w", plan.ID, err)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return domain.PlanRevision{}, fmt.Errorf("begin append execution-bound plan for %s: %w", outcomeID, err)
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
	if _, err := tx.ExecContext(ctx, `
INSERT INTO plan_revisions
    (id, outcome_id, number, contract_revision_number, status, summary, run_brief_core_digest, run_brief_compiled_digest, routing_decision_json)
VALUES (?, ?, ?, ?, ?, ?, ?, '', ?)`,
		plan.ID, plan.OutcomeID, plan.Number, plan.ContractRevisionNumber, string(plan.Status),
		plan.Summary, plan.RunBriefCoreDigest, string(routingJSON),
	); err != nil {
		return domain.PlanRevision{}, fmt.Errorf("create plan revision %s: %w", plan.ID, err)
	}

	unit := plan.WorkUnits[0]
	checks, err := marshalJSONStrings(unit.EvidenceChecks)
	if err != nil { return domain.PlanRevision{}, fmt.Errorf("plan %s evidence checks: %w", plan.ID, err) }
	stops, err := marshalJSONStrings(unit.StopConditions)
	if err != nil { return domain.PlanRevision{}, fmt.Errorf("plan %s stop conditions: %w", plan.ID, err) }
	if err := txq.CreateWorkUnit(ctx, gen.CreateWorkUnitParams{
		ID: unit.ID, PlanRevisionID: plan.ID, Kind: string(unit.Kind), Title: unit.Title,
		ContractRevisionNumber: unit.ContractRevisionNumber, OutputSummary: unit.OutputSummary,
		EvidenceChecks: checks, VerificationRequirement: unit.VerificationRequirement, StopConditions: stops,
	}); err != nil {
		return domain.PlanRevision{}, fmt.Errorf("create work unit %s: %w", unit.ID, err)
	}
	var model any
	if binding.ModelSelection == domain.ExecutionBindingModelExplicit {
		model = binding.Model
	}
	if _, err := tx.ExecContext(ctx, insertWorkUnitExecutionBindingSQL,
		unit.ID, string(binding.Provider), string(binding.ModelSelection), model,
	); err != nil {
		return domain.PlanRevision{}, fmt.Errorf("bind work unit %s execution: %w", unit.ID, err)
	}
	for _, grant := range plan.Grants {
		if err := txq.CreateCapabilityGrant(ctx, gen.CreateCapabilityGrantParams{
			ID: grant.ID, PlanRevisionID: plan.ID, Name: grant.Name, Scope: grant.Scope,
		}); err != nil {
			return domain.PlanRevision{}, fmt.Errorf("create capability grant %s: %w", grant.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.PlanRevision{}, fmt.Errorf("commit append execution-bound plan for %s: %w", outcomeID, err)
	}
	return plan, nil
}

// AppendPlanRevisionWithProvider preserves the pre-WT3 storage capability for
// historical/provider-binding tests. It deliberately records no model
// semantics, so the resulting row is readable but not executable by WT3.
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
	if err != nil { return domain.PlanRevision{}, fmt.Errorf("begin append provider-bound plan for %s: %w", outcomeID, err) }
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)
	maxNum, err := txq.MaxPlanRevisionNumber(ctx, outcomeID)
	if err != nil { return domain.PlanRevision{}, fmt.Errorf("max plan number for %s: %w", outcomeID, err) }
	switch v := maxNum.(type) {
	case int64: plan.Number = v + 1
	default: return domain.PlanRevision{}, fmt.Errorf("max plan number for %s: unexpected type %T", outcomeID, maxNum)
	}
	if err := plan.Validate(); err != nil { return domain.PlanRevision{}, err }
	if err := txq.CreatePlanRevision(ctx, gen.CreatePlanRevisionParams{
		ID: plan.ID, OutcomeID: plan.OutcomeID, Number: plan.Number,
		ContractRevisionNumber: plan.ContractRevisionNumber, Status: string(plan.Status),
		Summary: plan.Summary, RunBriefCoreDigest: plan.RunBriefCoreDigest,
	}); err != nil { return domain.PlanRevision{}, fmt.Errorf("create plan revision %s: %w", plan.ID, err) }
	unit := plan.WorkUnits[0]
	checks, err := marshalJSONStrings(unit.EvidenceChecks)
	if err != nil { return domain.PlanRevision{}, err }
	stops, err := marshalJSONStrings(unit.StopConditions)
	if err != nil { return domain.PlanRevision{}, err }
	if err := txq.CreateWorkUnit(ctx, gen.CreateWorkUnitParams{
		ID: unit.ID, PlanRevisionID: plan.ID, Kind: string(unit.Kind), Title: unit.Title,
		ContractRevisionNumber: unit.ContractRevisionNumber, OutputSummary: unit.OutputSummary,
		EvidenceChecks: checks, VerificationRequirement: unit.VerificationRequirement, StopConditions: stops,
	}); err != nil { return domain.PlanRevision{}, fmt.Errorf("create work unit %s: %w", unit.ID, err) }
	if _, err := tx.ExecContext(ctx, insertLegacyWorkUnitProviderBindingSQL, unit.ID, string(unit.Provider)); err != nil {
		return domain.PlanRevision{}, fmt.Errorf("bind work unit %s to provider %s: %w", unit.ID, unit.Provider, err)
	}
	for _, grant := range plan.Grants {
		if err := txq.CreateCapabilityGrant(ctx, gen.CreateCapabilityGrantParams{ID: grant.ID, PlanRevisionID: plan.ID, Name: grant.Name, Scope: grant.Scope}); err != nil {
			return domain.PlanRevision{}, fmt.Errorf("create capability grant %s: %w", grant.ID, err)
		}
	}
	if err := tx.Commit(); err != nil { return domain.PlanRevision{}, err }
	return plan, nil
}

// GetWorkUnitExecutionBinding reads exact provider/model authority. Provider-
// only history is surfaced as historical_unbound and is never auto-upgraded.
func (s *Store) GetWorkUnitExecutionBinding(ctx context.Context, workUnitID domain.WorkUnitID) (domain.ExecutionBinding, bool, error) {
	var provider string
	var selection sql.NullString
	var model sql.NullString
	err := s.readDB.QueryRowContext(ctx, getWorkUnitExecutionBindingSQL, workUnitID).Scan(&provider, &selection, &model)
	if errors.Is(err, sql.ErrNoRows) { return domain.ExecutionBinding{}, false, nil }
	if err != nil { return domain.ExecutionBinding{}, false, fmt.Errorf("get execution binding for work unit %s: %w", workUnitID, err) }
	binding := domain.ExecutionBinding{Provider: domain.AgentHarness(provider)}
	if selection.Valid {
		binding.ModelSelection = domain.ExecutionBindingModelSelection(selection.String)
		if model.Valid { binding.Model = model.String }
	} else {
		binding.ModelSelection = domain.ExecutionBindingModelHistoricalUnbound
	}
	if err := binding.ValidateReadable(); err != nil { return domain.ExecutionBinding{}, false, err }
	return binding, true, nil
}

// GetPlanRoutingDecision reads persisted recommendation provenance. Old Plans
// return nil rather than fabricated routing history.
func (s *Store) GetPlanRoutingDecision(ctx context.Context, planID domain.PlanRevisionID) (*domain.RoutingDecision, error) {
	var raw sql.NullString
	err := s.readDB.QueryRowContext(ctx, `SELECT routing_decision_json FROM plan_revisions WHERE id = ?`, planID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) || !raw.Valid || raw.String == "" { return nil, nil }
	if err != nil { return nil, fmt.Errorf("get routing decision for plan %s: %w", planID, err) }
	var decision domain.RoutingDecision
	if err := json.Unmarshal([]byte(raw.String), &decision); err != nil { return nil, fmt.Errorf("decode routing decision for plan %s: %w", planID, err) }
	return &decision, nil
}

// GetWorkUnitProvider is the compatibility projection used by pre-WT3 readers.
func (s *Store) GetWorkUnitProvider(ctx context.Context, workUnitID domain.WorkUnitID) (domain.AgentHarness, bool, error) {
	binding, found, err := s.GetWorkUnitExecutionBinding(ctx, workUnitID)
	if err != nil || !found { return "", found, err }
	return binding.Provider, true, nil
}
