package domain

import (
	"strings"
	"testing"
)

func TestValidateExactPlanCapabilityGrants(t *testing.T) {
	units := []WorkUnit{
		{ID: "read", RequiredCapabilities: []string{CapabilityWorktreeRead}},
		{ID: "edit", RequiredCapabilities: []string{CapabilityWorktreeRead, CapabilityWorktreeWrite}},
	}
	grants := func(names ...string) []CapabilityGrant {
		out := make([]CapabilityGrant, 0, len(names))
		for i, name := range names {
			out = append(out, CapabilityGrant{ID: CapabilityGrantID("grant-" + string(rune('a'+i))), Name: name, Scope: "worktree/*"})
		}
		return out
	}

	t.Run("accepts exact union", func(t *testing.T) {
		if err := ValidateExactPlanCapabilityGrants(grants(CapabilityWorktreeRead, CapabilityWorktreeWrite), units); err != nil {
			t.Fatalf("exact grants rejected: %v", err)
		}
	})

	t.Run("rejects missing grant", func(t *testing.T) {
		err := ValidateExactPlanCapabilityGrants(grants(CapabilityWorktreeRead), units)
		if err == nil || !strings.Contains(err.Error(), CapabilityWorktreeWrite) {
			t.Fatalf("missing grant error=%v", err)
		}
	})

	t.Run("rejects unused grant", func(t *testing.T) {
		err := ValidateExactPlanCapabilityGrants(grants(CapabilityWorktreeRead, CapabilityWorktreeWrite, CapabilityWorktreeExec), units)
		if err == nil || !strings.Contains(err.Error(), "unused") || !strings.Contains(err.Error(), CapabilityWorktreeExec) {
			t.Fatalf("unused grant error=%v", err)
		}
	})
}
