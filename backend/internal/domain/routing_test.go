package domain

import "testing"

func routingCandidate(id, provider string) RoutingCandidate {
	return RoutingCandidate{
		ID: id, Provider: provider, ModelSelection: ExecutionBindingModelProviderDefault,
		WorkerEligible: true, Readiness: CapabilitySupported,
		Capabilities: map[string]CapabilitySupport{CapabilityWorktreeRead: CapabilitySupported},
		Models:       map[string]CapabilitySupport{},
	}
}

func TestRouteExecutionUsesAdmissiblePreference(t *testing.T) {
	preferred := routingCandidate("z-provider", "claude-code")
	other := routingCandidate("a-provider", "codex")
	decision := RouteExecution(RoutingRequirements{
		Role: RoutingRoleWorker, HardCapabilities: []string{CapabilityWorktreeRead},
		Preference: &RoutingPreference{Provider: "claude-code", ModelSelection: ExecutionPreferenceModelProviderDefault},
	}, []RoutingCandidate{other, preferred}, "snapshot")
	if decision.RecommendedProvider != "claude-code" {
		t.Fatalf("recommended provider = %q", decision.RecommendedProvider)
	}
}

func TestRouteExecutionFallsBackDeterministicallyWhenPreferenceUnavailable(t *testing.T) {
	preferred := routingCandidate("preferred", "claude-code")
	preferred.Readiness = CapabilityUnsupported
	first := routingCandidate("a-provider", "codex")
	second := routingCandidate("b-provider", "opencode")
	decision := RouteExecution(RoutingRequirements{
		Role: RoutingRoleWorker, HardCapabilities: []string{CapabilityWorktreeRead},
		Preference: &RoutingPreference{Provider: "claude-code", ModelSelection: ExecutionPreferenceModelProviderDefault},
	}, []RoutingCandidate{second, preferred, first}, "snapshot")
	if decision.RecommendedCandidateID != "a-provider" || decision.RecommendedProvider != "codex" {
		t.Fatalf("deterministic fallback = candidate %q provider %q", decision.RecommendedCandidateID, decision.RecommendedProvider)
	}
}

func TestRouteExecutionExplicitPreferredModelIsCandidateLocal(t *testing.T) {
	preferred := routingCandidate("preferred", "claude-code")
	preferred.Models["model-z"] = CapabilityUnsupported
	other := routingCandidate("other", "codex")
	// codex has no model-z entry at all; that must not make it inadmissible.
	decision := RouteExecution(RoutingRequirements{
		Role: RoutingRoleWorker, HardCapabilities: []string{CapabilityWorktreeRead},
		Preference: &RoutingPreference{Provider: "claude-code", ModelSelection: ExecutionPreferenceModelExplicit, Model: "model-z"},
	}, []RoutingCandidate{preferred, other}, "snapshot")
	if decision.RecommendedProvider != "codex" || decision.Status != RoutingDecisionRecommended {
		t.Fatalf("candidate-local model fallback = %#v", decision)
	}
	for _, evaluation := range decision.Evaluations {
		if evaluation.Provider == "codex" && !evaluation.Admissible {
			t.Fatalf("unrelated provider was rejected by preferred model: %#v", evaluation)
		}
	}
}

func TestRouteExecutionExplicitPreferredModelBindsExactModel(t *testing.T) {
	preferred := routingCandidate("preferred", "claude-code")
	preferred.Models["model-z"] = CapabilitySupported
	decision := RouteExecution(RoutingRequirements{
		Role: RoutingRoleWorker, HardCapabilities: []string{CapabilityWorktreeRead},
		Preference: &RoutingPreference{Provider: "claude-code", ModelSelection: ExecutionPreferenceModelExplicit, Model: "model-z"},
	}, []RoutingCandidate{preferred}, "snapshot")
	if decision.RecommendedModelSelection != ExecutionBindingModelExplicit || decision.RecommendedModel != "model-z" {
		t.Fatalf("recommended model semantics = %q %q", decision.RecommendedModelSelection, decision.RecommendedModel)
	}
}

func TestRouteExecutionRejectsUnknownHardCapability(t *testing.T) {
	candidate := routingCandidate("provider", "codex")
	candidate.Capabilities[CapabilityWorktreeWrite] = CapabilityUnknown
	decision := RouteExecution(RoutingRequirements{
		Role: RoutingRoleWorker, HardCapabilities: []string{CapabilityWorktreeWrite},
	}, []RoutingCandidate{candidate}, "snapshot")
	if decision.Status != RoutingDecisionNoValidCandidate {
		t.Fatalf("unknown hard capability produced recommendation: %#v", decision)
	}
}

func TestRouteExecutionKeepsWorkerAndCoordinatorAdmissionIndependent(t *testing.T) {
	candidate := routingCandidate("provider", "codex")
	candidate.CoordinatorEligible = false
	worker := RouteExecution(RoutingRequirements{Role: RoutingRoleWorker, HardCapabilities: []string{CapabilityWorktreeRead}}, []RoutingCandidate{candidate}, "snapshot")
	coordinator := RouteExecution(RoutingRequirements{Role: RoutingRoleCoordinator, HardCapabilities: []string{CapabilityWorktreeRead}}, []RoutingCandidate{candidate}, "snapshot")
	if worker.Status != RoutingDecisionRecommended || coordinator.Status != RoutingDecisionNoValidCandidate {
		t.Fatalf("worker=%s coordinator=%s", worker.Status, coordinator.Status)
	}
}

func TestRouteExecutionNoImplicitProviderWhenNoCandidate(t *testing.T) {
	decision := RouteExecution(RoutingRequirements{Role: RoutingRoleWorker}, nil, "snapshot")
	if decision.Status != RoutingDecisionNoValidCandidate || decision.RecommendedProvider != "" {
		t.Fatalf("empty inventory invented provider: %#v", decision)
	}
}
