package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// LLMProviderID names model-backed reasoning in provenance records.
const LLMProviderID domain.IntelligenceProviderID = "waldo-llm"

// LLMProvider is Waldo's reasoning implementation. It proposes Contract and
// Plan material and nothing else: it starts no session, holds no authority,
// and its output is validated by the Go control plane before it can bind
// anything.
type LLMProvider struct {
	client ports.LLMClient
}

var _ ports.IntelligenceProvider = (*LLMProvider)(nil)

// NewLLMProvider wires model-backed intelligence behind the provider port.
func NewLLMProvider(client ports.LLMClient) *LLMProvider {
	return &LLMProvider{client: client}
}

func (*LLMProvider) ID() domain.IntelligenceProviderID { return LLMProviderID }

const contractSystemPrompt = `You are Waldo, the reasoning half of an outcome control plane for software work.

A person has stated something they want to be true. Your job is to turn that into a precise, verifiable Contract, or to ask ONE question if a genuinely material fact is missing.

Rules:
- Derive as much as you safely can. Do not interrogate. Ask a question ONLY when a reasonable person could build two materially different results from the same statement, and the choice changes what "done" means.
- Never ask about anything you could reasonably assume and state as an assumption instead.
- Success criteria must be observable and checkable by someone who did not do the work. "Works well" is not a criterion; "the CLI exits 0 and prints the parsed config" is.
- Each criterion carries the evidence that would prove it.
- The authority ceiling is the MAXIMUM you would ever need, and it must be the least that could do the job. Default to workspace read/write/execute. Only request network, commit, PR, deploy, or external effects when the stated outcome plainly cannot be reached without them.
- Stop conditions name the moments a human must decide before work continues.
- Non-goals matter: name the adjacent work you are deliberately NOT doing.

Return only the structured object.`

const planSystemPrompt = `You are Waldo, planning execution for an approved Contract in an outcome control plane.

Break the Contract into the smallest set of work units that can actually be executed and proved. You are proposing work, not authorizing it: the control plane derives capabilities, routing, stop policy, and verification from what you return.

Rules:
- Prefer few units. One unit is correct when the work is genuinely one step. Never split work just to look thorough.
- Each unit must produce something observable that moves at least one criterion toward proof.
- Use dependsOn only for real ordering constraints. Units with no dependency between them will be allowed to run in parallel, so do not serialize work that is genuinely independent.
- intent classifies the work: "inspect" reads only; "modify" edits files; "execute" runs commands; "modify_and_execute" does both. Choose the LEAST intent that can do the unit's job — it decides how much authority the unit is granted.
- criteriaCovered references the criterion aliases given to you (C1, C2, ...). Every criterion should be covered by at least one unit.
- evidenceIdeas are the artifacts that would prove the unit did its job.
- Record real assumptions and real blockers. An empty list is the honest answer when there are none; never invent them.

Return only the structured object.`

// contractSchema constrains the reply to one clarification or one proposal.
func contractSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"decision"},
		"properties": map[string]any{
			"decision": map[string]any{
				"type":        "string",
				"enum":        []any{"ask", "propose"},
				"description": "ask only when a material fact is missing",
			},
			"question": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"question", "reason", "recommendation", "alternatives", "deferralConsequence"},
				"properties": map[string]any{
					"question":            map[string]any{"type": "string"},
					"reason":              map[string]any{"type": "string", "description": "why this changes what done means"},
					"recommendation":      map[string]any{"type": "string", "description": "what you would choose"},
					"alternatives":        stringArray("the concrete options"),
					"deferralConsequence": map[string]any{"type": "string", "description": "what happens if unanswered"},
				},
			},
			"proposal": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"title", "desiredState", "criteria", "reviewMethod", "authorityCeiling", "stopConditions", "facet"},
				"properties": map[string]any{
					"title":        map[string]any{"type": "string", "description": "short result-shaped name, not a task name"},
					"desiredState": map[string]any{"type": "string", "description": "what will be true when this is done"},
					"criteria": map[string]any{
						"type":     "array",
						"minItems": 1,
						"maxItems": 12,
						"items": map[string]any{
							"type":                 "object",
							"additionalProperties": false,
							"required":             []any{"text", "evidenceExpected"},
							"properties": map[string]any{
								"text":             map[string]any{"type": "string"},
								"evidenceExpected": stringArray("artifacts that would prove this criterion"),
							},
						},
					},
					"reviewMethod":     map[string]any{"type": "string"},
					"constraints":      stringArray("limits the work must respect"),
					"nonGoals":         stringArray("adjacent work deliberately excluded"),
					"authorityCeiling": authoritySchema(),
					"stopConditions":   stringArray("moments a human must decide"),
					"assumptions":      stringArray("what you assumed rather than asked"),
					"facet": map[string]any{
						"type": "string",
						"enum": []any{"software", "research", "design", "documentation", "investigation", "evaluation", "operations"},
					},
				},
			},
		},
	}
}

