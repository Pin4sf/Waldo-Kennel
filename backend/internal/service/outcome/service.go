// Package outcome owns the Outcome control-plane authority: immutable Contracts,
// Plans, Attempts, proof, and owner decisions. Provider processes remain
// subordinate execution/provenance and cannot create or conclude responsibility.
package outcome

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// Manager is the controller-facing boundary for canonical Outcome work.
type Manager interface {
	Create(ctx context.Context, in CreateInput) (View, error)
	ReviseContract(ctx context.Context, id domain.OutcomeID, in ReviseContractInput) (View, error)
	Get(ctx context.Context, id domain.OutcomeID) (View, error)
	ListByProject(ctx context.Context, projectID domain.ProjectID) ([]View, error)
	ProposeDecomposition(ctx context.Context, parentID domain.OutcomeID, in ProposeDecompositionInput) (DecompositionView, error)
	AuthorizeDecomposition(ctx context.Context, parentID domain.OutcomeID, decompositionID domain.DecompositionRevisionID) (DecompositionView, error)
	LatestDecomposition(ctx context.Context, parentID domain.OutcomeID) (DecompositionView, error)
	WaiveContributionDependency(ctx context.Context, parentID domain.OutcomeID, in WaiveDependencyInput) (DecompositionView, error)
	AskForDecomposition(ctx context.Context, outcomeID domain.OutcomeID, expectedContractRevision int64) (DecompositionRequestView, error)
	SubmitAgentProposal(ctx context.Context, requestID domain.DecompositionRequestID, token string, in ProposeDecompositionInput, raw string) (DecompositionRequestView, error)
	LatestDecompositionRequest(ctx context.Context, outcomeID domain.OutcomeID) (DecompositionRequestView, error)
	CreateContribution(ctx context.Context, parentID domain.OutcomeID, in CreateContributionInput) (View, error)
	Composition(ctx context.Context, id domain.OutcomeID) (CompositionView, error)
	ProposePlan(ctx context.Context, outcomeID domain.OutcomeID, expectedContractRevision int64) (PlanView, error)
	ApprovePlan(ctx context.Context, outcomeID domain.OutcomeID, in ApprovePlanInput) (AuthorizedPlanView, error)
	GetLatestPlan(ctx context.Context, outcomeID domain.OutcomeID) (PlanView, error)
}

// CreateInput contains the initial owner-facing Outcome contract.
type CreateInput struct {
	ProjectID            domain.ProjectID
	Title                string
	Goal                 string
	SuccessCriteria      []string
	Review               string
	Constraints          []string
	NonGoals             []string
	Clarification        string
	EvidenceExpectations []domain.ContractEvidenceExpectation
	AuthorityCeiling     domain.ProposedAuthority
	StopConditions       []string
	TemporalCondition    *string
	Facets               []domain.ContractFacet
	ExecutionPreference  *domain.ExecutionPreference
	RequestKey           string
}

// ReviseContractInput contains a new immutable Contract revision.
type ReviseContractInput struct {
	ExpectedRevision     int64
	Goal                 string
	SuccessCriteria      []string
	Review               string
	Constraints          []string
	NonGoals             []string
	Clarification        string
	EvidenceExpectations []domain.ContractEvidenceExpectation
	AuthorityCeiling     domain.ProposedAuthority
	StopConditions       []string
	TemporalCondition    *string
	Facets               []domain.ContractFacet
	ExecutionPreference  *domain.ExecutionPreference
}

// View is the service projection of an Outcome and its current plan.
type View struct {
	Outcome    domain.Outcome
	Current    domain.ContractRevision
	History    []domain.ContractRevision
	LatestPlan *domain.PlanRevision
}

// Service is the single Outcome control-plane service. Optional seams are
// injected explicitly; absence fails closed rather than inventing provider or
// execution behavior.
type Service struct {
	store ports.OutcomeStore
	proof ports.OutcomeProofStore

	proposer ports.DecompositionProposer
	reaper   ports.AnalystSessionReaper
	clock    func() time.Time

	planIntelligence ports.IntelligenceProvider
	intelligenceRuns ports.IntelligenceRunStore
	routing          ports.ExecutionRoutingInventory

	PolicyLayers [][]string

	spawner    ports.AttemptSessionSpawner
	heartbeats heartbeatSource
	// receipts records what each attempt produced. Optional so a degraded
	// profile still schedules and reports truthfully; when absent, artifact
	// continuity is unavailable rather than silently faked.
	receipts ports.AttemptReceiptStore
	retainer ports.AttemptRetainer
	// checks executes an Attempt's approved deterministic checks under its own
	// frozen policy. Absent means a WorkUnit's checks simply do not run, which
	// leaves its criteria unproved rather than assumed proved.
	checks ports.AttemptCheckRunner

	staleHeartbeat time.Duration
}

// WithCheckRunner wires deterministic check execution into classification.
func (s *Service) WithCheckRunner(runner ports.AttemptCheckRunner) *Service {
	s.checks = runner
	return s
}

