// Package outcome owns the canonical Understand contract (#21): the daemon-side
// authority for ResponsibilitySpaces, Outcomes, and their immutable
// ContractRevisions. It deliberately knows nothing about plans, attempts,
// evidence, or acceptance — those belong to later Work slices — and nothing
// about providers: no provider message, session state, check, or commit can
// create, revise, or conclude an Outcome.
package outcome

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/google/uuid"
)

// Manager is the controller-facing boundary for the Outcome contract.
type Manager interface {
	// Create records a new Outcome with ContractRevision 1 in the project's
	// Work responsibility space. Replaying a delivered RequestKey resolves to
	// the original Outcome without writing anything.
	Create(ctx context.Context, in CreateInput) (OutcomeView, error)

	// ReviseContract appends ContractRevision n+1 and moves the
	// current-revision pointer from ExpectedRevision to it. A stale pointer is
	// reported as *ports.OutcomeConflictError; prior revisions stay untouched.
	ReviseContract(ctx context.Context, id domain.OutcomeID, in ReviseContractInput) (OutcomeView, error)

	// Get reads the Outcome view: canonical facts plus full revision history.
	Get(ctx context.Context, id domain.OutcomeID) (OutcomeView, error)
}

// CreateInput carries one user-authored Understand statement. RequestKey is the
// client's idempotency key for exactly-once creation.
type CreateInput struct {
	ProjectID       domain.ProjectID
	Title           string
	Goal            string
	SuccessCriteria []string
	Review          string
	Constraints     []string
	NonGoals        []string
	Clarification   string
	RequestKey      string
}

// ReviseContractInput supersedes the current contract. ExpectedRevision must
// name the current revision; anything else conflicts.
type ReviseContractInput struct {
	ExpectedRevision int64
	Goal             string
	SuccessCriteria  []string
	Review           string
	Constraints      []string
	NonGoals         []string
	Clarification    string
}

// OutcomeView is the read model over one Outcome's durable facts. Stage labels
// are derived by callers; nothing here persists presentation.
type OutcomeView struct {
	Outcome domain.Outcome
	Current domain.ContractRevision
	History []domain.ContractRevision
}

// Service implements Manager over ports.OutcomeStore.
type Service struct {
	store ports.OutcomeStore
	clock func() time.Time
}

// New builds the service. clock may be nil for wall-clock time.
func New(store ports.OutcomeStore, clock func() time.Time) *Service {
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, clock: clock}
}

var _ Manager = (*Service)(nil)

