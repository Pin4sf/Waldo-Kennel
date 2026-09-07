package domain

import (
	"fmt"
	"strings"
	"time"
)

// OutcomeID identifies one Outcome inside its ResponsibilitySpace.
type OutcomeID string

// IsZero reports whether the id is unset or blank.
func (id OutcomeID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// String returns the raw identifier value.
func (id OutcomeID) String() string {
	return string(id)
}

// ContractRevisionID identifies one immutable contract revision of an Outcome.
type ContractRevisionID string

// IsZero reports whether the id is unset or blank.
func (id ContractRevisionID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// String returns the raw identifier value.
func (id ContractRevisionID) String() string {
	return string(id)
}

// CriterionID identifies one success criterion inside one immutable
// ContractRevision. Its full identity is (ContractRevisionID, CriterionID):
// the same display text in a later revision is deliberately a different
// proof target.
type CriterionID string

// IsZero reports whether the id is unset or blank.
func (id CriterionID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// ContractCriterion is one stable, ordered success criterion.
type ContractCriterion struct {
	ID                 CriterionID
	ContractRevisionID ContractRevisionID
	Position           int64
	Text               string
}

// ContractEvidenceExpectation preserves the evidence shape proposed for one
// stable criterion. Evidence is still created later; these are expectations,
// never proof or acceptance.
type ContractEvidenceExpectation struct {
	CriterionID  CriterionID
	Descriptions []string
}

// Validate checks the criterion's immutable identity and display content.
func (c ContractCriterion) Validate() error {
	if c.ID.IsZero() {
		return fmt.Errorf("contract criterion id is required")
	}
	if c.ContractRevisionID.IsZero() {
		return fmt.Errorf("contract criterion revision id is required")
	}
	if c.Position < 1 {
		return fmt.Errorf("contract criterion position must be at least 1")
	}
	if strings.TrimSpace(c.Text) == "" {
		return fmt.Errorf("contract criterion text is required")
	}
	return nil
}

// Outcome is a durable, human-owned result recorded in one ResponsibilitySpace.
//
// An Outcome deliberately carries no lifecycle status field: provider
// completion, session exit, checks, or commits can never advance or close it.
// Its current stage is derived at read time from durable facts (current
// ContractRevision, PlanRevisions, Attempts, Evidence, Verification), and only
// an explicit owner AcceptanceDecision (#35) may conclude it. Zero
// CurrentRevisionNumber means the contract has not been created yet — never a
// terminal state.
type Outcome struct {
	ID      OutcomeID
	SpaceID ResponsibilitySpaceID
	// ParentID names the Outcome this one contributes to, or is zero for a
	// Project-level Outcome. A contributing Outcome is a full responsibility
	// in its own right — its own contract, plan, Attempts, Evidence,
	// Verification, and AcceptanceDecision — not a task inside its parent
	// (ADR 0007). Composition is capped at CompositionDepthLimit levels, so a
	// contributing Outcome may never itself be a parent.
	ParentID              OutcomeID
	Title                 string
	CurrentRevisionNumber int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// IsContributing reports whether this Outcome contributes to a parent.
func (o Outcome) IsContributing() bool { return !o.ParentID.IsZero() }

// Validate checks intrinsic outcome invariants.
func (o Outcome) Validate() error {
	if o.ID.IsZero() {
		return fmt.Errorf("outcome id is required")
	}
	if o.SpaceID.IsZero() {
		return fmt.Errorf("outcome responsibility space id is required")
	}
	if strings.TrimSpace(o.Title) == "" {
		return fmt.Errorf("outcome title is required")
	}
	if o.CurrentRevisionNumber < 0 {
		return fmt.Errorf("outcome current revision number must not be negative")
	}
	if o.ParentID == o.ID {
		return fmt.Errorf("outcome cannot contribute to itself")
	}
	return nil
}

// ContractRevision is one immutable statement of what an Outcome means: its
// goal, the success criteria that later bind Evidence and Verification, how the
// result will be reviewed, and optional constraints/non-goals plus the single
// material clarification captured during Understand in v0.
//
// Revisions are append-only. A new revision supersedes its predecessor and
// invalidates plans, grants, Evidence, and Verification bound to earlier
// revisions; prior revisions remain readable history and are never updated in
// place.
type ContractRevision struct {
	ID        ContractRevisionID
	OutcomeID OutcomeID
	Number    int64
	Goal      string
	// Criteria is the canonical stable-identity projection introduced by
	// Work E. SuccessCriteria remains the compatibility text projection used
	// by pre-0103 callers and stored JSON; storage writes both atomically.
	Criteria             []ContractCriterion
	SuccessCriteria      []string
	Review               string
	Constraints          []string
	NonGoals             []string
	Clarification        string
	EvidenceExpectations []ContractEvidenceExpectation
	AuthorityCeiling     ProposedAuthority
	StopConditions       []string
	TemporalCondition    *string
	Facets               []ContractFacet
	CreatedAt            time.Time
}

// Validate checks intrinsic revision invariants. Revision-number uniqueness
// per Outcome is enforced by storage alongside immutability.
func (r ContractRevision) Validate() error {
	if r.ID.IsZero() {
		return fmt.Errorf("contract revision id is required")
	}
	if r.OutcomeID.IsZero() {
		return fmt.Errorf("contract revision outcome id is required")
	}
	if r.Number < 1 {
		return fmt.Errorf("contract revision number must be at least 1")
	}
	if strings.TrimSpace(r.Goal) == "" {
		return fmt.Errorf("contract revision goal is required")
	}
	if len(r.SuccessCriteria) == 0 {
		return fmt.Errorf("contract revision requires at least one success criterion")
	}
	for i, criterion := range r.SuccessCriteria {
		if strings.TrimSpace(criterion) == "" {
			return fmt.Errorf("success criterion %d is blank", i+1)
		}
	}
	if len(r.Criteria) > 0 {
		if len(r.Criteria) != len(r.SuccessCriteria) {
			return fmt.Errorf("contract criterion identity count must match success criteria")
		}
		seen := make(map[CriterionID]struct{}, len(r.Criteria))
		for i, criterion := range r.Criteria {
			if err := criterion.Validate(); err != nil {
				return fmt.Errorf("success criterion %d: %w", i+1, err)
			}
			if criterion.ContractRevisionID != r.ID {
				return fmt.Errorf("success criterion %d binds another contract revision", i+1)
			}
			if criterion.Position != int64(i+1) {
				return fmt.Errorf("success criterion %d has unstable position %d", i+1, criterion.Position)
			}
			if strings.TrimSpace(criterion.Text) != strings.TrimSpace(r.SuccessCriteria[i]) {
				return fmt.Errorf("success criterion %d text differs from compatibility projection", i+1)
			}
			if _, ok := seen[criterion.ID]; ok {
				return fmt.Errorf("success criterion %d reuses criterion id %s", i+1, criterion.ID)
			}
			seen[criterion.ID] = struct{}{}
		}
	}
	if strings.TrimSpace(r.Review) == "" {
		return fmt.Errorf("contract revision review is required")
	}
	criterionIDs := make(map[CriterionID]struct{}, len(r.Criteria))
	for _, criterion := range r.Criteria {
		criterionIDs[criterion.ID] = struct{}{}
	}
	for i, expectation := range r.EvidenceExpectations {
		if expectation.CriterionID.IsZero() {
			return fmt.Errorf("evidence expectation %d criterion id is required", i+1)
		}
		if len(r.Criteria) > 0 {
			if _, ok := criterionIDs[expectation.CriterionID]; !ok {
				return fmt.Errorf("evidence expectation %d binds an unknown criterion", i+1)
			}
		}
		if len(expectation.Descriptions) == 0 {
			return fmt.Errorf("evidence expectation %d requires a description", i+1)
		}
		for _, description := range expectation.Descriptions {
			if strings.TrimSpace(description) == "" {
				return fmt.Errorf("evidence expectation %d contains a blank description", i+1)
			}
		}
	}
	for _, stop := range r.StopConditions {
		if strings.TrimSpace(stop) == "" {
			return fmt.Errorf("contract stop condition is blank")
		}
	}
	if r.TemporalCondition != nil && strings.TrimSpace(*r.TemporalCondition) == "" {
		return fmt.Errorf("contract temporal condition is blank")
	}
	for _, facet := range r.Facets {
		if err := facet.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// CriterionTexts returns the ordered criterion display projection. It prefers
// the canonical rows and falls back only for pre-0103 in-memory test fixtures.
func (r ContractRevision) CriterionTexts() []string {
	if len(r.Criteria) == 0 {
		return append([]string(nil), r.SuccessCriteria...)
	}
	out := make([]string, 0, len(r.Criteria))
	for _, criterion := range r.Criteria {
		out = append(out, criterion.Text)
	}
	return out
}
