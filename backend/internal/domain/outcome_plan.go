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

func (id PlanRevisionID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }
func (id PlanRevisionID) String() string { return string(id) }

// WorkUnitID identifies one unit of planned work inside a PlanRevision.
type WorkUnitID string

func (id WorkUnitID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }
func (id WorkUnitID) String() string { return string(id) }

// CapabilityGrantID identifies one scoped capability grant of a PlanRevision.
type CapabilityGrantID string

func (id CapabilityGrantID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }
func (id CapabilityGrantID) String() string { return string(id) }

type PlanStatus string

const (
	PlanStatusProposed PlanStatus = "proposed"
	PlanStatusApproved PlanStatus = "approved"
)

func (s PlanStatus) Valid() bool {
	switch s {
	case PlanStatusProposed, PlanStatusApproved:
		return true
	}
	return false
}

const (
	CapabilityWorktreeRead  = "worktree.read"
	CapabilityWorktreeWrite = "worktree.write"
	CapabilityWorktreeExec  = "worktree.exec"
)

var V0RequiredCapabilities = []string{
	CapabilityWorktreeRead,
	CapabilityWorktreeWrite,
	CapabilityWorktreeExec,
}

type WorkUnitKind string
const WorkUnitDirect WorkUnitKind = "direct"
func (k WorkUnitKind) Valid() bool { return k == WorkUnitDirect }

// WorkUnit is the immutable unit-level execution authority once its Plan is
// approved. Provider + model selection + model are one binding; Provider may
// be empty only for pre-0112 history, and ModelSelection may be empty only for
// pre-WT3 provider-bound history. Neither historical state is executable.
type WorkUnit struct {
	ID                      WorkUnitID
	Kind                    WorkUnitKind
	Title                   string
	ContractRevisionNumber  int64
	Provider                AgentHarness
	ModelSelection          ExecutionBindingModelSelection
	Model                   string
	OutputSummary           string
	EvidenceChecks          []string
	VerificationRequirement string
	StopConditions          []string
}

func (w WorkUnit) Validate() error {
	if w.ID.IsZero() { return fmt.Errorf("work unit id is required") }
	if !w.Kind.Valid() { return fmt.Errorf("unsupported work unit kind %q", w.Kind) }
	if strings.TrimSpace(w.Title) == "" { return fmt.Errorf("work unit title is required") }
	if w.ContractRevisionNumber < 1 { return fmt.Errorf("work unit contract revision number must be at least 1") }
	// Preserve historical provider/model-unbound rows as readable. New WT3
	// paths call ExecutionBindingForNewWork explicitly before persistence.
	if w.Provider != "" && w.ModelSelection != "" {
		if _, err := w.ExecutionBinding(); err != nil { return err }
	}
	if strings.TrimSpace(w.OutputSummary) == "" { return fmt.Errorf("work unit output summary is required") }
	if len(w.EvidenceChecks) == 0 { return fmt.Errorf("work unit requires at least one evidence check") }
	for i, check := range w.EvidenceChecks {
		if strings.TrimSpace(check) == "" { return fmt.Errorf("evidence check %d is blank", i+1) }
	}
	if strings.TrimSpace(w.VerificationRequirement) == "" { return fmt.Errorf("work unit verification requirement is required") }
	for i, stop := range w.StopConditions {
		if strings.TrimSpace(stop) == "" { return fmt.Errorf("stop condition %d is blank", i+1) }
	}
	return nil
}

// ExecutionBinding returns the exact binding represented by the WorkUnit. A
// provider-bound historical row with no model semantics is surfaced explicitly
// as historical_unbound instead of silently choosing a provider default.
func (w WorkUnit) ExecutionBinding() (ExecutionBinding, error) {
	if strings.TrimSpace(string(w.Provider)) == "" {
		return ExecutionBinding{}, fmt.Errorf("execution binding provider is required")
	}
	selection := w.ModelSelection
	if selection == "" {
		selection = ExecutionBindingModelHistoricalUnbound
	}
	binding := ExecutionBinding{Provider: w.Provider, ModelSelection: selection, Model: w.Model}
	if err := binding.ValidateReadable(); err != nil { return ExecutionBinding{}, err }
	return binding, nil
}

