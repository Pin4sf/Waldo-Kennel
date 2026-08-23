package domain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResponsibilitySpaceValidation(t *testing.T) {
	tests := []struct {
		name    string
		space   ResponsibilitySpace
		wantErr string
	}{
		{
			name: "valid work project space",
			space: ResponsibilitySpace{
				ID:        "rsp_01",
				Kind:      ResponsibilitySpaceWorkProject,
				ProjectID: "proj-1",
				CreatedAt: time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC),
			},
		},
		{name: "missing id", space: ResponsibilitySpace{Kind: ResponsibilitySpaceWorkProject, ProjectID: "proj-1"}, wantErr: "responsibility space id is required"},
		{name: "whitespace id", space: ResponsibilitySpace{ID: "   ", Kind: ResponsibilitySpaceWorkProject, ProjectID: "proj-1"}, wantErr: "responsibility space id is required"},
		{name: "empty project", space: ResponsibilitySpace{ID: "rsp_01", Kind: ResponsibilitySpaceWorkProject}, wantErr: "responsibility space project id is required"},
		{name: "unknown kind", space: ResponsibilitySpace{ID: "rsp_01", Kind: ResponsibilitySpaceKind("PersonalHome"), ProjectID: "proj-1"}, wantErr: "unsupported responsibility space kind"},
		{
			name:    "v0 admits only the work project kind",
			space:   ResponsibilitySpace{ID: "rsp_01", Kind: ResponsibilitySpaceKind(""), ProjectID: "proj-1"},
			wantErr: "unsupported responsibility space kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.space.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestOutcomeValidation(t *testing.T) {
	valid := Outcome{
		ID:                    "out_01",
		SpaceID:               "rsp_01",
		Title:                 "Local Focus Ledger",
		CurrentRevisionNumber: 1,
		CreatedAt:             time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC),
		UpdatedAt:             time.Date(2026, 8, 23, 10, 5, 0, 0, time.UTC),
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}

	tests := []struct {
		name    string
		mutate  func(*Outcome)
		wantErr string
	}{
		{name: "missing outcome id", mutate: func(o *Outcome) { o.ID = "" }, wantErr: "outcome id is required"},
		{name: "missing space", mutate: func(o *Outcome) { o.SpaceID = "" }, wantErr: "outcome responsibility space id is required"},
		{name: "blank title", mutate: func(o *Outcome) { o.Title = "   " }, wantErr: "outcome title is required"},
		{
			name:    "negative revision pointer",
			mutate:  func(o *Outcome) { o.CurrentRevisionNumber = -1 },
			wantErr: "outcome current revision number must not be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := valid
			tt.mutate(&candidate)
			err := candidate.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestOutcomeBeforeFirstRevisionIsValid(t *testing.T) {
	// An Outcome row may exist for the brief moment between creation and its
	// ContractRevision 1 commit; zero means "no contract yet", never a
	// provider-completion state.
	outcome := Outcome{ID: "out_01", SpaceID: "rsp_01", Title: "Local Focus Ledger"}
	if err := outcome.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil for pre-contract outcome", err)
	}
}

func TestContractRevisionValidation(t *testing.T) {
	valid := ContractRevision{
		ID:              "cr_01",
		OutcomeID:       "out_01",
		Number:          1,
		Goal:            "A user can record and review today's protected focus time locally.",
		SuccessCriteria: []string{"Entering a positive whole-minute duration creates one focus block."},
		Review:          "Deterministic checks plus owner walkthrough.",
		CreatedAt:       time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC),
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}

	tests := []struct {
		name    string
		mutate  func(*ContractRevision)
		wantErr string
	}{
		{name: "missing revision id", mutate: func(r *ContractRevision) { r.ID = "" }, wantErr: "contract revision id is required"},
		{name: "missing outcome", mutate: func(r *ContractRevision) { r.OutcomeID = "" }, wantErr: "contract revision outcome id is required"},
		{name: "revision numbers start at one", mutate: func(r *ContractRevision) { r.Number = 0 }, wantErr: "contract revision number must be at least 1"},
		{name: "negative revision number", mutate: func(r *ContractRevision) { r.Number = -2 }, wantErr: "contract revision number must be at least 1"},
		{name: "goal is required", mutate: func(r *ContractRevision) { r.Goal = "" }, wantErr: "contract revision goal is required"},
		{name: "whitespace goal", mutate: func(r *ContractRevision) { r.Goal = "  \t " }, wantErr: "contract revision goal is required"},
		{name: "at least one success criterion", mutate: func(r *ContractRevision) { r.SuccessCriteria = nil }, wantErr: "contract revision requires at least one success criterion"},
		{name: "criteria may not be blank", mutate: func(r *ContractRevision) { r.SuccessCriteria = []string{"real criterion", "   "} }, wantErr: "success criterion 2 is blank"},
		{name: "review is required", mutate: func(r *ContractRevision) { r.Review = "" }, wantErr: "contract revision review is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := valid
			tt.mutate(&candidate)
			err := candidate.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestContractRevisionOptionalFieldsMayBeEmpty(t *testing.T) {
	// Constraints, non-goals, and the single v0 material clarification are
	// optional at the domain boundary; persistence stores them verbatim.
	revision := ContractRevision{
		ID:              "cr_02",
		OutcomeID:       "out_01",
		Number:          2,
		Goal:            "Revised goal.",
		SuccessCriteria: []string{"Criterion A.", "Criterion B."},
		Review:          "Deterministic checks.",
	}
	if err := revision.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

// TestDonorOutcomeSemanticsAreRetired guards the #21 boundary: the inherited
// provider-oriented truth overlay (`OutcomeTask`, the `completed` lifecycle
// state, and the unwired marker-era plan builder) must not return to canonical
// domain code. Acceptance belongs to AcceptanceDecision (#35), never to an
// Outcome status field.
func TestDonorOutcomeSemanticsAreRetired(t *testing.T) {
	banned := []string{
		"OutcomeStatusCompleted",
		"OutcomeTaskStatusDone",
		"type OutcomeTask struct",
		"BuildPlan(",
	}

	roots := map[string]string{
		"domain":           ".",
		"service/outcome":  "../service/outcome",
	}

	for label, dir := range roots {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s package dir: %v", label, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("read %s/%s: %v", label, name, err)
			}
			for _, symbol := range banned {
				if strings.Contains(string(data), symbol) {
					t.Errorf("%s/%s contains retired donor symbol %q", label, name, symbol)
				}
			}
		}
	}
}
