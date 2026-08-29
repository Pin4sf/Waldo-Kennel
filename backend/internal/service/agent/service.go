package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	agentregistry "github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/registry"
	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

var (
	agentInstallProbeTimeout = 2 * time.Second
	agentAuthProbeTimeout    = 10 * time.Second
	agentRefreshMinInterval  = 10 * time.Second
	modelCatalogLoadTimeout  = 30 * time.Second
	// How long a cached catalog is trusted before AO asks a cache-first client to
	// revalidate in the background. Long, because rediscovery runs an agent CLI:
	// this covers drift a fingerprint cannot see, not routine correctness.
	modelCatalogTrustWindow = 6 * time.Hour
)

// catalogNeedsRevalidation reports whether a cached catalog is old enough to
// re-check. A zero timestamp comes from a record written before validation was
// tracked, so it counts as due.
func catalogNeedsRevalidation(validatedAt time.Time) bool {
	return validatedAt.IsZero() || time.Since(validatedAt) > modelCatalogTrustWindow
}

type modelLoadMode uint8

const (
	modelLoadCached modelLoadMode = iota
	modelLoadRevalidate
	modelLoadRefresh
)

type modelCatalogCall struct {
	done    chan struct{}
	catalog ports.AgentModelCatalog
	err     error
}

type probeResult struct {
	info       Info
	installed  bool
	authorized bool
}

// ProbeResult describes a fresh readiness probe for one supported agent.
type ProbeResult struct {
	Agent     Info `json:"agent"`
	Supported bool `json:"supported"`
	Installed bool `json:"installed"`
}

// Info is the user-facing identity for an agent adapter.
type Info struct {
	ID         string                `json:"id"`
	Label      string                `json:"label"`
	AuthStatus ports.AgentAuthStatus `json:"authStatus,omitempty" enum:"authorized,unauthorized,unknown" description:"Advisory local auth probe result. authorized means a recent local probe passed; spawn remains the authoritative validation point."`
	// Ready is the advisory profile-readiness probe for adapters whose launch
	// depends on user-selected configuration beyond a resolved binary. Absent
	// means the probe does not apply to this adapter. Spawn remains the
	// authoritative validation point.
	Ready       *bool  `json:"ready,omitempty" description:"Advisory profile-readiness probe for adapters whose launch needs more than an installed binary. Absent means the probe does not apply."`
	ReadyDetail string `json:"readyDetail,omitempty" description:"Adapter-explained readiness context: what was verified or what is missing."`
	// RequiresProfile mirrors the adapter's readiness capability: launch depends
	// on user-selected configuration (e.g. a dsh profile), so UIs must collect
	// it before offering this agent. The daemon enforces the same check at
	// spawn; this flag exists so clients never duplicate that policy.
	RequiresProfile bool `json:"requiresProfile,omitempty" description:"Launch requires user-selected profile configuration beyond an installed binary."`
	// Roles are the daemon's authoritative role admission for this harness.
	Roles AgentRoles `json:"roles" description:"Role admission derived from daemon policy. Clients must not re-derive it from provider names."`
}

// AgentRoles reports which responsibilities a harness is admitted for.
type AgentRoles struct {
	Worker       bool `json:"worker"`
	Coordinator  bool `json:"coordinator"`
	SwitchTarget bool `json:"switchTarget"`
}

// Inventory describes all daemon-supported agents and best-effort local probe
// results. Installed/authorized entries are advisory snapshots and can be stale;
// session spawn is the authoritative validation point for binary availability,
// runtime prerequisites, and model-call readiness.
type Inventory struct {
	Supported  []Info `json:"supported" description:"Agents supported by this daemon build."`
	Installed  []Info `json:"installed" description:"Agents whose binary resolved during the latest best-effort local catalog probe."`
	Authorized []Info `json:"authorized" description:"Compatibility list of installed agents whose local auth probe recently returned authorized. Advisory and stale-prone; spawn may still fail."`
}

// Service reports supported agent adapters and best-effort local readiness
// probes. Catalog readiness is advisory UI metadata, not a spawn precheck.
type Service struct {
	agents      []agentregistry.HarnessAgent
	cache       ports.AgentModelCatalogCache
	discoverer  ports.AgentModelDiscoverer
	projects    ProjectLookup
	resolverMu  map[string]*sync.Mutex
	modelCallMu sync.Mutex
	modelCalls  map[string]*modelCatalogCall

	mu          sync.RWMutex
	inventory   Inventory
	lastRefresh time.Time
	refreshMu   sync.Mutex
}

