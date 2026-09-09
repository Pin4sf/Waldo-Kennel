package sessionmanager

import (
	"context"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// validateAgentExecutionPolicy is shared by readiness and the launch path.
// Keeping one check at both boundaries prevents a provider from passing a
// preflight probe and then receiving a broader or different policy at spawn.
func validateAgentExecutionPolicy(
	ctx context.Context,
	agent ports.Agent,
	harness domain.AgentHarness,
	config ports.AgentConfig,
	policy *domain.AttemptExecutionPolicy,
) error {
	if policy == nil {
		return nil
	}
	if err := policy.Validate(); err != nil {
		return fmt.Errorf("invalid execution policy: %w", err)
	}
	checker, ok := agent.(ports.AgentExecutionPolicyChecker)
	if !ok {
		return &ports.ExecutionPolicyUnsupportedError{
			Harness:    harness,
			Capability: policy.RequiredCapabilities[0],
			Detail:     "the adapter has no behavioral capability-enforcement mapping",
		}
	}
	if err := checker.ValidateExecutionPolicy(ctx, config, *policy); err != nil {
		return err
	}
	return nil
}
