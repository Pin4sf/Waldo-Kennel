package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	sessionmanager "github.com/aoagents/agent-orchestrator/backend/internal/session_manager"
)

// agentIntakeAnalyzer starts a bounded session that proposes the Contract for
// one intake, and hands it the callback it must answer on.
//
// It is the same shape as agentDecompositionProposer — analyzer role, worker
// kind, fail-closed readiness, explicit JSON brief, loopback callback — with
// one deliberate difference, and it is the important one:
//
// DECOMPOSITION FAILS CLOSED. INTAKE DOES NOT.
//
// Decomposing is something an owner explicitly asked for and can be told no
// about. Intake is the entry point to the entire product: a person with no
// authorized agent must still be able to create an Outcome. So every reason
// this analyzer cannot ask an agent — no analyzer role, an unready harness, a
// spawn that fails — degrades to the deterministic offline proposal instead of
// refusing. The person still gets a Contract to review; it is simply one that
// nothing analyzed, and the renderer says so.
//
// It adds no provider knowledge. Readiness goes through the same
// ProfileReadinessForSpawn the ordinary spawn path uses, and an unready
// harness never falls back to a DIFFERENT provider — only to the offline
// baseline.
type agentIntakeAnalyzer struct {
	sessions attemptSessionControl
	projects projectConfigSource
	agents   ports.AgentResolver
	// offline is the deterministic floor this degrades to. It is never nil in
	// a wired daemon; a nil one would turn every degradation into a failure,
	// which is the behaviour this type exists to avoid.
	offline ports.IntakeAnalyzer
	// callbackBase is the daemon's own loopback origin, e.g.
	// "http://127.0.0.1:3031". The spawned agent posts its proposal here.
	callbackBase string
}

var _ ports.IntakeAnalyzer = agentIntakeAnalyzer{}

// Analyze resolves the analyzer role and starts one bounded session, or
// degrades to the offline floor when it cannot.
func (a agentIntakeAnalyzer) Analyze(ctx context.Context, in ports.IntakeAnalysisInput) (ports.IntakeAnalysisTicket, error) {
	harness, ok := a.resolveAnalyst(ctx, in.Session.ProjectID)
	if !ok {
		return a.degrade(ctx, in)
	}

	// Only now is an agent actually going to be asked, so only now is a
	// durable request worth writing. Opening it earlier would record an ask
	// that never happened every time this degrades.
	callback, err := in.Defer(ctx)
	if err != nil {
		// A refusal here is a real conflict — most often another agent is
		// already working on this intake — and must NOT be papered over by
		// quietly producing an offline proposal instead.
		return ports.IntakeAnalysisTicket{}, err
	}

	prompt, err := intakeBrief(in, callback, a.callbackBase)
	if err != nil {
		return ports.IntakeAnalysisTicket{}, err
	}
	session, _, _, err := a.sessions.Spawn(ctx, ports.SpawnConfig{
		ProjectID:   in.Session.ProjectID,
		Kind:        domain.KindWorker,
		Harness:     harness,
		Prompt:      prompt,
		DisplayName: "Propose Contract: " + excerpt(in.Session.Statement, 60),
	})
	if err != nil {
		// The spawn failed, so nothing will ever answer this request. Returning
		// the error leaves the intake in a retryable failure with the open ask
		// beside it; the request's own expiry closes it, and the owner can take
		// the offline proposal immediately.
		return ports.IntakeAnalysisTicket{}, fmt.Errorf("could not start %s to propose a Contract: %w", harness, err)
	}
	return ports.IntakeAnalysisTicket{
		SessionID: string(session.ID),
		Harness:   harness,
		Detail:    string(harness) + " is reading the project to propose a Contract",
	}, nil
}

