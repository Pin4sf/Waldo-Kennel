package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/adapters"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/deepseekharness"
	agentregistry "github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/registry"
	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// fakeProfileAgent is an installed adapter whose profile gate reports the
// given readiness, so enrichment can be tested without a real dsh binary.
type fakeProfileAgent struct {
	fakeAgent
	ready  bool
	detail string
}

// captureProfileAgent records the exact config the readiness probe received,
// proving Project-config profiles reach the adapter gate.
type captureProfileAgent struct {
	fakeAgent
	ready    bool
	captured *ports.AgentConfig
}

func (f captureProfileAgent) ProfileReadiness(_ context.Context, cfg ports.AgentConfig) (ports.AgentProfileReadiness, error) {
	if f.captured != nil {
		*f.captured = cfg
	}
	return ports.AgentProfileReadiness{Ready: f.ready, Detail: "captured"}, nil
}

// TestServiceResolveMissionRolesUsesProjectConfigForReadiness proves the
// Project-config profile reaches the readiness gate: a worker override
// carrying AgentConfig.Profile flips the honored DeepSeek preference to
// ready, and the adapter observed exactly that profile.
func TestServiceResolveMissionRolesUsesProjectConfigForReadiness(t *testing.T) {
	captured := ports.AgentConfig{}
	plugin := captureProfileAgent{fakeAgent: fakeAgent{}, ready: false, captured: &captured}
	svc := NewWithAgents([]agentregistry.HarnessAgent{
		{Harness: domain.HarnessDeepSeekHarness, Manifest: adapters.Manifest{ID: "deepseek-harness"}, Agent: plugin},
	})
	cfg := domain.ProjectConfig{
		Worker: domain.RoleOverride{Harness: domain.HarnessDeepSeekHarness, AgentConfig: domain.AgentConfig{Profile: "waldo-profile"}},
	}
	roles := svc.ResolveMissionRoles(context.Background(),
		domain.ProjectAgentPreferences{DefaultWorker: "deepseek-harness"}, cfg)
	if captured.Profile != "waldo-profile" {
		t.Fatalf("adapter observed profile %q, want waldo-profile from the role override", captured.Profile)
	}
	if roles.Worker.Ready {
		t.Fatalf("worker must stay not-ready when the gate refuses: %+v", roles.Worker)
	}
	if !strings.Contains(strings.ToLower(roles.Worker.Reason), "profile") {
		t.Fatalf("reason should carry the profile gate: %q", roles.Worker.Reason)
	}
}

func (f fakeProfileAgent) ProfileReadiness(context.Context, ports.AgentConfig) (ports.AgentProfileReadiness, error) {
	return ports.AgentProfileReadiness{Ready: f.ready, Detail: f.detail}, nil
}

// TestServiceResolveMissionRolesMismatchedOverrideHarnessIsCleared locks
// spawn parity: an override bound to a DIFFERENT harness than the resolved
// one contributes nothing — its Profile is cleared exactly as
// freshAgentConfig clears it at launch.
func TestServiceResolveMissionRolesMismatchedOverrideHarnessIsCleared(t *testing.T) {
	captured := ports.AgentConfig{}
	plugin := captureProfileAgent{fakeAgent: fakeAgent{}, ready: false, captured: &captured}
	svc := NewWithAgents([]agentregistry.HarnessAgent{
		{Harness: domain.HarnessDeepSeekHarness, Manifest: adapters.Manifest{ID: "deepseek-harness"}, Agent: plugin},
	})
	cfg := domain.ProjectConfig{
		Worker: domain.RoleOverride{Harness: domain.HarnessCodex, AgentConfig: domain.AgentConfig{Profile: "codex-side-profile"}},
	}
	roles := svc.ResolveMissionRoles(context.Background(),
		domain.ProjectAgentPreferences{DefaultWorker: "deepseek-harness"}, cfg)
	if got := captured.Profile; got != "" {
		t.Fatalf("readiness probed profile %q from a mismatched override; want empty", got)
	}
	if roles.Worker.Ready {
		t.Fatalf("worker must fail closed without an applicable profile: %+v", roles.Worker)
	}
	if !strings.Contains(strings.ToLower(roles.Worker.Reason), "profile") {
		t.Fatalf("reason should name the missing-profile gate: %q", roles.Worker.Reason)
	}
}

