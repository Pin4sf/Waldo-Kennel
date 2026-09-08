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
	preferred := routingCandidate("z-provider", "provider-z")
	other := routingCandidate("a-provider", "provider-a")
	decision := RouteExecution(RoutingRequirements{
		Role: RoutingRoleWorker, HardCapabilities: []string{CapabilityWorktreeRead},
		Preference: &RoutingPreference{Provider: "provider-z", ModelSelection: ExecutionPreferenceModelProviderDefault},
	}, []RoutingCandidate{other, preferred}, "snapshot")
	if decision.RecommendedProvider != "provider-z" {
		t.Fatalf("recommended provider = %q", decision.RecommendedProvider)
	}
}

func TestRouteExecutionFallsBackDeterministicallyWhenPreferenceUnavailable(t *testing.T) {
	preferred := routingCandidate("preferred", "provider-z")
	preferred.Readiness = CapabilityUnsupported
	first := routingCandidate("a-provider", "provider-a")
	second := routingCandidate("b-provider", "provider-b")
	decision := RouteExecution(RoutingRequirements{
		Role: RoutingRoleWorker, HardCapabilities: []string{CapabilityWorktreeRead},
		Preference: &RoutingPreference{Provider: "provider-z", ModelSelection: ExecutionPreferenceModelProviderDefault},
	}, []RoutingCandidate{second, preferred, first}, "snapshot")
	if decision.RecommendedCandidateID != "a-provider" || decision.RecommendedProvider != "provider-a" {
		t.Fatalf("deterministic fallback = candidate %q provider %q", decision.RecommendedCandidateID, decision.RecommendedProvider)
	}
}

func TestRouteExecutionExplicitPreferredModelIsCandidateLocal(t *testing.T) {
	preferred := routingCandidate("preferred", "provider-z")
	preferred.Models["model-z"] = CapabilityUnsupported
	other := routingCandidate("other", "provider-a")
	// provider-a has no model-z entry at all; that must not make it inadmissible.
	decision := RouteExecution(RoutingRequirements{
		Role: RoutingRoleWorker, HardCapabilities: []string{CapabilityWorktreeRead},
		Preference: &RoutingPreference{Provider: "provider-z", ModelSelection: ExecutionPreferenceModelExplicit, Model: "model-z"},
	}, []RoutingCandidate{preferred, other}, "snapshot")
	if decision.RecommendedProvider != "provider-a" || decision.Status != RoutingDecisionRecommended {
		t.Fatalf("candidate-local model fallback = %#v", decision)
	}
	for _, evaluation := range decision.Evaluations {
		if evaluation.Provider == "provider-a" && !evaluation.Admissible {
			t.Fatalf("unrelated provider was rejected by preferred model: %#v", evaluation)
		}
	}
}

func TestRouteExecutionExplicitPreferredModelBindsExactModel(t *testing.T) {
	preferred := routingCandidate("preferred", "provider-z")
	preferred.Models["model-z"] = CapabilitySupported
	decision := RouteExecution(RoutingRequirements{
		Role: RoutingRoleWorker, HardCapabilities: []string{CapabilityWorktreeRead},
		Preference: &RoutingPreference{Provider: "provider-z", ModelSelection: ExecutionPreferenceModelExplicit, Model: "model-z"},
	}, []RoutingCandidate{preferred}, "snapshot")
	if decision.RecommendedModelSelection != ExecutionBindingModelExplicit || decision.RecommendedModel != "model-z" {
		t.Fatalf("recommended model semantics = %q %q", decision.RecommendedModelSelection, decision.RecommendedModel)
	}
}

func TestRouteExecutionRejectsUnknownHardCapability(t *testing.T) {
	candidate := routingCandidate("provider", "provider-a")
	candidate.Capabilities[CapabilityWorktreeWrite] = CapabilityUnknown
	decision := RouteExecution(RoutingRequirements{
		Role: RoutingRoleWorker, HardCapabilities: []string{CapabilityWorktreeWrite},
	}, []RoutingCandidate{candidate}, "snapshot")
	if decision.Status != RoutingDecisionNoValidCandidate {
		t.Fatalf("unknown hard capability produced recommendation: %#v", decision)
	}
}

func TestRouteExecutionKeepsWorkerAndCoordinatorAdmissionIndependent(t *testing.T) {
	candidate := routingCandidate("provider", "provider-a")
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