func authoritySchema() map[string]any {
	props := map[string]any{}
	names := []string{"readWorkspace", "writeWorkspace", "executeLocal", "useNetwork", "commitLocal", "createPR", "deploy", "externalEffect"}
	required := make([]any, 0, len(names))
	for _, name := range names {
		props[name] = map[string]any{"type": "boolean"}
		required = append(required, name)
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             required,
		"properties":           props,
		"description":          "the least authority that could complete the outcome",
	}
}

func stringArray(description string) map[string]any {
	return map[string]any{
		"type":        "array",
		"items":       map[string]any{"type": "string"},
		"description": description,
	}
}

func planSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"summary", "workUnits"},
		"properties": map[string]any{
			"summary": map[string]any{"type": "string", "description": "one line on how this plan reaches the outcome"},
			"workUnits": map[string]any{
				"type":     "array",
				"minItems": 1,
				"maxItems": float64(domain.MaxPlanDraftWorkUnits),
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []any{"key", "title", "intent", "outputSummary", "criteriaCovered"},
					"properties": map[string]any{
						"key":   map[string]any{"type": "string", "description": "stable short id such as W1"},
						"title": map[string]any{"type": "string"},
						"intent": map[string]any{
							"type": "string",
							"enum": []any{"inspect", "modify", "execute", "modify_and_execute"},
						},
						"outputSummary":   map[string]any{"type": "string", "description": "the observable result of this unit"},
						"criteriaCovered": stringArray("criterion aliases such as C1"),
						"dependsOn":       stringArray("keys of units that must finish first"),
						"evidenceIdeas":   stringArray("artifacts that would prove this unit"),
					},
				},
			},
			"assumptions": stringArray("real assumptions only"),
			"blockers":    stringArray("real blockers only"),
		},
	}
}

type contractReply struct {
	Decision string `json:"decision"`
	Question *struct {
		Question            string   `json:"question"`
		Reason              string   `json:"reason"`
		Recommendation      string   `json:"recommendation"`
		Alternatives        []string `json:"alternatives"`
		DeferralConsequence string   `json:"deferralConsequence"`
	} `json:"question"`
	Proposal *struct {
		Title        string `json:"title"`
		DesiredState string `json:"desiredState"`
		Criteria     []struct {
			Text             string   `json:"text"`
			EvidenceExpected []string `json:"evidenceExpected"`
		} `json:"criteria"`
		ReviewMethod     string   `json:"reviewMethod"`
		Constraints      []string `json:"constraints"`
		NonGoals         []string `json:"nonGoals"`
		AuthorityCeiling struct {
			ReadWorkspace  bool `json:"readWorkspace"`
			WriteWorkspace bool `json:"writeWorkspace"`
			ExecuteLocal   bool `json:"executeLocal"`
			UseNetwork     bool `json:"useNetwork"`
			CommitLocal    bool `json:"commitLocal"`
			CreatePR       bool `json:"createPR"`
			Deploy         bool `json:"deploy"`
			ExternalEffect bool `json:"externalEffect"`
		} `json:"authorityCeiling"`
		StopConditions []string `json:"stopConditions"`
		Assumptions    []string `json:"assumptions"`
		Facet          string   `json:"facet"`
	} `json:"proposal"`
}

type planReply struct {
	Summary   string `json:"summary"`
	WorkUnits []struct {
		Key             string   `json:"key"`
		Title           string   `json:"title"`
		Intent          string   `json:"intent"`
		OutputSummary   string   `json:"outputSummary"`
		CriteriaCovered []string `json:"criteriaCovered"`
		DependsOn       []string `json:"dependsOn"`
		EvidenceIdeas   []string `json:"evidenceIdeas"`
	} `json:"workUnits"`
	Assumptions []string `json:"assumptions"`
	Blockers    []string `json:"blockers"`
}

