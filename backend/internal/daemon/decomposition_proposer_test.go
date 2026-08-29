package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/controllers"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// The brief and the route must name the same header. They are separate
// constants so the spawn path does not import the HTTP controllers, which
// means nothing but this test stops them drifting apart — and a drifted header
// would make every agent proposal unroutable.
func TestCallbackHeaderMatchesTheRoute(t *testing.T) {
	if decompositionCallbackHeader != controllers.DecompositionCallbackTokenHeader {
		t.Fatalf("brief header %q != route header %q; an agent would post an unroutable token",
			decompositionCallbackHeader, controllers.DecompositionCallbackTokenHeader)
	}
}

func briefInput() ports.DecompositionProposalInput {
	return ports.DecompositionProposalInput{
		RequestID:    "dreq-1",
		ProjectID:    "mer",
		OutcomeID:    "out-parent",
		OutcomeTitle: "OpenCode is a first-class harness",
		Contract: domain.ContractRevision{
			ID: "cr-1", Goal: "OpenCode is selectable and resumable.", Review: "Separate-session review.",
			Criteria: []domain.ContractCriterion{
				{ID: "crit-a", ContractRevisionID: "cr-1", Position: 1, Text: "Selectable for every role."},
				{ID: "crit-b", ContractRevisionID: "cr-1", Position: 2, Text: "Resumes truthfully."},
			},
		},
		CallbackToken:    "tok-abc",
		MaxContributions: domain.MaxProposedContributions,
		ParentAuthority:  domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true},
	}
}

func TestDecompositionBriefCarriesWhatAProposalNeeds(t *testing.T) {
	brief, err := decompositionBrief(briefInput(), "http://127.0.0.1:3031/")
	if err != nil {
		t.Fatalf("brief: %v", err)
	}

	// Stable criterion identities: without them a proposal cannot bind, and an
	// agent left to invent ids would be refused every time.
	for _, want := range []string{"crit-a", "crit-b", "Selectable for every role."} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief must carry %q", want)
		}
	}
	// The exact endpoint and token, with the trailing slash normalised away.
	if !strings.Contains(brief, "http://127.0.0.1:3031/api/v1/decomposition-requests/dreq-1/proposal") {
		t.Fatalf("brief must carry the exact callback endpoint:\n%s", brief)
	}
	if strings.Contains(brief, "3031//api") {
		t.Fatal("a trailing slash on the origin must not double up in the URL")
	}
	if !strings.Contains(brief, "tok-abc") || !strings.Contains(brief, decompositionCallbackHeader) {
		t.Fatal("brief must carry the callback token and its header")
	}
	// The rules the daemon actually enforces, so a refusal is not a surprise.
	for _, want := range []string{"claimed", "retainedCriteria", "Cycles are refused", "rationale"} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief must state the enforced rule %q", want)
		}
	}
	if !strings.Contains(brief, "at most 12") && !strings.Contains(brief, "At most 12") {
		t.Fatalf("brief must state the contribution cap:\n%s", brief)
	}
	// The authority it may not exceed, as data rather than prose.
	var authority map[string]bool
	start := strings.Index(brief, `{"readWorkspace"`)
	if start < 0 {
		t.Fatalf("brief must carry the authority ceiling as JSON:\n%s", brief)
	}
	end := strings.Index(brief[start:], "}") + start + 1
	if err := json.Unmarshal([]byte(brief[start:end]), &authority); err != nil {
		t.Fatalf("authority ceiling must be valid JSON: %v", err)
	}
	if !authority["readWorkspace"] || authority["deploy"] {
		t.Fatalf("authority ceiling did not survive: %+v", authority)
	}
	// It proposes; it does not decide, and it does not do the work.
	if !strings.Contains(brief, "proposing, not deciding") {
		t.Fatal("the brief must say the agent is proposing rather than deciding")
	}
	if !strings.Contains(brief, "Do not modify any file") {
		t.Fatal("the brief must bound the session to reading and posting")
	}
}

type proposerProject struct {
	record domain.ProjectRecord
	ok     bool
}

func (p proposerProject) GetProject(context.Context, string) (domain.ProjectRecord, bool, error) {
	return p.record, p.ok, nil
}

type recordingSpawn struct {
	cfg ports.SpawnConfig
	err error
}

func (r *recordingSpawn) Spawn(_ context.Context, cfg ports.SpawnConfig) (domain.Session, int, int, error) {
	r.cfg = cfg
	if r.err != nil {
		return domain.Session{}, 0, 0, r.err
	}
	return domain.Session{SessionRecord: domain.SessionRecord{ID: "sess-analyzer"}}, 0, 0, nil
}
func (r *recordingSpawn) Kill(context.Context, domain.SessionID) (bool, error) { return false, nil }
func (r *recordingSpawn) Get(context.Context, domain.SessionID) (domain.Session, error) {
	return domain.Session{}, nil
}

func TestProposeRefusesWhenTheProjectIsUnknown(t *testing.T) {
	proposer := agentDecompositionProposer{
		sessions: &recordingSpawn{}, projects: proposerProject{ok: false}, agents: stubAgentResolver{},
	}
	if _, err := proposer.Propose(context.Background(), briefInput()); err == nil {
		t.Fatal("an unregistered project must not spawn a proposer")
	}
}

func TestProposeRefusesWhenNothingIsWired(t *testing.T) {
	if _, err := (agentDecompositionProposer{}).Propose(context.Background(), briefInput()); err == nil {
		t.Fatal("an unwired proposer must fail closed rather than pretend it started")
	} else if !strings.Contains(err.Error(), "not fully wired") {
		t.Fatalf("refusal should name the wiring gap, got %v", err)
	}
}

func TestProposeSurfacesASpawnFailure(t *testing.T) {
	spawn := &recordingSpawn{err: errors.New("harness exited")}
	proposer := agentDecompositionProposer{
		sessions: spawn,
		projects: proposerProject{ok: true, record: domain.ProjectRecord{ID: "mer"}},
		agents:   stubAgentResolver{},
	}
	if _, err := proposer.Propose(context.Background(), briefInput()); err == nil {
		t.Fatal("a failed spawn must surface so the request can be closed")
	}
}

// stubAgentResolver reports no adapter, so readiness cannot pass. The tests
// above assert the refusals that happen BEFORE readiness is consulted; the
// readiness gate itself is the ordinary spawn path's, exercised there.
type stubAgentResolver struct{}

func (stubAgentResolver) Agent(domain.AgentHarness) (ports.Agent, bool) { return nil, false }
