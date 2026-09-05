package project_test

import (
	"context"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/service/project"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func TestProjectBriefRevisionLifecycle(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitetest.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	m := project.New(store)
	if _, err := m.Add(ctx, project.AddInput{
		Path:      gitRepo(t),
		ProjectID: ptr("brief"),
		Name:      ptr("Brief Project"),
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if current, ok, err := m.CurrentBrief(ctx, "brief"); err != nil || ok {
		t.Fatalf("CurrentBrief before first revision = %#v, %v, %v; want absent", current, ok, err)
	}
	if history, err := m.BriefHistory(ctx, "brief"); err != nil || len(history) != 0 {
		t.Fatalf("BriefHistory before first revision = %#v, %v; want empty", history, err)
	}

	v1, err := m.ReviseBrief(ctx, "brief", project.ReviseBriefInput{
		ExpectedCurrentRevisionNumber: 0,
		Purpose:                       "Make Kennel the durable control plane for outcome-oriented work.",
		ProductContext:                "Users govern outcomes; Kennel coordinates execution.",
		TechnicalContext:              "SQLite is canonical local truth.",
		ArchitectureSummary:           "Project -> Brief -> Outcome -> Contract -> Plan -> WorkUnit.",
		Conventions:                   []string{"provider-neutral domain"},
		Constraints:                   []string{"do not bind responsibility to provider sessions"},
		SetupExpectations:             "npm run bootstrap",
		RunExpectations:               "run daemon and desktop surfaces",
		TestExpectations:              "npm run test:foundation",
		Provenance:                    []string{"user-authored"},
	})
	if err != nil {
		t.Fatalf("ReviseBrief v1: %v", err)
	}
	if v1.Number != 1 || v1.ProjectID != "brief" {
		t.Fatalf("v1 = %#v, want project brief revision 1", v1)
	}

	current, ok, err := m.CurrentBrief(ctx, "brief")
	if err != nil || !ok {
		t.Fatalf("CurrentBrief v1 = %#v, %v, %v", current, ok, err)
	}
	if current.ID != v1.ID || current.Purpose != v1.Purpose {
		t.Fatalf("current v1 = %#v, want %#v", current, v1)
	}

	v2, err := m.ReviseBrief(ctx, "brief", project.ReviseBriefInput{
		ExpectedCurrentRevisionNumber: 1,
		Purpose:                       "Ground every Outcome and Plan in durable Project context.",
		ProductContext:                v1.ProductContext,
		TechnicalContext:              v1.TechnicalContext,
		ArchitectureSummary:           v1.ArchitectureSummary,
		Conventions:                   v1.Conventions,
		Constraints:                   append(v1.Constraints, "Brief changes do not mutate active Contracts"),
		SetupExpectations:             v1.SetupExpectations,
		RunExpectations:               v1.RunExpectations,
		TestExpectations:              v1.TestExpectations,
		Provenance:                    []string{"user-authored", "revised after architecture review"},
	})
	if err != nil {
		t.Fatalf("ReviseBrief v2: %v", err)
	}
	if v2.Number != 2 {
		t.Fatalf("v2.Number = %d, want 2", v2.Number)
	}

	history, err := m.BriefHistory(ctx, "brief")
	if err != nil {
		t.Fatalf("BriefHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("history len = %d, want 2: %#v", len(history), history)
	}
	if history[0].ID != v1.ID || history[0].Purpose != v1.Purpose {
		t.Fatalf("historical v1 changed: %#v", history[0])
	}
	if history[1].ID != v2.ID || history[1].Purpose != v2.Purpose {
		t.Fatalf("historical v2 = %#v, want %#v", history[1], v2)
	}

	_, err = m.ReviseBrief(ctx, "brief", project.ReviseBriefInput{
		ExpectedCurrentRevisionNumber: 1,
		Purpose:                       "Stale writer must not append this revision.",
	})
	wantCode(t, err, "PROJECT_BRIEF_REVISION_CONFLICT")

	history, err = m.BriefHistory(ctx, "brief")
	if err != nil {
		t.Fatalf("BriefHistory after conflict: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("stale write left an orphan revision: history = %#v", history)
	}
	current, ok, err = m.CurrentBrief(ctx, "brief")
	if err != nil || !ok || current.ID != v2.ID {
		t.Fatalf("current after conflict = %#v, %v, %v; want v2", current, ok, err)
	}
}

func TestProjectBriefRejectsInvalidRevisionBeforeStorage(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitetest.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	m := project.New(store)
	if _, err := m.Add(ctx, project.AddInput{
		Path:      gitRepo(t),
		ProjectID: ptr("brief"),
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, err = m.ReviseBrief(ctx, "brief", project.ReviseBriefInput{
		ExpectedCurrentRevisionNumber: 0,
		Purpose:                       "   ",
	})
	wantCode(t, err, "PROJECT_BRIEF_INVALID")

	history, historyErr := m.BriefHistory(ctx, "brief")
	if historyErr != nil {
		t.Fatalf("BriefHistory: %v", historyErr)
	}
	if len(history) != 0 {
		t.Fatalf("invalid write reached storage: %#v", history)
	}
}