// Deps contains optional durable dependencies for the agent catalog service.
type Deps struct {
	Cache      ports.AgentModelCatalogCache
	Discoverer ports.AgentModelDiscoverer
	Projects   ProjectLookup
}

// ProjectLookup resolves the registered working directory used for model
// discovery. The SQLite store satisfies this narrow read boundary.
type ProjectLookup interface {
	GetProject(ctx context.Context, id string) (domain.ProjectRecord, bool, error)
}

// New returns an agent inventory service backed by the daemon's shipped
// adapter registry.
func New() *Service {
	return NewWithDeps(Deps{})
}

// NewWithDeps returns the production service with durable model-catalog cache.
func NewWithDeps(deps Deps) *Service {
	return newService(agentregistry.Harnessed(), deps.Cache, deps.Projects, deps.Discoverer)
}

// NewWithAgents returns an inventory service over a caller-provided adapter
// slice. It is used by focused tests.
func NewWithAgents(agents []agentregistry.HarnessAgent) *Service {
	return newService(agents, nil, nil, nil)
}

func newService(agents []agentregistry.HarnessAgent, cache ports.AgentModelCatalogCache, projects ProjectLookup, discoverer ports.AgentModelDiscoverer) *Service {
	resolverMu := make(map[string]*sync.Mutex, len(agents))
	for _, item := range agents {
		resolverMu[string(item.Harness)] = &sync.Mutex{}
	}
	return &Service{agents: agents, cache: cache, discoverer: discoverer, projects: projects, resolverMu: resolverMu, modelCalls: map[string]*modelCatalogCall{}, inventory: Inventory{
		Supported:  supportedInfos(agents),
		Installed:  []Info{},
		Authorized: []Info{},
	}}
}

