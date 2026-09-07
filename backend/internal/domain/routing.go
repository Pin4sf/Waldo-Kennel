package domain

import (
	"sort"
	"strings"
)

// RoutingRole keeps worker and coordinator policy separate. A candidate must
// be explicitly eligible for the requested role; worker readiness never
// promotes a provider into coordinator authority.
type RoutingRole string

const (
	RoutingRoleWorker      RoutingRole = "worker"
	RoutingRoleCoordinator RoutingRole = "coordinator"
)

// CapabilitySupport is normalized adapter truth. Unknown remains distinct
// from unsupported so diagnostics stay honest; either fails a mandatory gate.
type CapabilitySupport string

const (
	CapabilitySupported   CapabilitySupport = "supported"
	CapabilityUnsupported CapabilitySupport = "unsupported"
	CapabilityUnknown     CapabilitySupport = "unknown"
)

// RoutingPreference is provider-neutral planning input consumed by the router.
// Provider is opaque data; routing policy never branches on its value.
type RoutingPreference struct {
	Provider       string
	ModelSelection ExecutionPreferenceModelSelection
	Model          string
}

// RoutingRequirements are derived from Outcome/Contract/WorkUnit semantics,
// not provider brands.
type RoutingRequirements struct {
	Role                 RoutingRole
	HardCapabilities     []string
	SoftCapabilities     []string
	ExecutionConstraints []string
	RequiredModel        string
	Preference           *RoutingPreference
}

// RoutingCandidate is a normalized provider/model candidate produced by
// adapters and inventory. Strengths are provider-neutral soft-fit signals.
type RoutingCandidate struct {
	ID                  string
	Provider            string
	ModelSelection      ExecutionBindingModelSelection
	Model               string
	WorkerEligible      bool
	CoordinatorEligible bool
	Readiness           CapabilitySupport
	Capabilities        map[string]CapabilitySupport
	Strengths           map[string]int
	Models              map[string]CapabilitySupport
}

// RoutingCandidateEvaluation preserves why each candidate did or did not pass.
type RoutingCandidateEvaluation struct {
	CandidateID  string   `json:"candidateId"`
	Provider     string   `json:"provider"`
	Admissible   bool     `json:"admissible"`
	Score        int      `json:"score"`
	RejectionCodes []string `json:"rejectionCodes,omitempty"`
	Reasons      []string `json:"reasons,omitempty"`
}

// RoutingDecisionStatus is persisted with the Plan proposal.
type RoutingDecisionStatus string

const (
	RoutingDecisionRecommended     RoutingDecisionStatus = "recommended"
	RoutingDecisionNoValidCandidate RoutingDecisionStatus = "no_valid_candidate"
)

const RoutingPolicyVersion = "wt3-v1"

// RoutingDecision is explainable recommendation state. It is not execution
// authority until a Plan containing the selected binding is approved.
type RoutingDecision struct {
	Status                RoutingDecisionStatus          `json:"status"`
	PolicyVersion         string                         `json:"policyVersion"`
	CapabilitySnapshot    string                         `json:"capabilitySnapshot,omitempty"`
	Role                  RoutingRole                    `json:"role"`
	EffectivePreference   *RoutingPreference             `json:"effectivePreference,omitempty"`
	Requirements          RoutingRequirements            `json:"requirements"`
	RecommendedCandidateID string                        `json:"recommendedCandidateId,omitempty"`
	RecommendedProvider   string                         `json:"recommendedProvider,omitempty"`
	RecommendedModelSelection ExecutionBindingModelSelection `json:"recommendedModelSelection,omitempty"`
	RecommendedModel      string                         `json:"recommendedModel,omitempty"`
	Evaluations           []RoutingCandidateEvaluation   `json:"evaluations"`
}

