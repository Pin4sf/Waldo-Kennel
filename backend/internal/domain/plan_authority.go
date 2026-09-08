package domain

import (
	"fmt"
	"sort"
	"strings"
)

// RequiredPlanCapabilities is the deterministic union of the minimum
// capabilities derived for every WorkUnit. Model output never grants authority.
func RequiredPlanCapabilities(units []WorkUnit) []string {
	seen := make(map[string]struct{})
	for _, unit := range units {
		for _, raw := range unit.RequiredCapabilities {
			name := strings.TrimSpace(raw)
			if name != "" {
				seen[name] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ValidateExactPlanCapabilityGrants rejects both under-granting and
// over-granting. Plan-level authority is currently exactly the union of
// WorkUnit requirements; a future independent Plan-level capability must be
// modeled explicitly rather than smuggled in as an unused grant.
func ValidateExactPlanCapabilityGrants(grants []CapabilityGrant, units []WorkUnit) error {
	required := RequiredPlanCapabilities(units)
	requiredSet := make(map[string]struct{}, len(required))
	for _, name := range required {
		requiredSet[name] = struct{}{}
	}

	grantedSet := make(map[string]struct{}, len(grants))
	for _, grant := range grants {
		name := strings.TrimSpace(grant.Name)
		if name != "" {
			grantedSet[name] = struct{}{}
		}
	}

	var missing []string
	for _, name := range required {
		if _, ok := grantedSet[name]; !ok {
			missing = append(missing, name)
		}
	}
	var excess []string
	for name := range grantedSet {
		if _, ok := requiredSet[name]; !ok {
			excess = append(excess, name)
		}
	}
	sort.Strings(excess)

	switch {
	case len(missing) > 0 && len(excess) > 0:
		return fmt.Errorf("plan capability grants are not least-privilege: missing %s; unused %s", strings.Join(missing, ", "), strings.Join(excess, ", "))
	case len(missing) > 0:
		return fmt.Errorf("plan capability grants are missing required capabilities: %s", strings.Join(missing, ", "))
	case len(excess) > 0:
		return fmt.Errorf("plan capability grants include unused capabilities: %s", strings.Join(excess, ", "))
	default:
		return nil
	}
}
