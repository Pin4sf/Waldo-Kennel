package daemon

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/controllers"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	intakevc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/intake"
)

// The brief and the route must name the same header. They are separate
// constants so the spawn path does not import the HTTP controllers, which
// means nothing but this test stops them drifting apart — and a drifted header
// would make every agent proposal unroutable.
func TestIntakeCallbackHeaderMatchesTheRoute(t *testing.T) {
	if intakeCallbackHeader != controllers.IntakeCallbackTokenHeader {
		t.Fatalf("brief header %q != route header %q; an agent would post an unroutable token",
			intakeCallbackHeader, controllers.IntakeCallbackTokenHeader)
	}
}

type stubProjects struct {
	record domain.ProjectRecord
	found  bool
	err    error
}

func (s stubProjects) GetProject(context.Context, string) (domain.ProjectRecord, bool, error) {
	return s.record, s.found, s.err
}

type stubSessions struct {
	spawned  ports.SpawnConfig
	spawnErr error
	calls    int
}

func (s *stubSessions) Spawn(_ context.Context, cfg ports.SpawnConfig) (domain.Session, int, int, error) {
	s.calls++
	s.spawned = cfg
	if s.spawnErr != nil {
		return domain.Session{}, 0, 0, s.spawnErr
	}
	return domain.Session{SessionRecord: domain.SessionRecord{ID: "sess-analyst"}}, 0, 0, nil
}
func (s *stubSessions) Kill(context.Context, domain.SessionID) (bool, error) { return false, nil }
func (s *stubSessions) Get(context.Context, domain.SessionID) (domain.Session, error) {
	return domain.Session{}, nil
}

// analysisInput carries a Defer that records whether a durable ask was opened.
func analysisInput(opened *bool) ports.IntakeAnalysisInput {
	return ports.IntakeAnalysisInput{
		Session: domain.IntakeSession{
			ID: "intake-1", ProjectID: "proj-1",
			Statement: "New contributors can run the app locally without asking anyone",
		},
		Defer: func(context.Context) (ports.IntakeCallback, error) {
			*opened = true
			return ports.IntakeCallback{RequestID: "ireq-1", Token: "tok-abc"}, nil
		},
	}
}

// Intake is the entry point to the product, so a project with no analyzer role
// resolved must still reach a proposal. Failing closed here — which is right
// for decomposition — would mean nobody without an authorized agent could
// create an Outcome at all.
func TestNoAnalystDegradesToTheOfflineProposalWithoutOpeningAnAsk(t *testing.T) {
	opened := false
	analyzer := agentIntakeAnalyzer{
		sessions: &stubSessions{},
		// No agentPreferences, so no analyzer role resolves.
		projects: stubProjects{record: domain.ProjectRecord{}, found: true},
		agents:   nil,
		offline:  intakevc.NewRuleBasedAnalyzer(),
	}
	ticket, err := analyzer.Analyze(context.Background(), analysisInput(&opened))
	if err != nil {
		t.Fatalf("Analyze() error = %v, want a degraded proposal", err)
	}
	if ticket.Inline == nil || ticket.Inline.Proposal == nil {
		t.Fatalf("degradation did not produce a proposal: %+v", ticket)
	}
	if opened {
		t.Fatal("a durable request was opened for an ask that never happened")
	}
	if ticket.SessionID != "" {
		t.Fatalf("degradation spawned a session: %q", ticket.SessionID)
	}
}

func TestUnwiredAnalyzerStillProduces(t *testing.T) {
	opened := false
	analyzer := agentIntakeAnalyzer{offline: intakevc.NewRuleBasedAnalyzer()}
	ticket, err := analyzer.Analyze(context.Background(), analysisInput(&opened))
	if err != nil || ticket.Inline == nil {
		t.Fatalf("unwired analyzer = %+v err=%v, want the offline floor", ticket, err)
	}
	if opened {
		t.Fatal("an unwired analyzer opened a durable request")
	}
}