// RouteExecution deterministically hard-gates then ranks normalized candidates.
// Preference wins an equivalent admissible comparison, while a materially
// stronger soft fit (>=2 points) may beat it. No provider identifier appears
// in policy logic other than opaque equality with the user's preference.
func RouteExecution(req RoutingRequirements, candidates []RoutingCandidate, snapshot string) RoutingDecision {
	decision := RoutingDecision{
		Status:              RoutingDecisionNoValidCandidate,
		PolicyVersion:       RoutingPolicyVersion,
		CapabilitySnapshot:  snapshot,
		Role:                req.Role,
		EffectivePreference: req.Preference,
		Requirements:        req,
		Evaluations:         make([]RoutingCandidateEvaluation, 0, len(candidates)),
	}

	type admitted struct {
		candidate RoutingCandidate
		score     int
		preferred bool
	}
	var admittedCandidates []admitted
	for _, candidate := range candidates {
		eval := RoutingCandidateEvaluation{CandidateID: candidate.ID, Provider: candidate.Provider}
		reject := func(code, reason string) {
			eval.RejectionCodes = append(eval.RejectionCodes, code)
			eval.Reasons = append(eval.Reasons, reason)
		}

		switch req.Role {
		case RoutingRoleWorker:
			if !candidate.WorkerEligible {
				reject("ROLE_INELIGIBLE", "candidate is not eligible for worker execution")
			}
		case RoutingRoleCoordinator:
			if !candidate.CoordinatorEligible {
				reject("ROLE_INELIGIBLE", "candidate is not eligible for coordinator execution")
			}
		default:
			reject("ROLE_UNKNOWN", "requested role is unsupported")
		}

		if candidate.Readiness != CapabilitySupported {
			code := "READINESS_UNKNOWN"
			if candidate.Readiness == CapabilityUnsupported {
				code = "NOT_READY"
			}
			reject(code, "candidate readiness is not confirmed")
		}
		for _, capability := range req.HardCapabilities {
			support := candidate.Capabilities[capability]
			if support != CapabilitySupported {
				code := "CAPABILITY_UNKNOWN"
				if support == CapabilityUnsupported {
					code = "CAPABILITY_UNSUPPORTED"
				}
				reject(code, "mandatory capability "+capability+" is not confirmed")
			}
		}
		if strings.TrimSpace(req.RequiredModel) != "" {
			support := candidate.Models[req.RequiredModel]
			if support != CapabilitySupported {
				code := "MODEL_UNKNOWN"
				if support == CapabilityUnsupported {
					code = "MODEL_UNSUPPORTED"
				}
				reject(code, "required model is not supported")
			}
		}
		if candidate.ModelSelection != ExecutionBindingModelExplicit && candidate.ModelSelection != ExecutionBindingModelProviderDefault {
			reject("MODEL_BINDING_INVALID", "candidate does not provide executable model semantics")
		}
		if candidate.ModelSelection == ExecutionBindingModelExplicit && strings.TrimSpace(candidate.Model) == "" {
			reject("MODEL_BINDING_INVALID", "candidate explicit model is empty")
		}
		if candidate.ModelSelection == ExecutionBindingModelProviderDefault && strings.TrimSpace(candidate.Model) != "" {
			reject("MODEL_BINDING_INVALID", "provider-default candidate names a model")
		}

		for _, soft := range req.SoftCapabilities {
			if candidate.Capabilities[soft] == CapabilitySupported {
				eval.Score += candidate.Strengths[soft]
			}
		}
		eval.Admissible = len(eval.RejectionCodes) == 0
		decision.Evaluations = append(decision.Evaluations, eval)
		if eval.Admissible {
			preferred := req.Preference != nil && candidate.Provider == req.Preference.Provider
			// An explicit preferred model only strengthens this candidate when
			// the candidate can actually bind that exact model. It never causes
			// another provider's model to inherit by name.
			if preferred && req.Preference.ModelSelection == ExecutionPreferenceModelExplicit {
				if candidate.Models[req.Preference.Model] != CapabilitySupported {
					preferred = false
				}
			}
			admittedCandidates = append(admittedCandidates, admitted{candidate: candidate, score: eval.Score, preferred: preferred})
		}
	}

	if len(admittedCandidates) == 0 {
		return decision
	}
	sort.SliceStable(admittedCandidates, func(i, j int) bool {
		a, b := admittedCandidates[i], admittedCandidates[j]
		if a.preferred != b.preferred {
			// Preference wins unless the non-preferred option has materially
			// stronger soft fit.
			if a.preferred && b.score < a.score+2 {
				return true
			}
			if b.preferred && a.score < b.score+2 {
				return false
			}
		}
		if a.score != b.score {
			return a.score > b.score
		}
		return a.candidate.ID < b.candidate.ID
	})
	winner := admittedCandidates[0].candidate
	decision.Status = RoutingDecisionRecommended
	decision.RecommendedCandidateID = winner.ID
	decision.RecommendedProvider = winner.Provider
	decision.RecommendedModelSelection = winner.ModelSelection
	decision.RecommendedModel = winner.Model
	return decision
}