// AnalyzeContract asks the model for one clarification or one editable
// proposal. Nothing it returns creates responsibility: the owner still
// confirms, and the control plane still validates.
func (p *LLMProvider) AnalyzeContract(ctx context.Context, request ports.ContractIntelligenceRequest) (ports.ContractIntelligenceResponse, error) {
	if p == nil || p.client == nil {
		return ports.ContractIntelligenceResponse{}, fmt.Errorf("waldo reasoning is not configured")
	}

	var input strings.Builder
	fmt.Fprintf(&input, "Stated outcome:\n%s\n", strings.TrimSpace(request.Session.Statement))
	if answer := strings.TrimSpace(request.ClarificationText); answer != "" {
		question := ""
		if request.Clarification != nil {
			question = strings.TrimSpace(request.Clarification.Question)
		}
		fmt.Fprintf(&input, "\nThe owner was asked: %s\nThey answered: %s\nDo not ask again; propose the Contract.\n", question, answer)
	}
	if request.PreviousProposal != nil {
		fmt.Fprintf(&input, "\nA previous proposal titled %q was not accepted. Produce a better one.\n", request.PreviousProposal.Title)
	}

	response, err := p.client.Complete(ctx, ports.LLMRequest{
		System:     contractSystemPrompt,
		User:       input.String(),
		SchemaName: "contract_proposal",
		Schema:     contractSchema(),
	})
	if err != nil {
		return ports.ContractIntelligenceResponse{}, err
	}

	var reply contractReply
	if err := json.Unmarshal(response.JSON, &reply); err != nil {
		return ports.ContractIntelligenceResponse{}, fmt.Errorf("waldo returned an unreadable contract proposal: %w", err)
	}

	provenance := ports.IntelligenceProvenance{
		EffectiveProvider: LLMProviderID,
		EffectiveModel:    response.EffectiveModel,
	}

	// A clarification is only honoured when the owner has not already
	// answered one: the control plane allows exactly one open ask.
	if reply.Decision == "ask" && reply.Question != nil && strings.TrimSpace(request.ClarificationText) == "" {
		return ports.ContractIntelligenceResponse{
			Result: ports.IntakeAnalysisResult{Clarification: &domain.ClarificationRequest{
				Question:            strings.TrimSpace(reply.Question.Question),
				Reason:              strings.TrimSpace(reply.Question.Reason),
				Recommendation:      strings.TrimSpace(reply.Question.Recommendation),
				Alternatives:        trimAll(reply.Question.Alternatives),
				DeferralConsequence: strings.TrimSpace(reply.Question.DeferralConsequence),
			}},
			Provenance: provenance,
		}, nil
	}

	if reply.Proposal == nil {
		return ports.ContractIntelligenceResponse{}, fmt.Errorf("waldo returned neither a question nor a contract proposal")
	}
	source := reply.Proposal

	criteria := make([]domain.ProposedCriterion, 0, len(source.Criteria))
	for _, criterion := range source.Criteria {
		text := strings.TrimSpace(criterion.Text)
		if text == "" {
			continue
		}
		criteria = append(criteria, domain.ProposedCriterion{
			Text:             text,
			EvidenceExpected: trimAll(criterion.EvidenceExpected),
		})
	}
	if len(criteria) == 0 {
		return ports.ContractIntelligenceResponse{}, fmt.Errorf("waldo proposed a contract with no success criteria")
	}

	proposal := &domain.OutcomeContractProposal{
		Title:        strings.TrimSpace(source.Title),
		DesiredState: strings.TrimSpace(source.DesiredState),
		Criteria:     criteria,
		ReviewMethod: strings.TrimSpace(source.ReviewMethod),
		Constraints:  trimAll(source.Constraints),
		NonGoals:     trimAll(source.NonGoals),
		AuthorityCeiling: domain.ProposedAuthority{
			ReadWorkspace:  source.AuthorityCeiling.ReadWorkspace,
			WriteWorkspace: source.AuthorityCeiling.WriteWorkspace,
			ExecuteLocal:   source.AuthorityCeiling.ExecuteLocal,
			UseNetwork:     source.AuthorityCeiling.UseNetwork,
			CommitLocal:    source.AuthorityCeiling.CommitLocal,
			CreatePR:       source.AuthorityCeiling.CreatePR,
			Deploy:         source.AuthorityCeiling.Deploy,
			ExternalEffect: source.AuthorityCeiling.ExternalEffect,
		},
		StopConditions:     trimAll(source.StopConditions),
		ClarificationNotes: trimAll(source.Assumptions),
		Facets:             []domain.ContractFacet{{Kind: facetKind(source.Facet), Summary: strings.TrimSpace(source.Title)}},
	}
	if answer := strings.TrimSpace(request.ClarificationText); answer != "" {
		proposal.TemporalCondition = &answer
	}

	return ports.ContractIntelligenceResponse{
		Result:     ports.IntakeAnalysisResult{Proposal: proposal},
		Provenance: provenance,
	}, nil
}