// resolveAnalyst names the harness admitted to analyze for this project, or
// reports that none is ready. Every "no" here is a reason to degrade, never a
// reason to fail: see the type comment.
func (a agentIntakeAnalyzer) resolveAnalyst(ctx context.Context, projectID domain.ProjectID) (domain.AgentHarness, bool) {
	if a.sessions == nil || a.projects == nil || a.agents == nil {
		return "", false
	}
	record, found, err := a.projects.GetProject(ctx, string(projectID))
	if err != nil || !found {
		return "", false
	}
	// The analyzer role decides which harness may propose. A harness not
	// admitted to analyze is not admitted here either, and there is no
	// fallback to a different provider.
	harness := domain.ResolveMissionRoles(record.Config.AgentPreferences).Analyzer.Harness
	if harness == "" {
		return "", false
	}
	readiness, err := sessionmanager.ProfileReadinessForSpawn(ctx, a.agents, record.Config, domain.KindWorker, harness, ports.AgentConfig{})
	if err != nil || !readiness.Ready {
		return "", false
	}
	return harness, true
}

// degrade produces the deterministic proposal. The intake completes normally;
// what the owner loses is analysis, not the ability to proceed.
func (a agentIntakeAnalyzer) degrade(ctx context.Context, in ports.IntakeAnalysisInput) (ports.IntakeAnalysisTicket, error) {
	if a.offline == nil {
		return ports.IntakeAnalysisTicket{}, fmt.Errorf("no analyzer is available for this project and no offline baseline is wired")
	}
	// The floor never defers, so it must not be handed a way to open a durable
	// request. Passing the real Defer through would let a future floor mint an
	// ask nothing would answer.
	offlineInput := in
	offlineInput.Defer = func(context.Context) (ports.IntakeCallback, error) {
		return ports.IntakeCallback{}, fmt.Errorf("the offline baseline does not defer")
	}
	return a.offline.Analyze(ctx, offlineInput)
}

// briefFacetKinds is the facet vocabulary the daemon admits, restated for the
// agent. A kind outside this set is refused at validation, so offering the
// list beats letting the agent guess.
var briefFacetKinds = []string{"software", "research", "design", "documentation", "investigation", "evaluation", "operations"}

