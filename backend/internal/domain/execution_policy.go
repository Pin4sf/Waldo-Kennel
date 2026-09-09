package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// AttemptExecutionPolicy is the immutable, least-privilege capability packet
// derived from one approved WorkUnit. It is intentionally narrower than a
// Project configuration: mutable preferences cannot widen an admitted
// Attempt. Attribution fields make the policy auditable at the runtime seam.
type AttemptExecutionPolicy struct {
	OutcomeID              OutcomeID         `json:"outcomeId"`
	PlanRevisionID         PlanRevisionID    `json:"planRevisionId"`
	WorkUnitID             WorkUnitID        `json:"workUnitId"`
	ContractRevisionNumber int64             `json:"contractRevisionNumber"`
	RunBriefCoreDigest     string            `json:"runBriefCoreDigest"`
	RequiredCapabilities   []string          `json:"requiredCapabilities"`
	Grants                 []CapabilityGrant `json:"grants"`
}

// BuildAttemptExecutionPolicy selects only grants required by the admitted
// WorkUnit and normalizes their order before they cross the runtime boundary.
func BuildAttemptExecutionPolicy(
	outcomeID OutcomeID,
	plan PlanRevision,
	unit WorkUnit,
	runBriefCoreDigest string,
) (AttemptExecutionPolicy, error) {
	if outcomeID.IsZero() || plan.ID.IsZero() || unit.ID.IsZero() {
		return AttemptExecutionPolicy{}, fmt.Errorf("execution policy requires outcome, plan, and work unit ids")
	}
	if plan.OutcomeID != outcomeID {
		return AttemptExecutionPolicy{}, fmt.Errorf("execution policy plan %s does not belong to outcome %s", plan.ID, outcomeID)
	}
	if strings.TrimSpace(runBriefCoreDigest) == "" {
		return AttemptExecutionPolicy{}, fmt.Errorf("execution policy requires the approved RunBrief digest")
	}

	grantByName := make(map[string]CapabilityGrant, len(plan.Grants))
	for _, grant := range plan.Grants {
		name := strings.TrimSpace(grant.Name)
		if name != "" {
			grant.Name = name
			grant.Scope = strings.TrimSpace(grant.Scope)
			grantByName[name] = grant
		}
	}
	required := append([]string(nil), unit.RequiredCapabilities...)
	for i := range required {
		required[i] = strings.TrimSpace(required[i])
	}
	required = uniqueSortedStrings(required)
	if len(required) == 0 {
		return AttemptExecutionPolicy{}, fmt.Errorf("work unit %s requires at least one capability", unit.ID)
	}

	grants := make([]CapabilityGrant, 0, len(required))
	for _, name := range required {
		grant, ok := grantByName[name]
		if !ok {
			return AttemptExecutionPolicy{}, fmt.Errorf("work unit %s is missing capability grant %q", unit.ID, name)
		}
		if err := grant.Validate(); err != nil {
			return AttemptExecutionPolicy{}, fmt.Errorf("capability grant %s: %w", grant.ID, err)
		}
		grants = append(grants, grant)
	}

	policy := AttemptExecutionPolicy{
		OutcomeID:              outcomeID,
		PlanRevisionID:         plan.ID,
		WorkUnitID:             unit.ID,
		ContractRevisionNumber: plan.ContractRevisionNumber,
		RunBriefCoreDigest:     strings.TrimSpace(runBriefCoreDigest),
		RequiredCapabilities:   required,
		Grants:                 grants,
	}
	if err := policy.Validate(); err != nil {
		return AttemptExecutionPolicy{}, err
	}
	return policy, nil
}

// Validate checks attribution, normalized capability names, and grant parity.
func (p AttemptExecutionPolicy) Validate() error {
	if p.OutcomeID.IsZero() || p.PlanRevisionID.IsZero() || p.WorkUnitID.IsZero() {
		return fmt.Errorf("execution policy attribution is incomplete")
	}
	if p.ContractRevisionNumber < 1 {
		return fmt.Errorf("execution policy contract revision must be positive")
	}
	if strings.TrimSpace(p.RunBriefCoreDigest) == "" {
		return fmt.Errorf("execution policy RunBrief digest is required")
	}
	required := uniqueSortedStrings(append([]string(nil), p.RequiredCapabilities...))
	if len(required) == 0 {
		return fmt.Errorf("execution policy requires at least one capability")
	}
	if len(required) != len(p.RequiredCapabilities) || !equalStrings(required, p.RequiredCapabilities) {
		return fmt.Errorf("execution policy capabilities must be sorted and unique")
	}
	if len(required) != len(p.Grants) {
		return fmt.Errorf("execution policy has %d required capabilities but %d grants", len(required), len(p.Grants))
	}
	for i, grant := range p.Grants {
		if err := grant.Validate(); err != nil {
			return fmt.Errorf("execution policy grant %d: %w", i+1, err)
		}
		if grant.Name != required[i] {
			return fmt.Errorf("execution policy grant %q does not match required capability %q", grant.Name, required[i])
		}
	}
	return nil
}

// Has reports whether this policy grants a named capability.
func (p AttemptExecutionPolicy) Has(name string) bool {
	name = strings.TrimSpace(name)
	for _, required := range p.RequiredCapabilities {
		if required == name {
			return true
		}
	}
	return false
}

// Digest returns the stable identity of the frozen policy packet.
func (p AttemptExecutionPolicy) Digest() (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshal execution policy: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