// The floor must never be handed a way to open a durable ask: a request minted
// during a degraded analysis would be one nothing could ever answer.
func TestDegradationDeniesTheOfflineFloorACallback(t *testing.T) {
	analyzer := agentIntakeAnalyzer{offline: ports.IntakeAnalyzer(deferHungryAnalyzer{})}
	_, err := analyzer.Analyze(context.Background(), analysisInput(new(bool)))
	if err == nil {
		t.Fatal("the offline floor was allowed to open a callback")
	}
}

type deferHungryAnalyzer struct{}

func (deferHungryAnalyzer) Analyze(ctx context.Context, in ports.IntakeAnalysisInput) (ports.IntakeAnalysisTicket, error) {
	if _, err := in.Defer(ctx); err != nil {
		return ports.IntakeAnalysisTicket{}, err
	}
	return ports.IntakeAnalysisTicket{SessionID: "should-not-happen"}, nil
}

// A conflict opening the ask — most often a second agent already working on
// this intake — is a real refusal and must not be papered over by quietly
// producing an offline proposal instead.
func TestAConflictOpeningTheAskIsReportedNotDegraded(t *testing.T) {
	analyzer := readyAnalyzer(&stubSessions{})
	input := analysisInput(new(bool))
	input.Defer = func(context.Context) (ports.IntakeCallback, error) {
		return ports.IntakeCallback{}, errors.New("an agent is already proposing for this intake")
	}
	ticket, err := analyzer.Analyze(context.Background(), input)
	if err == nil {
		t.Fatalf("a conflict was swallowed into %+v", ticket)
	}
	if ticket.Inline != nil {
		t.Fatal("a conflict produced an offline proposal instead of being reported")
	}
}

func TestSpawnFailureIsReportedRatherThanSilentlyDegraded(t *testing.T) {
	sessions := &stubSessions{spawnErr: errors.New("tmux is not installed")}
	opened := false
	ticket, err := readyAnalyzer(sessions).Analyze(context.Background(), analysisInput(&opened))
	if err == nil || !strings.Contains(err.Error(), "tmux is not installed") {
		t.Fatalf("spawn failure = %+v err=%v, want the adapter's own words", ticket, err)
	}
	if !opened {
		t.Fatal("the ask was never opened, so the spawn had nowhere to answer")
	}
}

func TestAskedAnalystSpawnsOneBoundedWorkerCarryingTheBrief(t *testing.T) {
	sessions := &stubSessions{}
	opened := false
	ticket, err := readyAnalyzer(sessions).Analyze(context.Background(), analysisInput(&opened))
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if !opened {
		t.Fatal("no durable ask was opened for a real spawn")
	}
	if ticket.Inline != nil {
		t.Fatal("a deferred analysis also answered inline")
	}
	if ticket.SessionID != "sess-analyst" || ticket.Harness != "codex" {
		t.Fatalf("ticket did not name the answering session and harness: %+v", ticket)
	}
	if sessions.calls != 1 {
		t.Fatalf("spawned %d sessions, want exactly one", sessions.calls)
	}
	// A worker, not an orchestrator: this is a bounded single-task session and
	// must not collide with the project's persistent coordinator.
	if sessions.spawned.Kind != domain.KindWorker {
		t.Fatalf("spawned kind = %q, want worker", sessions.spawned.Kind)
	}
	if !strings.HasPrefix(sessions.spawned.DisplayName, "Propose Contract:") {
		t.Fatalf("display name = %q", sessions.spawned.DisplayName)
	}
}

func readyAnalyzer(sessions *stubSessions) agentIntakeAnalyzer {
	return agentIntakeAnalyzer{
		sessions: sessions,
		projects: stubProjects{found: true, record: domain.ProjectRecord{
			Config: domain.ProjectConfig{AgentPreferences: domain.ProjectAgentPreferences{Analyzer: "codex"}},
		}},
		agents:       readyAgents{},
		offline:      intakevc.NewRuleBasedAnalyzer(),
		callbackBase: "http://127.0.0.1:3031",
	}
}