// intakeBrief renders the prompt handed to the proposing session.
//
// It is deliberately explicit rather than conversational: the agent's only job
// is to read the project and POST one JSON document, and everything needed to
// build a valid one — every field, the facet vocabulary, the authority flags,
// the one-question rule, the exact endpoint — is stated rather than implied.
func intakeBrief(in ports.IntakeAnalysisInput, callback ports.IntakeCallback, callbackBase string) (string, error) {
	facets, err := json.Marshal(briefFacetKinds)
	if err != nil {
		return "", err
	}

	var brief strings.Builder
	fmt.Fprintf(&brief, `You are proposing the Contract for one durable Outcome, before it exists.

WHAT THE OWNER ASKED FOR, VERBATIM:
%s
`, in.Session.Statement)

	if answer := strings.TrimSpace(in.ClarificationText); answer != "" {
		fmt.Fprintf(&brief, `
THEY HAVE ALREADY ANSWERED ONE CLARIFYING QUESTION:
  question: %s
  answer:   %s
You have used the one question this intake allows. Propose now.
`, clarificationQuestion(in.Clarification), answer)
	}

	fmt.Fprintf(&brief, `
READ THIS REPOSITORY FIRST. That is the entire reason you were asked instead of
a template. Success criteria that merely restate the sentence above are worse
than useless — the offline baseline already produces those. Criteria must be
specific to THIS project: name the real files, commands, endpoints, or
behaviours that would show the result is true, and make each one something a
person could check and disagree about.

THE CONTRACT YOU ARE PROPOSING:
  title          a short name for the Outcome.
  desiredState   what is true when this is done, as a state of the world and
                 not a list of steps.
  criteria       each with "text" (one thing that must be true) and
                 "evidenceExpected" (what would actually show it). At least
                 one criterion, and every criterion needs at least one piece
                 of expected evidence.
  reviewMethod   how the owner will check this, concretely.
  constraints    limits on HOW the work is done. Omit if there are none.
  nonGoals       what is explicitly out of scope. Omit if there are none.
  stopConditions at least one. Where the agent must stop and ask rather than
                 proceed.
  facets         at least one, each {"kind": ..., "summary": ...} where kind is
                 one of %s.
  temporalCondition  a time boundary, or null when there is none.
  authorityCeiling   what the agent doing this work may do. Request the LEAST
                 that could plausibly work. These are listed in escalating
                 order of consequence, and the last four are rarely justified
                 for one Outcome:
                   readWorkspace   read the project
                   writeWorkspace  change files in it
                   executeLocal    run commands locally
                   useNetwork      reach the network
                   commitLocal     commit
                   createPr        open a pull request
                   deploy          deploy
                   externalEffect  anything else the owner cannot take back

RULES the daemon enforces. A proposal breaking any of these is refused, and
the refusal is shown to the owner with your draft attached:
  - title, desiredState and reviewMethod must be non-empty.
  - At least one criterion, each with non-empty text and at least one piece of
    expected evidence.
  - At least one stop condition.
  - Every facet needs a summary and an admitted kind.
  - temporalCondition is either absent/null or non-blank. Never "".

IF ONE THING IS GENUINELY AMBIGUOUS and the answer would change what you
propose, you may ask exactly ONE question instead of proposing — but only if
you have not already been given an answer above. Prefer proposing with a
stated assumption: the owner edits every field of this before confirming it.

When you have decided, POST exactly one of these and then stop.

To propose:

  curl -X POST '%s/api/v1/intake-analysis-requests/%s/proposal' \
    -H 'Content-Type: application/json' \
    -H '%s: %s' \
    -d '{
      "proposal": {
        "title": "...",
        "desiredState": "...",
        "criteria": [{"text": "...", "evidenceExpected": ["..."]}],
        "reviewMethod": "...",
        "constraints": [],
        "nonGoals": [],
        "stopConditions": ["..."],
        "facets": [{"kind": "software", "summary": "..."}],
        "temporalCondition": null,
        "authorityCeiling": {
          "readWorkspace": true, "writeWorkspace": true, "executeLocal": true,
          "useNetwork": false, "commitLocal": false, "createPr": false,
          "deploy": false, "externalEffect": false
        }
      }
    }'

To ask the one question instead:

  curl -X POST '%s/api/v1/intake-analysis-requests/%s/proposal' \
    -H 'Content-Type: application/json' \
    -H '%s: %s' \
    -d '{
      "clarification": {
        "question": "...",
        "reason": "why the answer changes the proposal",
        "recommendation": "what you would assume if not answered",
        "alternatives": ["...", "..."],
        "deferralConsequence": "what happens if they do not answer"
      }
    }'

You are proposing, not deciding. Nothing you send creates any responsibility:
the owner reviews and edits every field before confirming it, and may discard
it entirely. Do not modify any file, run any build, or start any other work —
read the repository, POST once, and stop.
`,
		string(facets),
		strings.TrimRight(callbackBase, "/"), callback.RequestID, intakeCallbackHeader, callback.Token,
		strings.TrimRight(callbackBase, "/"), callback.RequestID, intakeCallbackHeader, callback.Token,
	)
	return brief.String(), nil
}

func clarificationQuestion(clarification *domain.ClarificationRequest) string {
	if clarification == nil {
		return "(not recorded)"
	}
	return clarification.Question
}

// excerpt keeps a spawned session's display name readable when the owner's
// statement is a paragraph.
func excerpt(statement string, limit int) string {
	trimmed := strings.TrimSpace(statement)
	if len(trimmed) <= limit {
		return trimmed
	}
	return strings.TrimSpace(trimmed[:limit]) + "…"
}

// intakeCallbackHeader mirrors controllers.IntakeCallbackTokenHeader. It is
// restated here rather than imported so the daemon wiring does not pull the
// HTTP controller package into the spawn path.
const intakeCallbackHeader = "X-Kennel-Intake-Token" //nolint:gosec // header NAME, not a credential.

// intakeAnalysisSweepInterval is how often a running daemon re-checks durable
// expiry. Well under the request TTL, so an abandoned ask reaches a verdict
// within a fraction of its own lifetime rather than at the next restart.
const intakeAnalysisSweepInterval = 2 * time.Minute