// TestServiceResolveMissionRolesEmptyOverrideHarnessIsCleared proves the
// unset-harness edge follows spawn: empty reads as mismatch and contributes
// nothing.
func TestServiceResolveMissionRolesEmptyOverrideHarnessIsCleared(t *testing.T) {
	captured := ports.AgentConfig{}
	plugin := captureProfileAgent{fakeAgent: fakeAgent{}, ready: true, captured: &captured}
	svc := NewWithAgents([]agentregistry.HarnessAgent{
		{Harness: domain.HarnessDeepSeekHarness, Manifest: adapters.Manifest{ID: "deepseek-harness"}, Agent: plugin},
	})
	cfg := domain.ProjectConfig{
		Worker: domain.RoleOverride{AgentConfig: domain.AgentConfig{Profile: "orphan-profile"}},
	}
	roles := svc.ResolveMissionRoles(context.Background(),
		domain.ProjectAgentPreferences{DefaultWorker: "deepseek-harness"}, cfg)
	if got := captured.Profile; got != "" {
		t.Fatalf("unset-harness override leaked profile %q into readiness; want empty", got)
	}
	if !roles.Worker.Ready {
		t.Fatalf("composing profile must keep the honored preference ready: %+v", roles.Worker)
	}
}

func TestEnrichMissionRolesFailsClosedWithoutProfile(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectAgentPreferences{DefaultWorker: "deepseek-harness"})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessDeepSeekHarness: {Installed: true, RequiresProfile: true, ProfileReady: boolPtr(false), AuthApplicable: true, Auth: ports.AgentAuthStatusAuthorized},
	}
	got := EnrichMissionRoles(base, facts)
	if got.Worker.Harness != domain.HarnessDeepSeekHarness {
		t.Fatalf("fail-closed must never silently fall back: %+v", got.Worker)
	}
	if got.Worker.Ready {
		t.Fatalf("worker must not be ready without a composed profile: %+v", got.Worker)
	}
	if !strings.Contains(strings.ToLower(got.Worker.Reason), "profile") {
		t.Fatalf("reason should name the profile gate: %q", got.Worker.Reason)
	}
}

func TestEnrichMissionRolesFailsClosedWhenAuthorizationNotGranted(t *testing.T) {
	for name, auth := range map[string]ports.AgentAuthStatus{
		"unauthorized": ports.AgentAuthStatusUnauthorized,
		"unknown":      ports.AgentAuthStatusUnknown,
	} {
		base := domain.ResolveMissionRoles(domain.ProjectAgentPreferences{})
		facts := map[domain.AgentHarness]RoleInventoryFact{
			domain.HarnessCodex: {Installed: true, AuthApplicable: true, Auth: auth},
		}
		got := EnrichMissionRoles(base, facts)
		if got.Coordinator.Ready {
			t.Fatalf("%s coordinator must fail closed: %+v", name, got.Coordinator)
		}
		if !strings.Contains(strings.ToLower(got.Coordinator.Reason), "authorization") {
			t.Fatalf("%s reason should name the authorization gate: %q", name, got.Coordinator.Reason)
		}
	}
}

func TestEnrichMissionRolesStacksIndependentBlockers(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectAgentPreferences{DefaultWorker: "deepseek-harness"})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessDeepSeekHarness: {Installed: true, RequiresProfile: true, ProfileReady: boolPtr(false), AuthApplicable: true, Auth: ports.AgentAuthStatusUnauthorized},
	}
	got := EnrichMissionRoles(base, facts)
	lower := strings.ToLower(got.Worker.Reason)
	if got.Worker.Ready {
		t.Fatalf("worker with two blockers must not be ready: %+v", got.Worker)
	}
	if !strings.Contains(lower, "profile") || !strings.Contains(lower, "authorization") {
		t.Fatalf("reason should name both independent gates: %q", got.Worker.Reason)
	}
}

// TestEnrichMissionRolesDeepSeekReadyAfterProfile proves the canonical unlock
// path: DeepSeek Harness deliberately does not implement the auth checker, so
// once it is installed and its profile is ready — exactly what the #60
// Profile contract will compose — the honored worker preference becomes ready.
func TestEnrichMissionRolesDeepSeekReadyAfterProfile(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectAgentPreferences{DefaultWorker: "deepseek-harness"})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessDeepSeekHarness: {Installed: true, RequiresProfile: true, ProfileReady: boolPtr(true), AuthApplicable: false, Auth: ports.AgentAuthStatusUnknown},
	}
	got := EnrichMissionRoles(base, facts)
	if got.Worker.Harness != domain.HarnessDeepSeekHarness {
		t.Fatalf("preference harness must be preserved: %+v", got.Worker)
	}
	if !got.Worker.Ready {
		t.Fatalf("profile-ready DeepSeek worker must become ready: %+v", got.Worker)
	}
	lower := strings.ToLower(got.Worker.Reason)
	for _, blocker := range []string{"not installed", "profile", "authorization"} {
		if strings.Contains(lower, blocker) {
			t.Fatalf("ready worker must not carry a blocking reason: %q", got.Worker.Reason)
		}
	}
}

func TestEnrichMissionRolesMarksUninstalledHarnessNotReady(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectAgentPreferences{})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessCodex: {Installed: false, AuthApplicable: true, Auth: ports.AgentAuthStatusAuthorized},
	}
	got := EnrichMissionRoles(base, facts)
	if got.Coordinator.Harness != domain.HarnessCodex || got.Coordinator.Ready {
		t.Fatalf("uninstalled default should be not-ready, not substituted: %+v", got.Coordinator)
	}
}

