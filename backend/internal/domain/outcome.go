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
	HoldReason       string
	Decision         *HumanDecisionRequest
	Checks           []OutcomeCheck
	Evidence         []OutcomeEvidence
	Transitions      []OutcomeTaskTransition
}

type OutcomeTaskStatus string

const (
	OutcomeTaskStatusPlanned               OutcomeTaskStatus = "planned"
	OutcomeTaskStatusAssigned              OutcomeTaskStatus = "assigned"
	OutcomeTaskStatusWorking               OutcomeTaskStatus = "working"
	OutcomeTaskStatusOnHold                OutcomeTaskStatus = "on_hold"
	OutcomeTaskStatusAwaitingHumanDecision OutcomeTaskStatus = "awaiting_human_decision"
	OutcomeTaskStatusInReview              OutcomeTaskStatus = "in_review"
	OutcomeTaskStatusReadyToMerge          OutcomeTaskStatus = "ready_to_merge"
	OutcomeTaskStatusRework                OutcomeTaskStatus = "rework"
	OutcomeTaskStatusFailed                OutcomeTaskStatus = "failed"
	OutcomeTaskStatusAccepted              OutcomeTaskStatus = "accepted"
	// Deprecated compatibility values for pre-Outcome marker state. New code
	// must use the explicit lifecycle states above.
	OutcomeTaskStatusRunning OutcomeTaskStatus = OutcomeTaskStatusWorking
	OutcomeTaskStatusBlocked OutcomeTaskStatus = OutcomeTaskStatusOnHold
	OutcomeTaskStatusDone    OutcomeTaskStatus = OutcomeTaskStatusAccepted
)

func (s OutcomeTaskStatus) Valid() bool {
	switch s {
	case OutcomeTaskStatusPlanned, OutcomeTaskStatusAssigned, OutcomeTaskStatusWorking, OutcomeTaskStatusOnHold,
		OutcomeTaskStatusAwaitingHumanDecision, OutcomeTaskStatusInReview, OutcomeTaskStatusReadyToMerge,
		OutcomeTaskStatusRework, OutcomeTaskStatusFailed, OutcomeTaskStatusAccepted:
		return true
	default:
		return false
	}
}

// OutcomeCheck is an approved, machine-reviewable or human-reviewable
// completion condition. A task enters review only when every required check
// has submitted evidence (or an explicitly recorded exception).
type OutcomeCheck struct {
	ID        string
	Statement string
	Required  bool
	Accepted  bool
}

// OutcomeEvidence is an immutable claim submitted by the assigned worker.
// It records a reference rather than retaining transcripts or artifact bodies.
type OutcomeEvidence struct {
	CheckID     string
	Reference   string
	SubmittedAt time.Time
}

// HumanDecisionRequest is the only state that can place a task in Needs you.
// Provider auth and permission prompts intentionally do not use this type.
type HumanDecisionRequest struct {
	Question       string
	Context        string
	Options        []HumanDecisionOption
	Recommendation string
	ResumeTarget   string
	ResolvedAt     *time.Time
	Answer         string
}

type HumanDecisionOption struct {
	ID       string
	Label    string
	Tradeoff string
}

// OutcomeTaskTransition supplies a compact durable audit of server-owned
// lifecycle changes. The daemon, never a free-form agent marker, appends it.
type OutcomeTaskTransition struct {
	From      OutcomeTaskStatus
	To        OutcomeTaskStatus
	Reason    string
	ChangedAt time.Time
}

// CanTransitionTo implements the evidence-gated task state machine. It keeps
// raw harness activity signals subordinate to task lifecycle authority.
func (s OutcomeTaskStatus) CanTransitionTo(next OutcomeTaskStatus) bool {
	if !s.Valid() || !next.Valid() || s == next {
		return false
	}
	switch s {
	case OutcomeTaskStatusPlanned:
		return next == OutcomeTaskStatusAssigned || next == OutcomeTaskStatusFailed
	case OutcomeTaskStatusAssigned:
		return next == OutcomeTaskStatusWorking || next == OutcomeTaskStatusOnHold || next == OutcomeTaskStatusFailed
	case OutcomeTaskStatusWorking:
		return next == OutcomeTaskStatusOnHold || next == OutcomeTaskStatusAwaitingHumanDecision || next == OutcomeTaskStatusInReview || next == OutcomeTaskStatusRework || next == OutcomeTaskStatusFailed
	case OutcomeTaskStatusOnHold:
		return next == OutcomeTaskStatusWorking || next == OutcomeTaskStatusAwaitingHumanDecision || next == OutcomeTaskStatusFailed
	case OutcomeTaskStatusAwaitingHumanDecision:
		return next == OutcomeTaskStatusWorking || next == OutcomeTaskStatusOnHold || next == OutcomeTaskStatusFailed
	case OutcomeTaskStatusInReview:
		return next == OutcomeTaskStatusReadyToMerge || next == OutcomeTaskStatusRework || next == OutcomeTaskStatusFailed
	case OutcomeTaskStatusReadyToMerge:
		return next == OutcomeTaskStatusAccepted || next == OutcomeTaskStatusRework
	case OutcomeTaskStatusRework:
		return next == OutcomeTaskStatusWorking || next == OutcomeTaskStatusOnHold || next == OutcomeTaskStatusFailed
	}
	return false
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