// ExecutionBindingForNewWork additionally rejects historical compatibility
// state so plan proposal/approval and Attempt admission cannot reinterpret it.
func (w WorkUnit) ExecutionBindingForNewWork() (ExecutionBinding, error) {
	binding, err := w.ExecutionBinding()
	if err != nil { return ExecutionBinding{}, err }
	if err := binding.ValidateForNewWork(); err != nil { return ExecutionBinding{}, err }
	return binding, nil
}

// BindExecution writes the binding fields together in memory before the
// append-only WorkUnit is persisted.
func (w *WorkUnit) BindExecution(binding ExecutionBinding) error {
	if err := binding.ValidateForNewWork(); err != nil { return err }
	w.Provider = binding.Provider
	w.ModelSelection = binding.ModelSelection
	w.Model = binding.Model
	return nil
}

type CapabilityGrant struct {
	ID CapabilityGrantID
	Name string
	Scope string
}
func (g CapabilityGrant) Validate() error {
	if g.ID.IsZero() { return fmt.Errorf("capability grant id is required") }
	if strings.TrimSpace(g.Name) == "" { return fmt.Errorf("capability grant name is required") }
	if strings.TrimSpace(g.Scope) == "" { return fmt.Errorf("capability grant scope is required") }
	return nil
}

// PlanRevision records recommendation state and becomes authority only through
// explicit owner approval. RoutingDecision remains evidence for why the
// proposed binding was selected; it never replaces the WorkUnit binding.
type PlanRevision struct {
	ID                     PlanRevisionID
	OutcomeID              OutcomeID
	Number                 int64
	ContractRevisionNumber int64
	Status                 PlanStatus
	Summary                string
	WorkUnits              []WorkUnit
	Grants                 []CapabilityGrant
	RoutingDecision        *RoutingDecision
	RunBriefCoreDigest     string
	RunBriefCompiledDigest string
	CreatedAt              time.Time
}

func (p PlanRevision) Validate() error {
	if p.ID.IsZero() { return fmt.Errorf("plan revision id is required") }
	if p.OutcomeID.IsZero() { return fmt.Errorf("plan revision outcome id is required") }
	if p.Number < 1 { return fmt.Errorf("plan revision number must be at least 1") }
	if p.ContractRevisionNumber < 1 { return fmt.Errorf("plan revision must bind a contract revision of at least 1") }
	if !p.Status.Valid() { return fmt.Errorf("unsupported plan status %q", p.Status) }
	if len(p.WorkUnits) != 1 { return fmt.Errorf("plan revision requires exactly one work unit, got %d", len(p.WorkUnits)) }
	if p.WorkUnits[0].Kind != WorkUnitDirect { return fmt.Errorf("plan revision work unit must be %q", WorkUnitDirect) }
	if err := p.WorkUnits[0].Validate(); err != nil { return err }
	if len(p.Grants) == 0 { return fmt.Errorf("plan revision requires at least one capability grant") }
	seen := make(map[string]bool, len(p.Grants))
	for i, grant := range p.Grants {
		if err := grant.Validate(); err != nil { return fmt.Errorf("grant %d: %w", i, err) }
		if seen[grant.Name] { return fmt.Errorf("duplicate capability grant %q", grant.Name) }
		seen[grant.Name] = true
	}
	if len(p.RunBriefCoreDigest) != 64 { return fmt.Errorf("plan revision requires the run brief core digest") }
	return nil
}

func (p PlanRevision) BindsCurrentContract(currentRevision int64) bool {
	return p.ContractRevisionNumber == currentRevision
}

