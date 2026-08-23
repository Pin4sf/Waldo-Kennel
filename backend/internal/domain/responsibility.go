package domain

import (
	"fmt"
	"strings"
	"time"
)

// ResponsibilitySpaceID identifies one responsibility space: the bounded area
// in which one kind of responsibility is recorded and closed. It is the shared
// identity other lanes (Home, Learning) consume; it never merges two spaces'
// lineages into one another.
type ResponsibilitySpaceID string

// IsZero reports whether the id is unset or blank.
func (id ResponsibilitySpaceID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// String returns the raw identifier value.
func (id ResponsibilitySpaceID) String() string {
	return string(id)
}

// ResponsibilitySpaceKind names which responsibility ontology a space carries.
// v0 admits only project-backed Work spaces; Home-owned kinds belong to their
// separately planned lane and are intentionally not modeled here yet.
type ResponsibilitySpaceKind string

// ResponsibilitySpaceWorkProject is the project-backed Work responsibility
// space: Outcomes for one registered Project live in exactly one such space.
const ResponsibilitySpaceWorkProject ResponsibilitySpaceKind = "WorkProject"

// Valid reports whether k is a supported space kind.
func (k ResponsibilitySpaceKind) Valid() bool {
	return k == ResponsibilitySpaceWorkProject
}

// ResponsibilitySpace is the durable root of one bounded responsibility area.
// The daemon owns its identity; renderers may only reference it.
type ResponsibilitySpace struct {
	ID        ResponsibilitySpaceID
	Kind      ResponsibilitySpaceKind
	ProjectID ProjectID
	CreatedAt time.Time
}

// Validate checks intrinsic space invariants. Uniqueness of the project-backed
// Work space per Project is enforced by storage, where concurrent creation is
// observable.
func (s ResponsibilitySpace) Validate() error {
	if s.ID.IsZero() {
		return fmt.Errorf("responsibility space id is required")
	}
	if !s.Kind.Valid() {
		return fmt.Errorf("unsupported responsibility space kind %q", s.Kind)
	}
	if strings.TrimSpace(string(s.ProjectID)) == "" {
		return fmt.Errorf("responsibility space project id is required")
	}
	return nil
}