// DraftPlan asks the model how the frozen Contract should be executed. The
// reply is non-authoritative: Kennel compiles the canonical PlanRevision.
func (p *LLMProvider) DraftPlan(ctx context.Context, request ports.PlanIntelligenceRequest) (ports.PlanIntelligenceResponse, error) {
	if p == nil || p.client == nil {
		return ports.PlanIntelligenceResponse{}, fmt.Errorf("waldo reasoning is not configured")
	}

	var input strings.Builder
	fmt.Fprintf(&input, "Outcome: %s\n\nGoal:\n%s\n\nSuccess criteria:\n", request.Outcome.Title, request.Contract.Goal)
	for _, alias := range sortedAliasKeys(request.CriterionAliases) {
		fmt.Fprintf(&input, "%s. %s\n", alias, criterionText(request.Contract, request.CriterionAliases[alias]))
	}
	if review := strings.TrimSpace(request.Contract.Review); review != "" {
		fmt.Fprintf(&input, "\nHow the result will be reviewed:\n%s\n", review)
	}
	if len(request.Contract.Constraints) > 0 {
		fmt.Fprintf(&input, "\nConstraints:\n- %s\n", strings.Join(request.Contract.Constraints, "\n- "))
	}
	if len(request.Contract.NonGoals) > 0 {
		fmt.Fprintf(&input, "\nExplicitly not in scope:\n- %s\n", strings.Join(request.Contract.NonGoals, "\n- "))
	}

	response, err := p.client.Complete(ctx, ports.LLMRequest{
		System:     planSystemPrompt,
		User:       input.String(),
		SchemaName: "plan_draft",
		Schema:     planSchema(),
	})
	if err != nil {
		return ports.PlanIntelligenceResponse{}, err
	}

	var reply planReply
	if err := json.Unmarshal(response.JSON, &reply); err != nil {
		return ports.PlanIntelligenceResponse{}, fmt.Errorf("waldo returned an unreadable plan draft: %w", err)
	}

	units := make([]domain.PlanDraftWorkUnit, 0, len(reply.WorkUnits))
	for _, unit := range reply.WorkUnits {
		key := strings.TrimSpace(unit.Key)
		if key == "" {
			continue
		}
		units = append(units, domain.PlanDraftWorkUnit{
			Key:             key,
			Title:           strings.TrimSpace(unit.Title),
			Intent:          domain.WorkUnitIntent(strings.TrimSpace(unit.Intent)),
			OutputSummary:   strings.TrimSpace(unit.OutputSummary),
			CriteriaCovered: trimAll(unit.CriteriaCovered),
			DependsOn:       trimAll(unit.DependsOn),
			EvidenceIdeas:   trimAll(unit.EvidenceIdeas),
		})
	}

	return ports.PlanIntelligenceResponse{
		Proposal: domain.PlanDraftProposal{
			Summary:     strings.TrimSpace(reply.Summary),
			WorkUnits:   units,
			Assumptions: trimAll(reply.Assumptions),
			Blockers:    trimAll(reply.Blockers),
		},
		Provenance: ports.IntelligenceProvenance{
			EffectiveProvider: LLMProviderID,
			EffectiveModel:    response.EffectiveModel,
		},
	}, nil
}

// sortedAliasKeys returns the model-facing criterion aliases in stable order
// so identical Contracts produce identical prompts.
func sortedAliasKeys(aliases map[string]domain.CriterionID) []string {
	keys := make([]string, 0, len(aliases))
	for alias := range aliases {
		keys = append(keys, alias)
	}
	sort.Strings(keys)
	return keys
}

// criterionText resolves one canonical criterion's display text.
func criterionText(contract domain.ContractRevision, id domain.CriterionID) string {
	for _, criterion := range contract.Criteria {
		if criterion.ID == id {
			return criterion.Text
		}
	}
	return ""
}

func trimAll(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func facetKind(raw string) domain.ContractFacetKind {
	switch kind := domain.ContractFacetKind(strings.TrimSpace(strings.ToLower(raw))); kind {
	case domain.ContractFacetSoftware, domain.ContractFacetResearch, domain.ContractFacetDesign,
		domain.ContractFacetDocumentation, domain.ContractFacetInvestigation,
		domain.ContractFacetEvaluation, domain.ContractFacetOperations:
		return kind
	default:
		return domain.ContractFacetSoftware
	}
}