func AuthorityIntersection(layers ...[]string) []string {
	var intersection []string
	for i, layer := range layers {
		allowed := make(map[string]bool, len(layer))
		for _, name := range layer { allowed[name] = true }
		if i == 0 {
			intersection = make([]string, 0, len(allowed))
			for name := range allowed { intersection = append(intersection, name) }
			sort.Strings(intersection)
			continue
		}
		kept := intersection[:0]
		for _, name := range intersection { if allowed[name] { kept = append(kept, name) } }
		intersection = kept
	}
	sort.Strings(intersection)
	return intersection
}

func GrantsFailClosed(grants []CapabilityGrant, authoritative []string) error {
	allowed := make(map[string]bool, len(authoritative))
	for _, name := range authoritative { allowed[name] = true }
	for _, grant := range grants {
		if !allowed[grant.Name] { return fmt.Errorf("capability %q is not authorized by every authority layer", grant.Name) }
	}
	return nil
}

func MissingRequiredCapabilities(grants []CapabilityGrant) []string {
	present := make(map[string]bool, len(grants))
	for _, grant := range grants { present[grant.Name] = true }
	var missing []string
	for _, name := range V0RequiredCapabilities { if !present[name] { missing = append(missing, name) } }
	return missing
}

type runBriefCore struct {
	ContractRevisionNumber  int64    `json:"contractRevisionNumber"`
	Goal                    string   `json:"goal"`
	SuccessCriteria         []string `json:"successCriteria"`
	Review                  string   `json:"review"`
	Constraints             []string `json:"constraints"`
	NonGoals                []string `json:"nonGoals"`
	Clarification           string   `json:"clarification"`
	WorkUnitTitle           string   `json:"workUnitTitle"`
	WorkUnitProvider        string   `json:"workUnitProvider,omitempty"`
	WorkUnitModelSelection  string   `json:"workUnitModelSelection,omitempty"`
	WorkUnitModel           string   `json:"workUnitModel,omitempty"`
	WorkUnitOutput          string   `json:"workUnitOutput"`
	EvidenceChecks          []string `json:"evidenceChecks"`
	VerificationRequirement string   `json:"verificationRequirement"`
	StopConditions          []string `json:"stopConditions"`
	Grants                  []string `json:"grants"`
}

// ComputeRunBriefCoreDigest includes the exact provider/model semantics. Model
// changes therefore require a fresh Plan and owner approval just like provider
// changes; Attempt can never reinterpret mutable Project configuration.
func ComputeRunBriefCoreDigest(revision ContractRevision, unit WorkUnit, grants []CapabilityGrant) (string, error) {
	if err := revision.Validate(); err != nil { return "", fmt.Errorf("run brief contract: %w", err) }
	if err := unit.Validate(); err != nil { return "", fmt.Errorf("run brief work unit: %w", err) }
	criteria := sortedTrimmed(revision.SuccessCriteria)
	constraints := sortedTrimmed(revision.Constraints)
	nonGoals := sortedTrimmed(revision.NonGoals)
	checks := sortedTrimmed(unit.EvidenceChecks)
	stops := sortedTrimmed(unit.StopConditions)
	names := make([]string, 0, len(grants))
	for _, grant := range grants { names = append(names, grant.Name+"@"+grant.Scope) }
	sort.Strings(names)
	core := runBriefCore{
		ContractRevisionNumber: revision.Number,
		Goal: revision.Goal,
		SuccessCriteria: criteria,
		Review: revision.Review,
		Constraints: constraints,
		NonGoals: nonGoals,
		Clarification: revision.Clarification,
		WorkUnitTitle: unit.Title,
		WorkUnitProvider: string(unit.Provider),
		WorkUnitModelSelection: string(unit.ModelSelection),
		WorkUnitModel: unit.Model,
		WorkUnitOutput: unit.OutputSummary,
		EvidenceChecks: checks,
		VerificationRequirement: unit.VerificationRequirement,
		StopConditions: stops,
		Grants: names,
	}
	encoded, err := json.Marshal(core)
	if err != nil { return "", fmt.Errorf("encode run brief core: %w", err) }
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func sortedTrimmed(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v != "" { out = append(out, v) }
	}
	sort.Strings(out)
	return out
}