// WithStaleHeartbeat configures the stale-attempt threshold.
func (s *Service) WithStaleHeartbeat(d time.Duration) *Service {
	if d > 0 {
		s.staleHeartbeat = d
	}
	return s
}

// New constructs the Outcome control-plane service.
func New(store ports.OutcomeStore, clock func() time.Time) *Service {
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	service := &Service{store: store, clock: clock}
	if proof, ok := store.(ports.OutcomeProofStore); ok {
		service.proof = proof
	}
	if runs, ok := store.(ports.IntelligenceRunStore); ok {
		service.intelligenceRuns = runs
	}
	// The SQLite store implements every one of these; the assertions keep the
	// service usable with narrower fakes in tests rather than forcing each one
	// to satisfy the whole surface.
	if receipts, ok := store.(ports.AttemptReceiptStore); ok {
		service.receipts = receipts
	}
	return service
}

// WithPlanning wires non-authoritative planning and deterministic execution
// routing. Neither dependency can authorize an Attempt; approval remains the
// only transition from proposal to execution authority.
func (s *Service) WithPlanning(provider ports.IntelligenceProvider, routing ports.ExecutionRoutingInventory) *Service {
	s.planIntelligence = provider
	s.routing = routing
	return s
}

// WithExecution attaches the exact-bound execution seam to this same Outcome
// service instance. There is no alternate execution service authority.
func (s *Service) WithExecution(spawner ports.AttemptSessionSpawner, heartbeats heartbeatSource) *Service {
	s.spawner = spawner
	s.heartbeats = heartbeats
	s.staleHeartbeat = domain.DefaultStaleHeartbeatWindow
	return s
}

// WithAttemptRetainer attaches the restart-safe workspace capture used before
// an ended Attempt can satisfy proof. Without it, the service fails closed and
// leaves the Attempt reconciled rather than classifying a manifest-only result.
func (s *Service) WithAttemptRetainer(retainer ports.AttemptRetainer) *Service {
	s.retainer = retainer
	return s
}

// WithAnalystSessionReaper attaches the compatibility analyst reaper.
func (s *Service) WithAnalystSessionReaper(reaper ports.AnalystSessionReaper) *Service {
	s.reaper = reaper
	return s
}

// WithDecompositionProposer attaches non-authoritative decomposition proposal.
func (s *Service) WithDecompositionProposer(proposer ports.DecompositionProposer) *Service {
	s.proposer = proposer
	return s
}

// WithProofStore attaches the durable proof store.
func (s *Service) WithProofStore(proof ports.OutcomeProofStore) *Service {
	s.proof = proof
	return s
}

var _ Manager = (*Service)(nil)

