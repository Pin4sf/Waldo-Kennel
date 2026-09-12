// Package codexpolicy contains the shared enforcement boundary for Codex's
// terminal and app-server adapters.
package codexpolicy

import (
	"fmt"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

const supportedScope = "worktree/*"

// SandboxFor validates the exact policy shapes Codex can enforce without
// widening the admitted WorkUnit. Codex cannot express restricted subpaths or
// arbitrary capability combinations, so those policies must fail closed.
func SandboxFor(policy domain.AttemptExecutionPolicy) (string, error) {
	if err := policy.Validate(); err != nil {
		return "", fmt.Errorf("codex execution policy: %w", err)
	}
	for _, grant := range policy.Grants {
		if grant.Scope != supportedScope {
			return "", unsupported(grant.Name, fmt.Sprintf("scope %q is not enforceable; Codex only admits %q", grant.Scope, supportedScope))
		}
	}

	switch strings.Join(policy.RequiredCapabilities, ",") {
	case domain.CapabilityWorktreeRead:
		return "read-only", nil
	case domain.CapabilityWorktreeExec + "," + domain.CapabilityWorktreeRead + "," + domain.CapabilityWorktreeWrite:
		if len(policy.ApprovedChecks) == 0 {
			return "", unsupported(domain.CapabilityWorktreeExec, "execution was granted without an exact approved check vector")
		}
		return "workspace-write", nil
	default:
		return "", unsupported(strings.Join(policy.RequiredCapabilities, ","), "Codex cannot represent this exact capability set without granting a broader sandbox")
	}
}

func unsupported(capability, detail string) error {
	return &ports.ExecutionPolicyUnsupportedError{
		Harness:    domain.HarnessCodex,
		Capability: capability,
		Detail:     detail,
	}
}
