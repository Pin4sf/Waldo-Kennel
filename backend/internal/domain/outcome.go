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
	// OutcomeStatusDraft is an outcome that has not entered planning.
	OutcomeStatusDraft OutcomeStatus = "draft"
	// OutcomeStatusPlanning is an outcome whose execution plan is being defined.
	OutcomeStatusPlanning OutcomeStatus = "planning"
	// OutcomeStatusInProgress is an outcome with active execution work.
	OutcomeStatusInProgress OutcomeStatus = "in_progress"
	// OutcomeStatusBlocked is an outcome that requires an input or dependency.
	OutcomeStatusBlocked OutcomeStatus = "blocked"
	// OutcomeStatusCompleted is an outcome whose acceptance contract is satisfied.
	OutcomeStatusCompleted OutcomeStatus = "completed"
)

// Valid reports whether s is a recognized durable outcome lifecycle value.
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

// OutcomeTaskStatus is the durable execution state of one outcome task.
type OutcomeTaskStatus string

const (
	// OutcomeTaskStatusPlanned is a task that has not begun execution.
	OutcomeTaskStatusPlanned OutcomeTaskStatus = "planned"
	// OutcomeTaskStatusRunning is a task with active execution work.
	OutcomeTaskStatusRunning OutcomeTaskStatus = "running"
	// OutcomeTaskStatusBlocked is a task waiting on an input or dependency.
	OutcomeTaskStatusBlocked OutcomeTaskStatus = "blocked"
	// OutcomeTaskStatusDone is a task whose work has finished.
	OutcomeTaskStatusDone OutcomeTaskStatus = "done"
)

// Valid reports whether s is a recognized durable outcome-task state.
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
