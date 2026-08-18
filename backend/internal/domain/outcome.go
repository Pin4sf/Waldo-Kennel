package domain

import (
	"fmt"
	"strings"
	"time"
)

// Outcome is a durable, human-defined result that may require several worker
// sessions. It deliberately sits above provider sessions: a worker is evidence
// and execution activity for an outcome, never the outcome's identity.
type Outcome struct {
	ID                 string
	ProjectID          ProjectID
	Title              string
	Definition         string
	AcceptanceCriteria []string
	Status             OutcomeStatus
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// OutcomeStatus is the durable lifecycle of an outcome. UI state such as
// "workers running" remains derived from its tasks and sessions.
type OutcomeStatus string

const (
	OutcomeStatusDraft      OutcomeStatus = "draft"
	OutcomeStatusPlanning   OutcomeStatus = "planning"
	OutcomeStatusInProgress OutcomeStatus = "in_progress"
	OutcomeStatusBlocked    OutcomeStatus = "blocked"
	OutcomeStatusCompleted  OutcomeStatus = "completed"
)

func (s OutcomeStatus) Valid() bool {
	switch s {
	case OutcomeStatusDraft, OutcomeStatusPlanning, OutcomeStatusInProgress, OutcomeStatusBlocked, OutcomeStatusCompleted:
		return true
	default:
		return false
	}
}

// OutcomeTask is the provider-neutral work contract created from an outcome
// plan. SessionID is assigned only after AO has successfully created a worker.
type OutcomeTask struct {
	ID               string
	OutcomeID        string
	Title            string
	Brief            string
	DependsOn        []string
	RequestedHarness AgentHarness
	AssignedHarness  AgentHarness
	SessionID        SessionID
	Status           OutcomeTaskStatus
}

type OutcomeTaskStatus string

const (
	OutcomeTaskStatusPlanned OutcomeTaskStatus = "planned"
	OutcomeTaskStatusRunning OutcomeTaskStatus = "running"
	OutcomeTaskStatusBlocked OutcomeTaskStatus = "blocked"
	OutcomeTaskStatusDone    OutcomeTaskStatus = "done"
)

func (s OutcomeTaskStatus) Valid() bool {
	switch s {
	case OutcomeTaskStatusPlanned, OutcomeTaskStatusRunning, OutcomeTaskStatusBlocked, OutcomeTaskStatusDone:
		return true
	default:
		return false
	}
}

// Validate checks only intrinsic outcome invariants. Dependency and harness
// placement validation belongs to the planning service, where the whole task
// graph and runtime availability are visible.
func (o Outcome) Validate() error {
	if strings.TrimSpace(o.ID) == "" {
		return fmt.Errorf("outcome id is required")
	}
	if strings.TrimSpace(string(o.ProjectID)) == "" {
		return fmt.Errorf("outcome project id is required")
	}
	if strings.TrimSpace(o.Title) == "" {
		return fmt.Errorf("outcome title is required")
	}
	if strings.TrimSpace(o.Definition) == "" {
		return fmt.Errorf("outcome definition is required")
	}
	if !o.Status.Valid() {
		return fmt.Errorf("invalid outcome status %q", o.Status)
	}
	return nil
}
