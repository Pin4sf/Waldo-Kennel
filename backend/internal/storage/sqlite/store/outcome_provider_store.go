package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

const getWorkUnitExecutionBindingSQL = `
SELECT provider, model_selection, model
FROM work_unit_provider_bindings
WHERE work_unit_id = ?`

// GetWorkUnitExecutionBinding reads exact provider/model authority. Historical
// provider-only rows are surfaced as historical_unbound and never upgraded.
func (s *Store) GetWorkUnitExecutionBinding(ctx context.Context, workUnitID domain.WorkUnitID) (domain.ExecutionBinding, bool, error) {
	var provider string
	var selection sql.NullString
	var model sql.NullString
	err := s.readDB.QueryRowContext(ctx, getWorkUnitExecutionBindingSQL, workUnitID).Scan(&provider, &selection, &model)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ExecutionBinding{}, false, nil
	}
	if err != nil {
		return domain.ExecutionBinding{}, false, fmt.Errorf("get execution binding for work unit %s: %w", workUnitID, err)
	}
	binding := domain.ExecutionBinding{Provider: domain.AgentHarness(provider)}
	if selection.Valid {
		binding.ModelSelection = domain.ExecutionBindingModelSelection(selection.String)
		if model.Valid {
			binding.Model = model.String
		}
	} else {
		binding.ModelSelection = domain.ExecutionBindingModelHistoricalUnbound
	}
	if err := binding.ValidateReadable(); err != nil {
		return domain.ExecutionBinding{}, false, err
	}
	return binding, true, nil
}

// GetWorkUnitProvider is a read-only compatibility projection for historical
// callers. New execution admission consumes GetWorkUnitExecutionBinding.
func (s *Store) GetWorkUnitProvider(ctx context.Context, workUnitID domain.WorkUnitID) (domain.AgentHarness, bool, error) {
	binding, found, err := s.GetWorkUnitExecutionBinding(ctx, workUnitID)
	if err != nil || !found {
		return "", found, err
	}
	return binding.Provider, true, nil
}