func TestEnrichMissionRolesPassesReadyRolesThrough(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectAgentPreferences{})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessCodex: {Installed: true, AuthApplicable: true, Auth: ports.AgentAuthStatusAuthorized},
	}
	got := EnrichMissionRoles(base, facts)
	for name, role := range map[string]domain.ResolvedAgentRole{
		"analyzer": got.Analyzer, "coordinator": got.Coordinator,
		"worker": got.Worker, "verifier": got.Verifier,
	} {
		if !role.Ready {
			t.Fatalf("%s should stay ready when installed and authorized with no profile gate: %+v", name, role)
		}
	}
}

func TestServiceResolveMissionRolesProbesLiveInventory(t *testing.T) {
	svc := NewWithAgents([]agentregistry.HarnessAgent{
		{Harness: domain.HarnessDeepSeekHarness, Manifest: adapters.Manifest{ID: "deepseek-harness"}, Agent: &deepseekharness.Plugin{}},
		{Harness: domain.HarnessCodex, Manifest: adapters.Manifest{ID: "codex"}, Agent: fakeAuthAgent{fakeAgent: fakeAgent{}, status: ports.AgentAuthStatusAuthorized}},
	})
	roles := svc.ResolveMissionRoles(context.Background(), domain.ProjectAgentPreferences{DefaultWorker: "deepseek-harness"}, domain.ProjectConfig{})
	// The real dsh binary is absent on this machine's test PATH and no profile
	// is configured: the preference is honored but readiness fails closed.
	if roles.Worker.Harness != domain.HarnessDeepSeekHarness {
		t.Fatalf("preference harness must be preserved: %+v", roles.Worker)
	}
	if roles.Worker.Ready {
		t.Fatalf("deepseek worker must fail closed without binary+profile: %+v", roles.Worker)
	}
	if !strings.Contains(strings.ToLower(roles.Worker.Reason), "not installed") &&
		!strings.Contains(strings.ToLower(roles.Worker.Reason), "profile") {
		t.Fatalf("reason should name the blocking gate: %q", roles.Worker.Reason)
	}
	if !roles.Coordinator.Ready {
		t.Fatalf("codex coordinator (installed, no profile gate) should be ready: %+v", roles.Coordinator)
	}
}

// TestResolveMissionRolesCoordinatorFallbackIgnoresCandidateOrder proves the
// canonical coordinator default is deterministic: with several admitted,
// authorized adapters in the inventory — probed in different orders — a
// stored non-coordinator preference still falls back to the same canonical
// default with identical proposals.
func TestResolveMissionRolesCoordinatorFallbackIgnoresCandidateOrder(t *testing.T) {
	prefs := domain.ProjectAgentPreferences{DefaultWorker: "deepseek-harness"}
	build := func(order []domain.AgentHarness) *Service {
		items := make([]agentregistry.HarnessAgent, 0, len(order))
		for _, harness := range order {
			items = append(items, agentregistry.HarnessAgent{
				Harness:  harness,
				Manifest: adapters.Manifest{ID: string(harness)},
				Agent:    fakeAuthAgent{fakeAgent: fakeAgent{}, status: ports.AgentAuthStatusAuthorized},
			})
		}
		// The real DeepSeek adapter carries its own profile gate; keep one in
		// the inventory so the worker role exercises the profile path too.
		return NewWithAgents(append(items, agentregistry.HarnessAgent{
			Harness:  domain.HarnessDeepSeekHarness,
			Manifest: adapters.Manifest{ID: "deepseek-harness"},
			Agent:    &deepseekharness.Plugin{},
		}))
	}
	orderings := [][]domain.AgentHarness{
		{domain.HarnessCodex, domain.HarnessClaudeCode, domain.HarnessAider},
		{domain.HarnessAider, domain.HarnessCodex, domain.HarnessClaudeCode},
	}
	var baseline domain.ResolvedMissionRoles
	for i, order := range orderings {
		got := build(order).ResolveMissionRoles(context.Background(), prefs, domain.ProjectConfig{})
		if got.Coordinator.Harness != domain.HarnessCodex || got.Coordinator.Source != domain.RoleSourceDefault {
			t.Fatalf("ordering %d coordinator = %+v, want canonical codex/default", i, got.Coordinator)
		}
		if got.Worker.Harness != domain.HarnessDeepSeekHarness || got.Worker.Source != domain.RoleSourcePreference {
			t.Fatalf("ordering %d worker = %+v, want honored deepseek-harness preference", i, got.Worker)
		}
		if i == 0 {
			baseline = got
			continue
		}
		if got != baseline {
			t.Fatalf("reordered candidates changed the proposal:\nfirst=%+v\nsecond=%+v", baseline, got)
		}
	}
}

func boolPtr(b bool) *bool { return &b }
