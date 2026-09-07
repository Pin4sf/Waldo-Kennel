package domain

import (
	"fmt"
	"strings"
	"time"
)

// ProjectBriefRevisionID identifies one immutable Project Brief revision.
type ProjectBriefRevisionID string

// IsZero reports whether the id is unset or blank.
func (id ProjectBriefRevisionID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// String returns the raw identifier value.
func (id ProjectBriefRevisionID) String() string { return string(id) }

// ProjectBriefRevision is one immutable snapshot of durable, user-governed
// Project context. It grounds Outcome creation and planning, but carries no
// Outcome authority, execution state, proof state, or acceptance state.
type ProjectBriefRevision struct {
	ID                  ProjectBriefRevisionID
	ProjectID           ProjectID
	Number              int64
	Purpose             string
	ProductContext      string
	TechnicalContext    string
	ArchitectureSummary string
	Conventions         []string
	Constraints         []string
	SetupExpectations   string
	RunExpectations     string
	TestExpectations    string
	Provenance          []string
	CreatedAt           time.Time
}

// Validate checks intrinsic revision invariants. Revision-number uniqueness
// per Project and append-only immutability are enforced by storage.
func (r ProjectBriefRevision) Validate() error {
	if r.ID.IsZero() {
		return fmt.Errorf("project brief revision id is required")
	}
	if strings.TrimSpace(string(r.ProjectID)) == "" {
		return fmt.Errorf("project brief revision project id is required")
	}
	if r.Number < 1 {
		return fmt.Errorf("project brief revision number must be at least 1")
	}
	if strings.TrimSpace(r.Purpose) == "" {
		return fmt.Errorf("project brief revision purpose is required")
	}
	if err := validateProjectBriefStrings("convention", r.Conventions); err != nil {
		return err
	}
	if err := validateProjectBriefStrings("constraint", r.Constraints); err != nil {
		return err
	}
	if err := validateProjectBriefStrings("provenance entry", r.Provenance); err != nil {
		return err
	}
	return nil
}

func validateProjectBriefStrings(label string, values []string) error {
	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("project brief revision %s %d is blank", label, i+1)
		}
	}
	return nil
}