// List returns the cached agent inventory without running probes. Installed and
// authorized entries come from the last explicit Refresh call and are advisory:
// they can be stale by the time a user starts a session, and session spawn
// performs the authoritative binary/runtime validation.
func (s *Service) List(ctx context.Context) (Inventory, error) {
	if err := ctx.Err(); err != nil {
		return Inventory{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneInventory(s.inventory), nil
}

// Refresh runs the bounded local binary/auth probes, updates the cached
// inventory, and returns the new snapshot. Refreshes are serialized and
// rate-limited so repeated frontend reloads cannot stampede agent CLIs.
func (s *Service) Refresh(ctx context.Context) (Inventory, error) {
	if err := ctx.Err(); err != nil {
		return Inventory{}, err
	}
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	s.mu.RLock()
	if !s.lastRefresh.IsZero() && time.Since(s.lastRefresh) < agentRefreshMinInterval {
		cached := cloneInventory(s.inventory)
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	results := make(chan probeResult, len(s.agents))
	var wg sync.WaitGroup
	for _, item := range s.agents {
		if !item.Harness.IsSelectableForNewWork() {
			continue
		}
		if err := ctx.Err(); err != nil {
			return Inventory{}, err
		}
		wg.Add(1)
		go func(item agentregistry.HarnessAgent) {
			defer wg.Done()
			results <- s.probeAgent(ctx, item)
		}(item)
	}
	wg.Wait()
	close(results)

	supported := make([]Info, 0, len(s.agents))
	installed := make([]Info, 0, len(s.agents))
	authorized := make([]Info, 0, len(s.agents))
	for res := range results {
		supported = append(supported, res.info)
		if res.installed {
			installed = append(installed, res.info)
		}
		if res.authorized {
			authorized = append(authorized, res.info)
		}
	}
	sortInfos(supported)
	sortInfos(installed)
	sortInfos(authorized)
	next := Inventory{
		Supported:  supported,
		Installed:  installed,
		Authorized: authorized,
	}
	s.mu.Lock()
	s.inventory = cloneInventory(next)
	s.lastRefresh = time.Now()
	s.mu.Unlock()
	return next, nil
}

// Probe runs a fresh bounded binary/auth probe for one agent, bypassing the
// catalog refresh rate limit. It is intended for user-initiated preflight paths
// where a cached negative catalog result may be stale.
func (s *Service) Probe(ctx context.Context, agentID string) (ProbeResult, error) {
	if err := ctx.Err(); err != nil {
		return ProbeResult{}, err
	}
	for _, item := range s.agents {
		info := Info{ID: string(item.Harness), Label: item.Manifest.Name}
		if info.Label == "" {
			info.Label = info.ID
		}
		if info.ID != agentID {
			continue
		}
		res := s.probeAgent(ctx, item)
		return ProbeResult{
			Agent:     res.info,
			Supported: true,
			Installed: res.installed,
		}, nil
	}
	return ProbeResult{Agent: Info{ID: agentID}, Supported: false, Installed: false}, nil
}

// Models returns one normalized model catalog. Cached values survive daemon
// restarts; refresh forces a new documented CLI discovery attempt. Discovery
// failures degrade to the last cached catalog or a custom model input.
func (s *Service) Models(ctx context.Context, agentID, projectID string, refresh bool) (ports.AgentModelCatalog, error) {
	mode := modelLoadCached
	if refresh {
		mode = modelLoadRefresh
	}
	return s.coalesceModelLoad(ctx, agentID, projectID, mode)
}

// RevalidateModels applies the same installed-version check as the normal read
// path. It remains as a compatibility route for older clients.
func (s *Service) RevalidateModels(ctx context.Context, agentID, projectID string) (ports.AgentModelCatalog, error) {
	return s.coalesceModelLoad(ctx, agentID, projectID, modelLoadRevalidate)
}

func (s *Service) coalesceModelLoad(
	ctx context.Context,
	agentID, projectID string,
	mode modelLoadMode,
) (ports.AgentModelCatalog, error) {
	key := agentID + "\x00" + projectID + "\x00" + strconv.Itoa(int(mode))
	s.modelCallMu.Lock()
	if active := s.modelCalls[key]; active != nil {
		s.modelCallMu.Unlock()
		select {
		case <-active.done:
			return active.catalog, active.err
		case <-ctx.Done():
			return ports.AgentModelCatalog{}, ctx.Err()
		}
	}
	call := &modelCatalogCall{done: make(chan struct{})}
	s.modelCalls[key] = call
	s.modelCallMu.Unlock()

	loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), modelCatalogLoadTimeout)
	go func() {
		defer cancel()
		call.catalog, call.err = s.loadModels(loadCtx, agentID, projectID, mode)
		s.modelCallMu.Lock()
		delete(s.modelCalls, key)
		close(call.done)
		s.modelCallMu.Unlock()
	}()

	select {
	case <-call.done:
		return call.catalog, call.err
	case <-ctx.Done():
		return ports.AgentModelCatalog{}, ctx.Err()
	}
}

func (s *Service) loadModels(ctx context.Context, agentID, projectID string, mode modelLoadMode) (ports.AgentModelCatalog, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentModelCatalog{}, err
	}
	item, ok := s.agent(agentID)
	if !ok {
		return ports.AgentModelCatalog{}, apierr.NotFound("AGENT_NOT_FOUND", "Unknown agent adapter")
	}
	if s.discoverer == nil {
		return ports.AgentModelCatalog{}, apierr.Internal("MODEL_DISCOVERY_UNAVAILABLE", "Model discovery is unavailable")
	}
	discovery, err := s.projectDiscoveryContext(ctx, projectID)
	if err != nil {
		return ports.AgentModelCatalog{}, err
	}
	cached, hasCached, err := s.cachedCatalog(ctx, agentID, projectID)
	if err != nil {
		return ports.AgentModelCatalog{}, err
	}
	var binary string
	if resolver, ok := item.Agent.(ports.AgentBinaryResolver); ok {
		lock := s.resolverMu[agentID]
		lock.Lock()
		resolved, err := resolver.ResolveBinary(ctx)
		lock.Unlock()
		if err == nil {
			binary = resolved
		}
	}
	request := ports.AgentModelDiscoveryRequest{
		AgentID: agentID, Binary: binary, WorkingDir: discovery.workingDir, Env: discovery.env,
	}
	// Fingerprints the same inputs the discovery run would read, so a change to
	// either the executable or the configuration behind it invalidates the cache.
	version := s.discoverer.CatalogFingerprint(ctx, request)
	if hasCached && mode != modelLoadRefresh && cached.BinaryVersion == version {
		// A command-backed catalog can drift without the binary or its config
		// changing (a provider adds a model), which no fingerprint can see. Ask
		// cache-first clients to revalidate in the background once the catalog is
		// old enough, so staleness resolves itself instead of waiting for someone
		// to press a refresh button.
		cached.Catalog.RefreshRecommended = catalogNeedsRevalidation(cached.Catalog.ValidatedAt)
		return cached.Catalog, nil
	}

	discovered, discoverErr := s.discoverer.Discover(ctx, request)
	discovered.BinaryVersion = version
	discovered.ValidatedAt = time.Now().UTC()
	discovered.RefreshRecommended = false
	if discoverErr != nil {
		if hasCached && len(cached.Catalog.Models) > len(discovered.Models) {
			cached.Catalog.Stale = true
			cached.Catalog.Warning = discoverErr.Error()
			cached.Catalog.ValidatedAt = time.Now().UTC()
			cached.Catalog.RefreshRecommended = false
			if err := s.saveCatalog(ctx, projectID, cached.Catalog); err != nil {
				cached.Catalog.Warning = appendCacheWarning(cached.Catalog.Warning)
			}
			return cached.Catalog, nil
		}
		if len(discovered.Models) > 0 {
			discovered.Stale = true
			discovered.Warning = discoverErr.Error()
			if err := s.saveCatalog(ctx, projectID, discovered); err != nil {
				discovered.Warning = appendCacheWarning(discovered.Warning)
			}
			return discovered, nil
		}
		if hasCached {
			cached.Catalog.Stale = true
			cached.Catalog.Warning = discoverErr.Error()
			cached.Catalog.ValidatedAt = time.Now().UTC()
			cached.Catalog.RefreshRecommended = false
			if err := s.saveCatalog(ctx, projectID, cached.Catalog); err != nil {
				cached.Catalog.Warning = appendCacheWarning(cached.Catalog.Warning)
			}
			return cached.Catalog, nil
		}
		fallback := s.discoverer.Manual(agentID)
		fallback.BinaryVersion = version
		fallback.ValidatedAt = time.Now().UTC()
		fallback.Stale = true
		fallback.Warning = discoverErr.Error()
		return fallback, nil
	}
	if err := s.saveCatalog(ctx, projectID, discovered); err != nil {
		discovered.Warning = appendCacheWarning(discovered.Warning)
	}
	return discovered, nil
}

