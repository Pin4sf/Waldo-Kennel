package domain

import (
	"strings"
	"testing"
	"time"
)

func validProjectBriefRevision() ProjectBriefRevision {
	return ProjectBriefRevision{
		ID:                  ProjectBriefRevisionID("pbr_1"),
		ProjectID:           ProjectID("p1"),
		Number:              1,
		Purpose:             "Make Kennel the durable control plane for outcome-oriented work.",
		ProductContext:      "Users govern outcomes; Kennel coordinates execution.",
		TechnicalContext:    "SQLite is canonical local truth.",
		ArchitectureSummary: "Project -> Brief -> Outcome -> Contract -> Plan -> WorkUnit.",
		Conventions:         []string{"provider-neutral domain"},
		Constraints:         []string{"do not bind responsibility to provider sessions"},
		SetupExpectations:   "npm run bootstrap",
		RunExpectations:     "run daemon and desktop surfaces",
		TestExpectations:    "npm run test:foundation",
		Provenance:          []string{"user-authored"},
		CreatedAt:           time.Unix(1, 0).UTC(),
	}
}

func TestProjectBriefRevisionValidate(t *testing.T) {
	if err := validProjectBriefRevision().Validate(); err != nil {
		t.Fatalf("valid revision rejected: %v", err)
	}
}

func TestProjectBriefRevisionValidateRejectsMissingIdentityAndPurpose(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ProjectBriefRevision)
		want   string
	}{
		{"id", func(r *ProjectBriefRevision) { r.ID = "" }, "id is required"},
		{"project", func(r *ProjectBriefRevision) { r.ProjectID = "" }, "project id is required"},
		{"number", func(r *ProjectBriefRevision) { r.Number = 0 }, "number must be at least 1"},
		{"purpose", func(r *ProjectBriefRevision) { r.Purpose = "  " }, "purpose is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := validProjectBriefRevision()
			tt.mutate(&r)
			err := r.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestProjectBriefRevisionValidateRejectsBlankListEntries(t *testing.T) {
	r := validProjectBriefRevision()
	r.Constraints = []string{"must stay local-first", "  "}
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "constraint 2 is blank") {
		t.Fatalf("Validate() error = %v", err)
	}
}
