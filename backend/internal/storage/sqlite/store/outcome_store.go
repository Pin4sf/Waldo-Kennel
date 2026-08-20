package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/service/outcome"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/gen"
)

// CreateOutcomePlan durably records one validated, proposed revision and its
// complete task graph. The transaction means a partially written graph can
// never become eligible for approval or worker assignment.
func (s *Store) CreateOutcomePlan(ctx context.Context, plan outcome.Plan, revision int64, at time.Time) error {
	if revision < 1 {
		return fmt.Errorf("outcome plan revision must be positive")
	}
	if err := plan.Outcome.Validate(); err != nil {
		return err
	}
	if len(plan.Tasks) == 0 {
		return fmt.Errorf("outcome plan requires at least one task")
	}
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("marshal outcome plan: %w", err)
	}
	at = at.UTC()
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.inTx(ctx, "create outcome plan", func(q *gen.Queries) error {
		if err := q.CreateOutcome(ctx, gen.CreateOutcomeParams{
			ID: plan.Outcome.ID, ProjectID: string(plan.Outcome.ProjectID), Title: plan.Outcome.Title,
			Definition: plan.Outcome.Definition, Status: string(plan.Outcome.Status), CreatedAt: formatOutcomeTime(at), UpdatedAt: formatOutcomeTime(at),
		}); err != nil {
			return err
		}
		if err := q.CreateOutcomePlanRevision(ctx, gen.CreateOutcomePlanRevisionParams{
			OutcomeID: plan.Outcome.ID, Revision: revision, PlanJson: string(planJSON), CreatedAt: formatOutcomeTime(at),
		}); err != nil {
			return err
		}
		for _, task := range plan.Tasks {
			if err := q.CreateOutcomeTask(ctx, gen.CreateOutcomeTaskParams{
				ID: task.ID, OutcomeID: task.OutcomeID, Title: task.Title, Brief: task.Brief,
				RequestedHarness: nullOutcomeString(string(task.RequestedHarness)), AssignedHarness: nullOutcomeString(string(task.AssignedHarness)),
				Status: string(task.Status), HoldReason: nullOutcomeString(task.HoldReason), CreatedAt: formatOutcomeTime(at), UpdatedAt: formatOutcomeTime(at),
			}); err != nil {
				return err
			}
		}
		// Plans retain the proposer-visible task order. Dependencies may point
		// forward in that order, so insert their edges only once every target
		// task row exists.
		for _, task := range plan.Tasks {
			for _, dependency := range task.DependsOn {
				if err := q.CreateOutcomeTaskDependency(ctx, gen.CreateOutcomeTaskDependencyParams{TaskID: task.ID, DependsOnTaskID: dependency}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// GetOutcomePlan returns the durable graph used by future daemon-owned
// approval and assignment operations. Checks and evidence are intentionally
// excluded until their write paths are introduced.
func (s *Store) GetOutcomePlan(ctx context.Context, id string) (outcome.Plan, bool, error) {
	row, err := s.qr.GetOutcome(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return outcome.Plan{}, false, nil
	}
	if err != nil {
		return outcome.Plan{}, false, fmt.Errorf("get outcome %s: %w", id, err)
	}
	tasks, err := s.qr.ListOutcomeTasks(ctx, id)
	if err != nil {
		return outcome.Plan{}, false, fmt.Errorf("list outcome tasks %s: %w", id, err)
	}
	dependencies, err := s.qr.ListOutcomeTaskDependencies(ctx, id)
	if err != nil {
		return outcome.Plan{}, false, fmt.Errorf("list outcome task dependencies %s: %w", id, err)
	}
	createdAt, err := parseOutcomeTime(row.CreatedAt)
	if err != nil {
		return outcome.Plan{}, false, fmt.Errorf("parse outcome %s created_at: %w", id, err)
	}
	updatedAt, err := parseOutcomeTime(row.UpdatedAt)
	if err != nil {
		return outcome.Plan{}, false, fmt.Errorf("parse outcome %s updated_at: %w", id, err)
	}
	byID := make(map[string]int, len(tasks))
	result := outcome.Plan{Outcome: domain.Outcome{ID: row.ID, ProjectID: domain.ProjectID(row.ProjectID), Title: row.Title, Definition: row.Definition, Status: domain.OutcomeStatus(row.Status), CreatedAt: createdAt, UpdatedAt: updatedAt}}
	for _, task := range tasks {
		item := domain.OutcomeTask{ID: task.ID, OutcomeID: task.OutcomeID, Title: task.Title, Brief: task.Brief, Status: domain.OutcomeTaskStatus(task.Status), HoldReason: task.HoldReason.String}
		if task.RequestedHarness.Valid {
			item.RequestedHarness = domain.AgentHarness(task.RequestedHarness.String)
		}
		if task.AssignedHarness.Valid {
			item.AssignedHarness = domain.AgentHarness(task.AssignedHarness.String)
		}
		if task.WorkerSessionID.Valid {
			item.SessionID = domain.SessionID(task.WorkerSessionID.String)
		}
		result.Tasks = append(result.Tasks, item)
		byID[item.ID] = len(result.Tasks) - 1
	}
	for _, dependency := range dependencies {
		if index, ok := byID[dependency.TaskID]; ok {
			result.Tasks[index].DependsOn = append(result.Tasks[index].DependsOn, dependency.DependsOnTaskID)
		}
	}
	return result, true, nil
}

func nullOutcomeString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func formatOutcomeTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func parseOutcomeTime(value string) (time.Time, error) { return time.Parse(time.RFC3339Nano, value) }