func appendCacheWarning(current string) string {
	const next = "Models loaded, but AO could not update the model cache."
	if current == "" {
		return next
	}
	return current + " " + next
}

type projectDiscovery struct {
	workingDir string
	env        map[string]string
}

func (s *Service) projectDiscoveryContext(ctx context.Context, projectID string) (projectDiscovery, error) {
	if projectID == "" || s.projects == nil {
		return projectDiscovery{}, nil
	}
	project, ok, err := s.projects.GetProject(ctx, projectID)
	if err != nil {
		return projectDiscovery{}, apierr.Internal("PROJECT_LOAD_FAILED", "Failed to load project")
	}
	if !ok || !project.ArchivedAt.IsZero() {
		return projectDiscovery{}, apierr.NotFound("PROJECT_NOT_FOUND", "Unknown project")
	}
	return projectDiscovery{workingDir: project.Path, env: project.Config.Env}, nil
}

type decodedCatalog struct {
	Catalog       ports.AgentModelCatalog
	BinaryVersion string
}

func (s *Service) cachedCatalog(ctx context.Context, agentID, projectID string) (decodedCatalog, bool, error) {
	if s.cache == nil {
		return decodedCatalog{}, false, nil
	}
	record, ok, err := s.cache.GetAgentModelCatalog(ctx, agentID, projectID)
	if err != nil || !ok {
		return decodedCatalog{}, ok, err
	}
	var catalog ports.AgentModelCatalog
	if err := json.Unmarshal([]byte(record.CatalogJSON), &catalog); err != nil {
		return decodedCatalog{}, false, fmt.Errorf("decode cached model catalog for %s: %w", agentID, err)
	}
	if catalog.Models == nil {
		catalog.Models = []ports.AgentModelInfo{}
	}
	return decodedCatalog{Catalog: catalog, BinaryVersion: record.BinaryVersion}, true, nil
}

func (s *Service) saveCatalog(ctx context.Context, projectID string, catalog ports.AgentModelCatalog) error {
	if s.cache == nil {
		return nil
	}
	data, err := json.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("encode model catalog for %s: %w", catalog.AgentID, err)
	}
	return s.cache.UpsertAgentModelCatalog(ctx, ports.CachedAgentModelCatalog{
		AgentID:       catalog.AgentID,
		ProjectID:     projectID,
		BinaryVersion: catalog.BinaryVersion,
		CatalogJSON:   string(data),
		Source:        catalog.Source,
		FetchedAt:     catalog.FetchedAt,
	})
}