// readyAgents returns an adapter with no profile gate, which
// ProfileReadinessForSpawn reports as ready — spawn stays the authoritative
// validation point.
type readyAgents struct{}

func (readyAgents) Agent(domain.AgentHarness) (ports.Agent, bool) { return gatelessAgent{}, true }

type gatelessAgent struct{}

func (gatelessAgent) GetConfigSpec(context.Context) (ports.ConfigSpec, error) {
	return ports.ConfigSpec{}, nil
}
func (gatelessAgent) GetLaunchCommand(context.Context, ports.LaunchConfig) ([]string, error) {
	return nil, nil
}
func (gatelessAgent) GetPromptDeliveryStrategy(context.Context, ports.LaunchConfig) (ports.PromptDeliveryStrategy, error) {
	return ports.PromptDeliveryInCommand, nil
}
func (gatelessAgent) GetAgentHooks(context.Context, ports.WorkspaceHookConfig) error { return nil }
func (gatelessAgent) GetRestoreCommand(context.Context, ports.RestoreConfig) ([]string, bool, error) {
	return nil, false, nil
}
func (gatelessAgent) SessionInfo(context.Context, ports.SessionRef) (ports.SessionInfo, bool, error) {
	return ports.SessionInfo{}, false, nil
}

// The brief is the whole interface to the agent. Everything it needs to build
// a proposal the daemon will accept has to be stated in it, because a missing
// rule becomes a refused draft the owner has to read and fix.
func TestIntakeBriefCarriesWhatAProposalNeeds(t *testing.T) {
	brief, err := intakeBrief(
		ports.IntakeAnalysisInput{Session: domain.IntakeSession{Statement: "Contributors can run the app locally"}},
		ports.IntakeCallback{RequestID: "ireq-1", Token: "tok-abc"},
		"http://127.0.0.1:3031/",
	)
	if err != nil {
		t.Fatalf("intakeBrief() error = %v", err)
	}
	for _, want := range []string{
		"Contributors can run the app locally",
		// The endpoint, with the trailing slash on the base not doubled.
		"http://127.0.0.1:3031/api/v1/intake-analysis-requests/ireq-1/proposal",
		intakeCallbackHeader + ": tok-abc",
		// Reading the repo is the entire reason an agent was asked.
		"READ THIS REPOSITORY FIRST",
		// Every field the review screen edits, so a draft is not missing half
		// the contract the owner is about to see.
		"evidenceExpected", "stopConditions", "authorityCeiling", "temporalCondition",
		"constraints", "nonGoals", "facets", "reviewMethod", "desiredState",
		// The admitted facet vocabulary, rather than letting it guess.
		`"software"`, `"operations"`,
		// Least privilege, stated as an instruction and not just a field list.
		"Request the LEAST",
		// The one-question rule and the shape for asking it.
		"clarification", "deferralConsequence",
		// It proposes; it does not decide, and it does not do the work.
		"You are proposing, not deciding",
	} {
		if !strings.Contains(brief, want) {
			t.Errorf("brief is missing %q", want)
		}
	}
	if strings.Contains(brief, "3031//api") {
		t.Error("callback base double-slash was not trimmed")
	}
}

func TestBriefTellsAnAnsweredAgentItsQuestionIsSpent(t *testing.T) {
	brief, err := intakeBrief(ports.IntakeAnalysisInput{
		Session:           domain.IntakeSession{Statement: "Show focus time today"},
		Clarification:     &domain.ClarificationRequest{Question: "Which time boundary?"},
		ClarificationText: "The Mac local calendar day",
	}, ports.IntakeCallback{RequestID: "ireq-1", Token: "tok"}, "http://127.0.0.1:3031")
	if err != nil {
		t.Fatalf("intakeBrief() error = %v", err)
	}
	for _, want := range []string{"Which time boundary?", "The Mac local calendar day", "used the one question"} {
		if !strings.Contains(brief, want) {
			t.Errorf("answered-clarification brief is missing %q", want)
		}
	}
}
