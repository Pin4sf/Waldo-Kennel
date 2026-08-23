package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// PlanRevisionID identifies one immutable plan revision of an Outcome.
type PlanRevisionID string

// IsZero reports whether the id is unset or blank.
func (id PlanRevisionID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// String returns the raw identifier value.
func (id PlanRevisionID) String() string {
	return string(id)
}

// WorkUnitID identifies one unit of planned work inside a PlanRevision.
type WorkUnitID string

// IsZero reports whether the id is unset or blank.
func (id WorkUnitID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// String returns the raw identifier value.
func (id WorkUnitID) String() string {
	return string(id)
}

// CapabilityGrantID identifies one scoped capability grant of a PlanRevision.
type CapabilityGrantID string

// IsZero reports whether the id is unset or blank.
func (id CapabilityGrantID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// String returns the raw identifier value.
func (id CapabilityGrantID) String() string {
	return string(id)
}

// PlanStatus is the durable approval state of a PlanRevision. Proposals are
// created proposed; only an explicit owner approval moves them to approved.
// There is deliberately no completed state: execution facts (#31) and owner
// acceptance (#35) live further down the lineage.
type PlanStatus string

const (
	// PlanStatusProposed marks a validated but not yet authorized plan.
	PlanStatusProposed PlanStatus = "proposed"
	// PlanStatusApproved marks an owner-authorized plan whose capability
	// grants are active. Approval is final for this revision; supersession
	// happens only through a new contract revision binding a new plan.
	PlanStatusApproved PlanStatus = "approved"
)

// Valid reports whether s is a supported plan status.
func (s PlanStatus) Valid() bool {
	switch s {
	case PlanStatusProposed, PlanStatusApproved:
		return true
	}
	return false
}

// V0 capability names. The first slice is local-only by the locked effect
// ceiling; anything outside this set fails authorization closed.
const (
	CapabilityWorktreeRead  = "worktree.read"
	CapabilityWorktreeWrite = "worktree.write"
	CapabilityWorktreeExec  = "worktree.exec"
)

// V0RequiredCapabilities lists the capabilities every direct Work Unit needs.
// They are required: absence fails authorization closed rather than silently
// narrowing the unit.
var V0RequiredCapabilities = []string{
	CapabilityWorktreeRead,
	CapabilityWorktreeWrite,
	CapabilityWorktreeExec,
}

// WorkUnitKind names the shape of a planned unit. v0 admits only the single
// direct unit; delegation shapes arrive with later slices and would be a
// product-contract change.
type WorkUnitKind string

// WorkUnitDirect is one smallest-sufficient unit executed directly against
// the isolated worktree.
const WorkUnitDirect WorkUnitKind = "direct"

// Valid reports whether k is a supported work-unit kind.
func (k WorkUnitKind) Valid() bool {
	return k == WorkUnitDirect
}

// WorkUnit is the single planned unit of execution bound to one contract
// revision. It states what will be produced, how success will be evidenced,
// and where the unit must stop — before any agent is admitted (#31).
type WorkUnit struct {
	ID                      WorkUnitID
	Kind                    WorkUnitKind
	Title                   string
	ContractRevisionNumber  int64
	OutputSummary           string
	EvidenceChecks          []string
	VerificationRequirement string
	StopConditions          []string
}

// Validate checks intrinsic work-unit invariants.
func (w WorkUnit) Validate() error {
	if w.ID.IsZero() {
		return fmt.Errorf("work unit id is required")
	}
	if !w.Kind.Valid() {
		return fmt.Errorf("unsupported work unit kind %q", w.Kind)
	}
	if strings.TrimSpace(w.Title) == "" {
		return fmt.Errorf("work unit title is required")
	}
	if w.ContractRevisionNumber < 1 {
		return fmt.Errorf("work unit contract revision number must be at least 1")
	}
	if strings.TrimSpace(w.OutputSummary) == "" {
		return fmt.Errorf("work unit output summary is required")
	}
	if len(w.EvidenceChecks) == 0 {
		return fmt.Errorf("work unit requires at least one evidence check")
	}
	for i, check := range w.EvidenceChecks {
		if strings.TrimSpace(check) == "" {
			return fmt.Errorf("evidence check %d is blank", i+1)
		}
	}
	if strings.TrimSpace(w.VerificationRequirement) == "" {
		return fmt.Errorf("work unit verification requirement is required")
	}
	for i, stop := range w.StopConditions {
		if strings.TrimSpace(stop) == "" {
			return fmt.Errorf("stop condition %d is blank", i+1)
		}
	}
	return nil
}

// CapabilityGrant is one named capability scoped to the isolated worktree.
// Grants are part of the frozen brief: they activate on approval and are
// never widened afterward — a different scope is a new plan.
type CapabilityGrant struct {
	ID    CapabilityGrantID
	Name  string
	Scope string
}

// Validate checks intrinsic grant invariants. Whether the name is admissible
// at all is an authority question decided above this type.
func (g CapabilityGrant) Validate() error {
	if g.ID.IsZero() {
		return fmt.Errorf("capability grant id is required")
	}
	if strings.TrimSpace(g.Name) == "" {
		return fmt.Errorf("capability grant name is required")
	}
	if strings.TrimSpace(g.Scope) == "" {
		return fmt.Errorf("capability grant scope is required")
	}
	return nil
}

// PlanRevision is one immutable proposal-and-authorization record binding the
// current contract revision to exactly one direct Work Unit, its scoped
// capability grants, and the RunBrief core digest computed over that freeze.
//
// A later contract revision supersedes every plan bound to earlier revisions;
// prior plans stay readable history and are never updated in place.
type PlanRevision struct {
	ID                     PlanRevisionID
	OutcomeID              OutcomeID
	Number                 int64
	ContractRevisionNumber int64
	Status                 PlanStatus
	Summary                string
	WorkUnits              []WorkUnit
	Grants                 []CapabilityGrant
	RunBriefCoreDigest     string
	RunBriefCompiledDigest string
	CreatedAt              time.Time
}

// Validate checks intrinsic plan invariants. Plan-number uniqueness per
// Outcome and status transitions are enforced by storage alongside
// immutability.
func (p PlanRevision) Validate() error {
	if p.ID.IsZero() {
		return fmt.Errorf("plan revision id is required")
	}
	if p.OutcomeID.IsZero() {
		return fmt.Errorf("plan revision outcome id is required")
	}
	if p.Number < 1 {
		return fmt.Errorf("plan revision number must be at least 1")
	}
	if p.ContractRevisionNumber < 1 {
		return fmt.Errorf("plan revision must bind a contract revision of at least 1")
	}
	if !p.Status.Valid() {
		return fmt.Errorf("unsupported plan status %q", p.Status)
	}
	if len(p.WorkUnits) != 1 {
		return fmt.Errorf("plan revision requires exactly one work unit, got %d", len(p.WorkUnits))
	}
	if p.WorkUnits[0].Kind != WorkUnitDirect {
		return fmt.Errorf("plan revision work unit must be %q", WorkUnitDirect)
	}
	if err := p.WorkUnits[0].Validate(); err != nil {
		return err
	}
	if len(p.Grants) == 0 {
		return fmt.Errorf("plan revision requires at least one capability grant")
	}
	seen := make(map[string]bool, len(p.Grants))
	for i, grant := range p.Grants {
		if err := grant.Validate(); err != nil {
			return fmt.Errorf("grant %d: %w", i, err)
		}
		if seen[grant.Name] {
			return fmt.Errorf("duplicate capability grant %q", grant.Name)
		}
		seen[grant.Name] = true
	}
	if len(p.RunBriefCoreDigest) != 64 {
		return fmt.Errorf("plan revision requires the run brief core digest")
	}
	return nil
}

// BindsCurrentContract reports whether the plan still names the Outcome's
// current revision. A false answer is the material-change signal: the plan is
// unapprovable and a new proposal must compute a fresh RunBrief.
func (p PlanRevision) BindsCurrentContract(currentRevision int64) bool {
	return p.ContractRevisionNumber == currentRevision
}

// AuthorityIntersection intersects capability-name allow-lists across layers
// (contract constraints, project policy ceiling, runtime admission). A
// capability survives only if every layer allows it, so a lower layer can
// never widen an upper layer's decision. Duplicates within a layer collapse.
func AuthorityIntersection(layers ...[]string) []string {
	var intersection []string
	for i, layer := range layers {
		allowed := make(map[string]bool, len(layer))
		for _, name := range layer {
			allowed[name] = true
		}
		if i == 0 {
			intersection = make([]string, 0, len(allowed))
			for name := range allowed {
				intersection = append(intersection, name)
			}
			sort.Strings(intersection)
			continue
		}
		kept := intersection[:0]
		for _, name := range intersection {
			if allowed[name] {
				kept = append(kept, name)
			}
		}
		intersection = kept
	}
	sort.Strings(intersection)
	return intersection
}

// GrantsFailClosed verifies every grant name is present in the authoritative
// allow-list. Unknown or narrowed-out capabilities abort authorization with
// the offender named — they are never silently dropped.
func GrantsFailClosed(grants []CapabilityGrant, authoritative []string) error {
	allowed := make(map[string]bool, len(authoritative))
	for _, name := range authoritative {
		allowed[name] = true
	}
	for _, grant := range grants {
		if !allowed[grant.Name] {
			return fmt.Errorf("capability %q is not authorized by every authority layer", grant.Name)
		}
	}
	return nil
}

// MissingRequiredCapabilities names required capabilities absent from grants.
func MissingRequiredCapabilities(grants []CapabilityGrant) []string {
	present := make(map[string]bool, len(grants))
	for _, grant := range grants {
		present[grant.Name] = true
	}
	var missing []string
	for _, name := range V0RequiredCapabilities {
		if !present[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

// runBriefCore is the canonical serialization hashed into the core digest.
// Slice fields are sorted so semantically identical briefs hash identically
// regardless of construction order.
type runBriefCore struct {
	ContractRevisionNumber  int64    `json:"contractRevisionNumber"`
	Goal                    string   `json:"goal"`
	SuccessCriteria         []string `json:"successCriteria"`
	Review                  string   `json:"review"`
	Constraints             []string `json:"constraints"`
	NonGoals                []string `json:"nonGoals"`
	Clarification           string   `json:"clarification"`
	WorkUnitTitle           string   `json:"workUnitTitle"`
	WorkUnitOutput          string   `json:"workUnitOutput"`
	EvidenceChecks          []string `json:"evidenceChecks"`
	VerificationRequirement string   `json:"verificationRequirement"`
	StopConditions          []string `json:"stopConditions"`
	Grants                  []string `json:"grants"`
}

// ComputeRunBriefCoreDigest freezes the contract content together with the
// single Work Unit and grants into the provider-neutral core digest. Any
// material change to those inputs yields a different digest, which is what
// forces a fresh brief (and therefore a fresh authorization) downstream.
func ComputeRunBriefCoreDigest(revision ContractRevision, unit WorkUnit, grants []CapabilityGrant) (string, error) {
	if err := revision.Validate(); err != nil {
		return "", fmt.Errorf("run brief contract: %w", err)
	}
	if err := unit.Validate(); err != nil {
		return "", fmt.Errorf("run brief work unit: %w", err)
	}

	criteria := sortedTrimmed(revision.SuccessCriteria)
	constraints := sortedTrimmed(revision.Constraints)
	nonGoals := sortedTrimmed(revision.NonGoals)
	checks := sortedTrimmed(unit.EvidenceChecks)
	stops := sortedTrimmed(unit.StopConditions)
	names := make([]string, 0, len(grants))
	for _, grant := range grants {
		names = append(names, grant.Name+"@"+grant.Scope)
	}
	sort.Strings(names)

	core := runBriefCore{
		ContractRevisionNumber:  revision.Number,
		Goal:                    revision.Goal,
		SuccessCriteria:         criteria,
		Review:                  revision.Review,
		Constraints:             constraints,
		NonGoals:                nonGoals,
		Clarification:           revision.Clarification,
		WorkUnitTitle:           unit.Title,
		WorkUnitOutput:          unit.OutputSummary,
		EvidenceChecks:          checks,
		VerificationRequirement: unit.VerificationRequirement,
		StopConditions:          stops,
		Grants:                  names,
	}
	encoded, err := json.Marshal(core)
	if err != nil {
		return "", fmt.Errorf("encode run brief core: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func sortedTrimmed(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