func (s *Service) agent(agentID string) (agentregistry.HarnessAgent, bool) {
	for _, item := range s.agents {
		if string(item.Harness) == agentID {
			return item, true
		}
	}
	return agentregistry.HarnessAgent{}, false
}

func supportedInfos(agents []agentregistry.HarnessAgent) []Info {
	supported := make([]Info, 0, len(agents))
	for _, item := range agents {
		if !item.Harness.IsSelectableForNewWork() {
			continue
		}
		info := Info{ID: string(item.Harness), Label: item.Manifest.Name}
		if info.Label == "" {
			info.Label = info.ID
		}
		supported = append(supported, info)
	}
	sortInfos(supported)
	return supported
}

func cloneInventory(in Inventory) Inventory {
	return Inventory{
		Supported:  cloneInfos(in.Supported),
		Installed:  cloneInfos(in.Installed),
		Authorized: cloneInfos(in.Authorized),
	}
}

func cloneInfos(in []Info) []Info {
	out := make([]Info, len(in))
	copy(out, in)
	return out
}

func (s *Service) probeAgent(ctx context.Context, item agentregistry.HarnessAgent) probeResult {
	info := Info{ID: string(item.Harness), Label: item.Manifest.Name}
	if info.Label == "" {
		info.Label = info.ID
	}
	info.Roles = agentRoles(item.Harness)
	if _, ok := item.Agent.(ports.AgentProfileReadinessChecker); ok {
		info.RequiresProfile = true
	}
	probeCtx, cancel := context.WithTimeout(ctx, agentInstallProbeTimeout)
	defer cancel()
	resolver, ok := item.Agent.(ports.AgentBinaryResolver)
	if !ok {
		return probeResult{info: info}
	}
	lock := s.resolverMu[info.ID]
	lock.Lock()
	defer lock.Unlock()
	if _, err := resolver.ResolveBinary(probeCtx); err != nil {
		return probeResult{info: info}
	}
	authCtx, authCancel := context.WithTimeout(ctx, agentAuthProbeTimeout)
	defer authCancel()
	info.AuthStatus = authStatus(authCtx, item.Agent)
	// Profile readiness is advisory and probed with an empty config: it can
	// only describe the no-selection baseline ("what is missing"). A project's
	// configured profile is validated authoritatively at spawn.
	if checker, ok := item.Agent.(ports.AgentProfileReadinessChecker); ok {
		readinessCtx, readinessCancel := context.WithTimeout(ctx, agentInstallProbeTimeout)
		defer readinessCancel()
		if readiness, err := checker.ProfileReadiness(readinessCtx, ports.AgentConfig{}); err == nil {
			ready := readiness.Ready
			info.Ready = &ready
			info.ReadyDetail = readiness.Detail
		}
	}
	return probeResult{info: info, installed: true, authorized: info.AuthStatus == ports.AgentAuthStatusAuthorized}
}

// agentRoles derives the authoritative role admission for a harness from the
// domain predicates, so API consumers never re-implement provider policy.
func agentRoles(harness domain.AgentHarness) AgentRoles {
	return AgentRoles{
		Worker:       harness.IsSelectableForNewWork(),
		Coordinator:  harness.IsSelectableAsCoordinator(),
		SwitchTarget: harness.IsSelectableAsSwitchTarget(),
	}
}

func authStatus(ctx context.Context, a ports.Agent) ports.AgentAuthStatus {
	checker, ok := a.(ports.AgentAuthChecker)
	if !ok {
		return ports.AgentAuthStatusUnknown
	}
	status, err := checker.AuthStatus(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return ports.AgentAuthStatusUnknown
		}
		return ports.AgentAuthStatusUnknown
	}
	switch status {
	case ports.AgentAuthStatusAuthorized, ports.AgentAuthStatusUnauthorized:
		return status
	default:
		return ports.AgentAuthStatusUnknown
	}
}

func sortInfos(infos []Info) {
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ID < infos[j].ID
	})
}