// Create persists an Outcome with its initial Contract revision.
func (s *Service) Create(ctx context.Context, in CreateInput) (View, error) {
	if strings.TrimSpace(string(in.ProjectID)) == "" {
		return View{}, apierr.Invalid("PROJECT_REQUIRED", "Choose the project this Outcome belongs to", nil)
	}
	if strings.TrimSpace(in.RequestKey) == "" {
		return View{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this create", nil)
	}
	content := normalizeContractContent(in.RequestKey, in.Title, in.Goal, in.SuccessCriteria, in.Review, in.Constraints, in.NonGoals, in.Clarification)
	if err := validateTitle(content.title); err != nil {
		return View{}, err
	}
	if err := validateContractCore(content); err != nil {
		return View{}, err
	}

	if existing, ok, err := s.store.FindOutcomeByIdempotencyKey(ctx, content.requestKey); err != nil {
		return View{}, err
	} else if ok {
		return s.Get(ctx, existing.ID)
	}

	space, err := s.store.EnsureWorkResponsibilitySpace(ctx, in.ProjectID)
	if err != nil {
		return View{}, mapStoreSpaceError(err)
	}
	now := s.clock()
	outcomeRecord := domain.Outcome{
		ID:      domain.OutcomeID("out-" + uuid.NewString()),
		SpaceID: space.ID,
		Title:   content.title,
	}
	first := domain.ContractRevision{
		ID:                   domain.ContractRevisionID("cr-" + uuid.NewString()),
		OutcomeID:            outcomeRecord.ID,
		Goal:                 content.goal,
		SuccessCriteria:      content.criteria,
		Review:               content.review,
		Constraints:          content.constraints,
		NonGoals:             content.nonGoals,
		Clarification:        content.clarification,
		EvidenceExpectations: append([]domain.ContractEvidenceExpectation(nil), in.EvidenceExpectations...),
		AuthorityCeiling:     in.AuthorityCeiling,
		StopConditions:       append([]string(nil), in.StopConditions...),
		TemporalCondition:    in.TemporalCondition,
		Facets:               append([]domain.ContractFacet(nil), in.Facets...),
		ExecutionPreference:  in.ExecutionPreference,
		CreatedAt:            now,
	}
	first.Criteria = stableCriteria(first.ID, first.SuccessCriteria)
	if err := s.store.CreateOutcomeWithContract(ctx, outcomeRecord, first, content.requestKey); err != nil {
		if existing, ok, findErr := s.store.FindOutcomeByIdempotencyKey(ctx, content.requestKey); findErr == nil && ok {
			return s.Get(ctx, existing.ID)
		}
		return View{}, err
	}
	return s.Get(ctx, outcomeRecord.ID)
}

// ListByProject returns Outcome projections for a project.
func (s *Service) ListByProject(ctx context.Context, projectID domain.ProjectID) ([]View, error) {
	if strings.TrimSpace(string(projectID)) == "" {
		return nil, apierr.Invalid("PROJECT_REQUIRED", "Choose the project whose Outcomes should be listed", nil)
	}
	records, err := s.store.ListOutcomesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	views := make([]View, 0, len(records))
	for _, record := range records {
		view, err := s.Get(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		latestPlan, found, err := s.store.GetLatestPlanRevision(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		if found {
			view.LatestPlan = &latestPlan
		}
		views = append(views, view)
	}
	return views, nil
}

// ReviseContract appends an owner-authored Contract revision.
func (s *Service) ReviseContract(ctx context.Context, id domain.OutcomeID, in ReviseContractInput) (View, error) {
	if in.ExpectedRevision < 1 {
		return View{}, apierr.Invalid("EXPECTED_REVISION_REQUIRED", "State which contract revision this edit supersedes", nil)
	}
	content := normalizeContractContent("", "", in.Goal, in.SuccessCriteria, in.Review, in.Constraints, in.NonGoals, in.Clarification)
	if err := validateContractCore(content); err != nil {
		return View{}, err
	}

	if _, ok, err := s.store.GetOutcome(ctx, id); err != nil {
		return View{}, err
	} else if !ok {
		return View{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}

	next := domain.ContractRevision{
		ID:                   domain.ContractRevisionID("cr-" + uuid.NewString()),
		OutcomeID:            id,
		Goal:                 content.goal,
		SuccessCriteria:      content.criteria,
		Review:               content.review,
		Constraints:          content.constraints,
		NonGoals:             content.nonGoals,
		Clarification:        content.clarification,
		EvidenceExpectations: append([]domain.ContractEvidenceExpectation(nil), in.EvidenceExpectations...),
		AuthorityCeiling:     in.AuthorityCeiling,
		StopConditions:       append([]string(nil), in.StopConditions...),
		TemporalCondition:    in.TemporalCondition,
		Facets:               append([]domain.ContractFacet(nil), in.Facets...),
		ExecutionPreference:  in.ExecutionPreference,
		CreatedAt:            s.clock(),
	}
	next.Criteria = stableCriteria(next.ID, next.SuccessCriteria)
	number, err := s.store.AppendContractRevision(ctx, id, in.ExpectedRevision, next)
	if err != nil {
		var conflict *ports.OutcomeConflictError
		if errors.As(err, &conflict) {
			return View{}, apierr.New(apierr.KindConflict, "OUTCOME_CONTRACT_CONFLICT",
				fmt.Sprintf("Contract moved to revision %s; reload and retry against it", strconv.FormatInt(conflict.CurrentRevisionNum, 10)),
				map[string]any{
					"outcomeId":        string(id),
					"expectedRevision": conflict.ExpectedRevisionNum,
					"currentRevision":  conflict.CurrentRevisionNum,
				})
		}
		return View{}, err
	}
	_ = number
	return s.Get(ctx, id)
}

func stableCriteria(revisionID domain.ContractRevisionID, texts []string) []domain.ContractCriterion {
	criteria := make([]domain.ContractCriterion, 0, len(texts))
	for i, text := range texts {
		criteria = append(criteria, domain.ContractCriterion{
			ID:                 domain.CriterionID("crit-" + uuid.NewString()),
			ContractRevisionID: revisionID,
			Position:           int64(i + 1),
			Text:               text,
		})
	}
	return criteria
}

// Get returns the current Outcome projection.
func (s *Service) Get(ctx context.Context, id domain.OutcomeID) (View, error) {
	record, ok, err := s.store.GetOutcome(ctx, id)
	if err != nil {
		return View{}, err
	}
	if !ok {
		return View{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	history, err := s.store.ListContractRevisions(ctx, id)
	if err != nil {
		return View{}, err
	}
	for _, rev := range history {
		if rev.Number == record.CurrentRevisionNumber {
			return View{Outcome: record, Current: rev, History: history}, nil
		}
	}
	return View{}, fmt.Errorf("outcome %s points at missing revision %d", id, record.CurrentRevisionNumber)
}

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

func mapStoreSpaceError(err error) error {
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "foreign key") || strings.Contains(lower, "no such table") {
		return apierr.NotFound("PROJECT_NOT_FOUND", "Register that project before recording Outcomes")
	}
	return err
}