func (s *Service) Create(ctx context.Context, in CreateInput) (OutcomeView, error) {
	if strings.TrimSpace(string(in.ProjectID)) == "" {
		return OutcomeView{}, apierr.Invalid("PROJECT_REQUIRED", "Choose the project this Outcome belongs to", nil)
	}
	if strings.TrimSpace(in.RequestKey) == "" {
		return OutcomeView{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this create", nil)
	}
	content := normalizeContractContent(in.RequestKey, in.Title, in.Goal, in.SuccessCriteria, in.Review, in.Constraints, in.NonGoals, in.Clarification)
	if err := validateTitle(content.title); err != nil {
		return OutcomeView{}, err
	}
	if err := validateContractCore(content); err != nil {
		return OutcomeView{}, err
	}

	// Replay first: a delivered create never writes twice.
	if existing, ok, err := s.store.FindOutcomeByIdempotencyKey(ctx, content.requestKey); err != nil {
		return OutcomeView{}, err
	} else if ok {
		return s.Get(ctx, existing.ID)
	}

	space, err := s.store.EnsureWorkResponsibilitySpace(ctx, in.ProjectID)
	if err != nil {
		return OutcomeView{}, mapStoreSpaceError(err)
	}
	now := s.clock()
	outcomeRecord := domain.Outcome{
		ID:      domain.OutcomeID("out-" + uuid.NewString()),
		SpaceID: space.ID,
		Title:   content.title,
	}
	first := domain.ContractRevision{
		ID:              domain.ContractRevisionID("cr-" + uuid.NewString()),
		OutcomeID:       outcomeRecord.ID,
		Goal:            content.goal,
		SuccessCriteria: content.criteria,
		Review:          content.review,
		Constraints:     content.constraints,
		NonGoals:        content.nonGoals,
		Clarification:   content.clarification,
		CreatedAt:       now,
	}
	if err := s.store.CreateOutcomeWithContract(ctx, outcomeRecord, first, content.requestKey); err != nil {
		// Either a genuine failure or a lost replay race against an identical
		// request; resolve through the key so both paths serve the winner.
		if existing, ok, findErr := s.store.FindOutcomeByIdempotencyKey(ctx, content.requestKey); findErr == nil && ok {
			return s.Get(ctx, existing.ID)
		}
		return OutcomeView{}, err
	}
	return s.Get(ctx, outcomeRecord.ID)
}

func (s *Service) ReviseContract(ctx context.Context, id domain.OutcomeID, in ReviseContractInput) (OutcomeView, error) {
	if in.ExpectedRevision < 1 {
		return OutcomeView{}, apierr.Invalid("EXPECTED_REVISION_REQUIRED", "State which contract revision this edit supersedes", nil)
	}
	content := normalizeContractContent("", "", in.Goal, in.SuccessCriteria, in.Review, in.Constraints, in.NonGoals, in.Clarification)
	if err := validateContractCore(content); err != nil {
		return OutcomeView{}, err
	}

	next := domain.ContractRevision{
		OutcomeID:       id,
		Goal:            content.goal,
		SuccessCriteria: content.criteria,
		Review:          content.review,
		Constraints:     content.constraints,
		NonGoals:        content.nonGoals,
		Clarification:   content.clarification,
		CreatedAt:       s.clock(),
	}
	if _, err := s.store.AppendContractRevision(ctx, id, in.ExpectedRevision, next); err != nil {
		return OutcomeView{}, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id domain.OutcomeID) (OutcomeView, error) {
	record, ok, err := s.store.GetOutcome(ctx, id)
	if err != nil {
		return OutcomeView{}, err
	}
	if !ok {
		return OutcomeView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	history, err := s.store.ListContractRevisions(ctx, id)
	if err != nil {
		return OutcomeView{}, err
	}
	for _, rev := range history {
		if rev.Number == record.CurrentRevisionNumber {
			return OutcomeView{Outcome: record, Current: rev, History: history}, nil
		}
	}
	return OutcomeView{}, fmt.Errorf("outcome %s points at missing revision %d", id, record.CurrentRevisionNumber)
}

// contractContent is trimmed, validated Understand input.
type contractContent struct {
	requestKey    string
	title         string
	goal          string
	criteria      []string
	review        string
	constraints   []string
	nonGoals      []string
	clarification string
}

func normalizeContractContent(requestKey, title, goal string, criteria []string, review string, constraints, nonGoals []string, clarification string) contractContent {
	trimList := func(in []string) []string {
		out := make([]string, 0, len(in))
		for _, v := range in {
			v = strings.TrimSpace(v)
			if v != "" {
				out = append(out, v)
			}
		}
		return out
	}
	return contractContent{
		requestKey:    strings.TrimSpace(requestKey),
		title:         strings.TrimSpace(title),
		goal:          strings.TrimSpace(goal),
		criteria:      trimList(criteria),
		review:        strings.TrimSpace(review),
		constraints:   trimList(constraints),
		nonGoals:      trimList(nonGoals),
		clarification: strings.TrimSpace(clarification),
	}
}

func validateTitle(title string) error {
	const maxTitleLen = 200
	switch {
	case title == "":
		return apierr.Invalid("OUTCOME_TITLE_REQUIRED", "Give the Outcome a short title", nil)
	case len(title) > maxTitleLen:
		return apierr.Invalid("OUTCOME_TITLE_TOO_LONG", fmt.Sprintf("Keep the title under %d characters", maxTitleLen), nil)
	}
	return nil
}

// validateContractCore applies the Goal/Success/Review requirements every
// immutable revision must satisfy.
func validateContractCore(c contractContent) error {
	switch {
	case c.goal == "":
		return apierr.Invalid("OUTCOME_GOAL_REQUIRED", "State the goal this Outcome pursues", nil)
	case len(c.criteria) == 0:
		return apierr.Invalid("OUTCOME_CRITERIA_REQUIRED", "Name at least one success criterion", nil)
	case c.review == "":
		return apierr.Invalid("OUTCOME_REVIEW_REQUIRED", "Describe how the result will be reviewed", nil)
	}
	return nil
}

// mapStoreSpaceError translates foreign-key failures on unknown projects into
// the shared not-found envelope.
func mapStoreSpaceError(err error) error {
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "foreign key") || strings.Contains(lower, "no such table") {
		return apierr.NotFound("PROJECT_NOT_FOUND", "Register that project before recording Outcomes")
	}
	return err
}