// RoleInventoryFact is the live adapter admission truth for one harness,
// probed from the same inventory primitive the agents list serves. Auth is
// part of admission ONLY where the adapter implements the auth checker: an
// adapter without one (DeepSeek Harness deliberately does not) signals its
// readiness through installation and profile facts instead, so a missing
// probe must never read as a refused grant.
type RoleInventoryFact struct {
	Installed       bool
	RequiresProfile bool
	ProfileReady    *bool
	// AuthApplicable reports whether this adapter participates in authorization.
	AuthApplicable bool
	// AuthOptional reports whether the adapter can work without a grant, so an
	// inconclusive probe is its healthy state rather than a missing
	// precondition. Meaningful only when AuthApplicable.
	AuthOptional bool
	// Auth is the probed grant status; meaningful only when AuthApplicable.
	Auth ports.AgentAuthStatus
}

func (s *Service) agentFor(harness domain.AgentHarness) (agentregistry.HarnessAgent, bool) {
	for _, item := range s.agents {
		if item.Harness == harness {
			return item, true
		}
	}
	return agentregistry.HarnessAgent{}, false
}

// InventoryRoleFacts probes installation, authorization, and advisory profile
// readiness for each named harness through probeAgent — the same primitive
// that serves the agents catalog — so both paths report identical truth. The
// caller's context bounds every probe; child timeouts derive from it.
func (s *Service) InventoryRoleFacts(ctx context.Context, harnesses []domain.AgentHarness) map[domain.AgentHarness]RoleInventoryFact {
	out := make(map[domain.AgentHarness]RoleInventoryFact, len(harnesses))
	for _, harness := range harnesses {
		fact := RoleInventoryFact{Auth: ports.AgentAuthStatusUnknown}
		item, ok := s.agentFor(harness)
		if ok {
			fact.AuthApplicable = isAuthApplicable(item)
			fact.AuthOptional = isAuthOptional(item)
			res := s.probeAgent(ctx, item)
			fact.Installed = res.installed
			fact.RequiresProfile = res.info.RequiresProfile
			fact.ProfileReady = res.info.Ready
			if res.info.AuthStatus != "" {
				fact.Auth = res.info.AuthStatus
			}
		}
		out[harness] = fact
	}
	return out
}

// isAuthApplicable reports whether the adapter participates in authorization:
// adapters without an auth checker (DeepSeek Harness by design) are exempt,
// and their readiness is installation + profile only.
func isAuthApplicable(item agentregistry.HarnessAgent) bool {
	_, ok := item.Agent.(ports.AgentAuthChecker)
	return ok
}

// isAuthOptional reports whether the adapter declares that a provider grant is
// information rather than a precondition (opencode, which ships usable free
// models). Adapters that say nothing keep the strict default.
func isAuthOptional(item agentregistry.HarnessAgent) bool {
	reporter, ok := item.Agent.(ports.AgentOptionalAuth)
	return ok && reporter.AuthOptional()
}

// authBlocksReadiness decides whether a probed grant status should stop a role
// from being ready. An affirmative refusal always blocks. An inconclusive probe
// blocks only for adapters that need a grant to run at all: for one that
// declares auth optional, "no credential found" is its working state, and
// treating that as refusal would gate an agent that runs fine without one.
func authBlocksReadiness(fact RoleInventoryFact) bool {
	if fact.Auth == ports.AgentAuthStatusUnauthorized {
		return true
	}
	if fact.AuthOptional {
		return false
	}
	return fact.Auth != ports.AgentAuthStatusAuthorized
}

// EnrichMissionRoles layers live inventory truth onto the pure capability
// proposal. Readiness failures never substitute another harness — they flip
// Ready to false and name every blocking gate, so callers fail closed instead
// of silently falling back. Authorization fails closed only where the gate
// applies: an adapter that does not implement the auth checker (DeepSeek
// Harness) proves readiness through installation and profile facts instead,
// while a probed unknown or refused grant blocks.
func EnrichMissionRoles(base domain.ResolvedMissionRoles, facts map[domain.AgentHarness]RoleInventoryFact) domain.ResolvedMissionRoles {
	enrich := func(role domain.ResolvedAgentRole) domain.ResolvedAgentRole {
		fact, known := facts[role.Harness]
		if !known {
			return role
		}
		if !fact.Installed {
			role.Ready = false
			role.Reason += "; harness is not installed"
		}
		if fact.RequiresProfile && (fact.ProfileReady == nil || !*fact.ProfileReady) {
			role.Ready = false
			role.Reason += "; profile readiness fails closed (no composed profile)"
		}
		if fact.AuthApplicable && authBlocksReadiness(fact) {
			role.Ready = false
			role.Reason += "; agent authorization is not granted"
		}
		return role
	}
	base.Worker = enrich(base.Worker)
	base.Analyzer = enrich(base.Analyzer)
	base.Coordinator = enrich(base.Coordinator)
	base.Verifier = enrich(base.Verifier)
	return base
}

