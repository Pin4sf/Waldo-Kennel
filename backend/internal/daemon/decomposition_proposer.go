package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	sessionmanager "github.com/Pin4sf/Waldo-Kennel/backend/internal/session_manager"
)

// agentDecompositionProposer starts a bounded session that proposes how to
// decompose one Outcome, and hands it the callback it must answer on.
//
// It spawns a WORKER-kind session on the ANALYZER role's harness. The kind and
// the role answer different questions: analyzer is the role whose job is to
// analyze and propose, while worker is the bounded, single-task session shape
// this needs. Spawning an orchestrator instead would collide with the
// project's own persistent coordinator, which this is not.
//
// It adds no provider knowledge. Readiness goes through the same
// ProfileReadinessForSpawn the ordinary spawn path uses, and an unready
// harness fails closed rather than falling back to another provider.
type agentDecompositionProposer struct {
	sessions attemptSessionControl
	projects projectConfigSource
	agents   ports.AgentResolver
	// callbackBase is the daemon's own loopback origin, e.g.
	// "http://127.0.0.1:3031". The spawned agent posts its proposal here.
	callbackBase string
}

var _ ports.DecompositionProposer = agentDecompositionProposer{}

// Propose resolves the analyzer role, fails closed when it is not ready, and
// starts one bounded session carrying the brief.
func (p agentDecompositionProposer) Propose(ctx context.Context, in ports.DecompositionProposalInput) (ports.DecompositionProposalTicket, error) {
	if p.sessions == nil || p.projects == nil || p.agents == nil {
		return ports.DecompositionProposalTicket{}, fmt.Errorf("decomposition proposer is not fully wired")
	}
	record, ok, err := p.projects.GetProject(ctx, string(in.ProjectID))
	if err != nil {
		return ports.DecompositionProposalTicket{}, err
	}
	if !ok {
		return ports.DecompositionProposalTicket{}, fmt.Errorf("project %s is not registered", in.ProjectID)
	}

	// The analyzer role decides which harness may propose. A harness that is
	// not admitted to coordinate is not admitted to decompose either.
	roles := domain.ResolveMissionRoles(record.Config)
	harness := roles.Analyzer.Harness
	if harness == "" {
		return ports.DecompositionProposalTicket{}, fmt.Errorf("no analyzer harness is resolved for project %s", in.ProjectID)
	}
	readiness, err := sessionmanager.ProfileReadinessForSpawn(ctx, p.agents, record.Config, domain.KindWorker, harness, ports.AgentConfig{})
	if err != nil {
		return ports.DecompositionProposalTicket{}, err
	}
	if !readiness.Ready {
		// Fail closed, and say which harness and why. There is no fallback to
		// a different provider anywhere in this path.
		return ports.DecompositionProposalTicket{}, fmt.Errorf("%s cannot propose a decomposition: %s", harness, readiness.Detail)
	}

	prompt, err := decompositionBrief(in, p.callbackBase)
	if err != nil {
		return ports.DecompositionProposalTicket{}, err
	}
	session, _, _, err := p.sessions.Spawn(ctx, ports.SpawnConfig{
		ProjectID:   in.ProjectID,
		Kind:        domain.KindWorker,
		Harness:     harness,
		Prompt:      prompt,
		DisplayName: "Propose decomposition: " + in.OutcomeTitle,
	})
	if err != nil {
		return ports.DecompositionProposalTicket{}, err
	}
	return ports.DecompositionProposalTicket{
		SessionID: string(session.ID),
		Detail:    string(harness) + " is proposing a decomposition",
	}, nil
}

// briefCriterion is how one parent criterion reaches the agent: its STABLE id
// alongside its text. The id is what makes a proposal bindable — an agent that
// had to invent identities could not produce one the daemon would accept.
type briefCriterion struct {
	CriterionID string `json:"criterionId"`
	Text        string `json:"text"`
}

// briefAuthority renders the ceiling in the API's own casing rather than the
// domain type's Go field names. The agent sees this alongside an API endpoint,
// so the two should not disagree about what a field is called. The domain type
// deliberately carries no JSON tags — its wire mapping lives in the DTO layer.
type briefAuthority struct {
	ReadWorkspace  bool `json:"readWorkspace"`
	WriteWorkspace bool `json:"writeWorkspace"`
	ExecuteLocal   bool `json:"executeLocal"`
	UseNetwork     bool `json:"useNetwork"`
	CommitLocal    bool `json:"commitLocal"`
	CreatePR       bool `json:"createPr"`
	Deploy         bool `json:"deploy"`
	ExternalEffect bool `json:"externalEffect"`
}

