package controllers_test

import (
	"context"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// controllerRouting is the execution-routing double for the functional HTTP
// tests. It reports one fully capable, ready candidate so Plan compilation can
// bind a provider without probing the machine.
type controllerRouting struct{}

var _ ports.ExecutionRoutingInventory = controllerRouting{}

func (controllerRouting) RoutingSnapshot(context.Context, domain.ProjectID, *domain.RoutingPreference) (ports.RoutingInventorySnapshot, error) {
	return ports.RoutingInventorySnapshot{
		SnapshotID: "controller-test-snapshot",
		Candidates: []domain.RoutingCandidate{{
			ID:                  string(domain.HarnessCodex),
			Provider:            string(domain.HarnessCodex),
			ModelSelection:      domain.ExecutionBindingModelProviderDefault,
			WorkerEligible:      true,
			CoordinatorEligible: domain.HarnessCodex.IsSelectableAsCoordinator(),
			Readiness:           domain.CapabilitySupported,
			Capabilities: map[string]domain.CapabilitySupport{
				domain.CapabilityWorktreeRead:  domain.CapabilitySupported,
				domain.CapabilityWorktreeWrite: domain.CapabilitySupported,
				domain.CapabilityWorktreeExec:  domain.CapabilitySupported,
			},
			Models: map[string]domain.CapabilitySupport{},
		}},
	}, nil
}