// ResolveMissionRoles combines stored preferences with live inventory truth
// into one daemon-resolved role proposal. Assignments are proposals for
// future Missions; historical sessions and approved Plans keep their
// immutable provider identity regardless of what this returns.
func (s *Service) ResolveMissionRoles(ctx context.Context, prefs domain.ProjectAgentPreferences, cfg domain.ProjectConfig) domain.ResolvedMissionRoles {
	base := domain.ResolveMissionRoles(prefs)
	facts := s.InventoryRoleFacts(ctx, s.uniqueHarnesses(base))
	// Profile readiness is Project-config-aware: each role is probed with the
	// AgentConfig that launch would actually merge (role override when set,
	// otherwise the shared base), so a persisted profile flips readiness here
	// instead of the UI reporting "no profile selected" after a save.
	for _, role := range []struct {
		harness         domain.AgentHarness
		overrideHarness domain.AgentHarness
		override        domain.AgentConfig
	}{
		{base.Worker.Harness, cfg.Worker.Harness, cfg.Worker.AgentConfig},
		{base.Analyzer.Harness, cfg.Orchestrator.Harness, cfg.Orchestrator.AgentConfig},
		{base.Coordinator.Harness, cfg.Orchestrator.Harness, cfg.Orchestrator.AgentConfig},
		{base.Verifier.Harness, cfg.Orchestrator.Harness, cfg.Orchestrator.AgentConfig},
	} {
		fact := facts[role.harness]
		if !fact.RequiresProfile || fact.ProfileReady != nil && *fact.ProfileReady {
			continue
		}
		item, ok := s.agentFor(role.harness)
		if !ok {
			continue
		}
		checker, ok := item.Agent.(ports.AgentProfileReadinessChecker)
		if !ok {
			continue
		}
		// Spawn clears provider-owned fields after merging the role override when
		// the stored role harness does not match the launch harness
		// (session_manager.freshAgentConfig). Readiness applies the identical
		// rule so the two can never disagree.
		override := role.override
		applies := overrideAppliesTo(role.overrideHarness, role.harness)
		if !applies {
			override = domain.AgentConfig{}
		}
		probeConfig := roleConfig(cfg, override)
		if !applies {
			probeConfig.Model = ""
			probeConfig.Mode = ""
			probeConfig.Profile = ""
		}
		probeCtx, cancel := context.WithTimeout(ctx, agentInstallProbeTimeout)
		readiness, err := checker.ProfileReadiness(probeCtx, probeConfig)
		cancel()
		if err != nil {
			continue
		}
		ready := readiness.Ready
		fact.ProfileReady = &ready
		facts[role.harness] = fact
	}
	return EnrichMissionRoles(base, facts)
}

func (s *Service) uniqueHarnesses(base domain.ResolvedMissionRoles) []domain.AgentHarness {
	seen := map[domain.AgentHarness]struct{}{
		base.Analyzer.Harness:    {},
		base.Coordinator.Harness: {},
		base.Worker.Harness:      {},
		base.Verifier.Harness:    {},
	}
	names := make([]domain.AgentHarness, 0, len(seen))
	for harness := range seen {
		names = append(names, harness)
	}
	return names
}

// overrideAppliesTo mirrors session_manager's freshAgentConfig rule exactly:
// an override whose harness differs from the resolved launch harness —
// including an UNSET harness — is cleared before launch, so it must be
// treated as cleared here too. Only an explicit harness match carries the
// override's Model/Mode/Profile forward.
func overrideAppliesTo(overrideHarness, resolved domain.AgentHarness) bool {
	return overrideHarness != "" && overrideHarness == resolved
}

// roleConfig merges a role override over the shared base; set fields win.
func roleConfig(cfg domain.ProjectConfig, override domain.AgentConfig) domain.AgentConfig {
	if override.Profile != "" {
		cfg.AgentConfig.Profile = override.Profile
	}
	if override.Model != "" {
		cfg.AgentConfig.Model = override.Model
	}
	if override.Mode != "" {
		cfg.AgentConfig.Mode = override.Mode
	}
	if override.Permissions != "" {
		cfg.AgentConfig.Permissions = override.Permissions
	}
	return cfg.AgentConfig
}