// decompositionBrief renders the prompt handed to the proposing session.
//
// It is deliberately explicit about the contract rather than conversational:
// the agent's only job is to POST one JSON document, and everything it needs
// to build a valid one — criterion ids, the authority ceiling it may not
// exceed, the cap, the exact endpoint — is stated rather than implied.
func decompositionBrief(in ports.DecompositionProposalInput, callbackBase string) (string, error) {
	criteria := make([]briefCriterion, 0, len(in.Contract.Criteria))
	for _, criterion := range in.Contract.Criteria {
		criteria = append(criteria, briefCriterion{CriterionID: string(criterion.ID), Text: criterion.Text})
	}
	criteriaJSON, err := json.MarshalIndent(criteria, "", "  ")
	if err != nil {
		return "", err
	}
	authorityJSON, err := json.Marshal(briefAuthority{
		ReadWorkspace:  in.ParentAuthority.ReadWorkspace,
		WriteWorkspace: in.ParentAuthority.WriteWorkspace,
		ExecuteLocal:   in.ParentAuthority.ExecuteLocal,
		UseNetwork:     in.ParentAuthority.UseNetwork,
		CommitLocal:    in.ParentAuthority.CommitLocal,
		CreatePR:       in.ParentAuthority.CreatePR,
		Deploy:         in.ParentAuthority.Deploy,
		ExternalEffect: in.ParentAuthority.ExternalEffect,
	})
	if err != nil {
		return "", err
	}

	var brief strings.Builder
	fmt.Fprintf(&brief, `You are proposing how to decompose one durable Outcome into contributing Outcomes.

OUTCOME: %s
DESIRED RESULT: %s
REVIEW METHOD: %s

Its success criteria, with the stable identities you MUST use verbatim:
%s

Read this repository and propose the smallest sufficient set of contributing
Outcomes that would make the desired result true. Each contributing Outcome is
a full responsibility with its own goal, success criteria, and review method —
not a task.

RULES the daemon enforces. A proposal breaking any of these is refused:
  - Every parent criterion above must be claimed by at least one contributing
    Outcome, or listed in retainedCriteria for the owner to prove directly.
    There is no third option.
  - Every contributing Outcome must claim at least one parent criterion. There
    is no "foundation" or "prerequisite" contribution that claims none: if some
    work only enables another contribution, it belongs INSIDE that one, not
    beside it. A contribution proving nothing about the parent could be
    abandoned without changing whether the parent is done, which is what makes
    it a task rather than a responsibility.
  - Use the criterionId values exactly as given. Invented ids are refused.
  - A criterion may not be both claimed and retained.
  - At most %d contributing Outcomes.
  - Declare ordering in "dependencies" when one contribution must finish
    before another starts. Cycles are refused.
  - "rationale" is required: explain the shape in plain language, because the
    owner reviews this before authorizing it.

Every contributing Outcome inherits this authority ceiling and may not exceed
it: %s

When you have decided, POST exactly this shape and then stop:

  curl -X POST '%s/api/v1/decomposition-requests/%s/proposal' \
    -H 'Content-Type: application/json' \
    -H '%s: %s' \
    -d '{
      "rationale": "...",
      "contributors": [
        {
          "ref": "c1",
          "title": "...",
          "goal": "...",
          "successCriteria": ["..."],
          "review": "...",
          "claimedCriteria": ["<criterionId from above>"]
        }
      ],
      "retainedCriteria": [],
      "dependencies": [{"fromRef": "c1", "toRef": "c2"}]
    }'

You are proposing, not deciding. Nothing you send creates any responsibility:
the owner reviews and authorizes it. Do not modify any file, run any build, or
start any other work — read the repository, POST the proposal, and stop.
`,
		in.OutcomeTitle,
		in.Contract.Goal,
		in.Contract.Review,
		string(criteriaJSON),
		in.MaxContributions,
		string(authorityJSON),
		strings.TrimRight(callbackBase, "/"),
		in.RequestID,
		decompositionCallbackHeader,
		in.CallbackToken,
	)
	return brief.String(), nil
}

// decompositionCallbackHeader mirrors controllers.DecompositionCallbackTokenHeader.
// It is restated here rather than imported so the daemon wiring does not pull
// the HTTP controller package into the spawn path.
const decompositionCallbackHeader = "X-Kennel-Decomposition-Token" //nolint:gosec // header NAME, not a credential.
