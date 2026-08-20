package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/service/outcome"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/sqlitetest"
)

func TestCreateOutcomePlanPersistsCompleteGraph(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	project := domain.ProjectRecord{ID: "project", Path: "/tmp/project", DisplayName: "Project", RegisteredAt: time.Now().UTC()}
	if err := s.UpsertProject(ctx, project); err != nil {
		t.Fatalf("UpsertProject: %v", err)
	}
	plan, err := outcome.BuildPlan(domain.Outcome{ID: "outcome", ProjectID: "project", Title: "Ship", Definition: "Ship it.", Status: domain.OutcomeStatusPlanning}, []domain.OutcomeTask{
		{ID: "build", Title: "Build", Brief: "Build it.", DependsOn: []string{"design"}, RequestedHarness: domain.HarnessCodex},
		{ID: "design", Title: "Design", Brief: "Design it."},
	}, []domain.AgentHarness{domain.HarnessCodex})
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if err := s.CreateOutcomePlan(ctx, plan, 1, time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("CreateOutcomePlan: %v", err)
	}
	got, ok, err := s.GetOutcomePlan(ctx, "outcome")
	if err != nil || !ok {
		t.Fatalf("GetOutcomePlan = (%+v, %t, %v)", got, ok, err)
	}
	if got.Outcome.Title != "Ship" || len(got.Tasks) != 2 || got.Tasks[0].DependsOn[0] != "design" {
		t.Fatalf("durable plan = %+v", got)
	}
	if got.Tasks[0].AssignedHarness != domain.HarnessCodex || got.Tasks[0].RequestedHarness != domain.HarnessCodex {
		t.Fatalf("durable harnesses = %+v", got.Tasks)
	}
}
