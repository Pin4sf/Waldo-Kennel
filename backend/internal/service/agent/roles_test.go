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

func (f fakeProfileAgent) ProfileReadiness(context.Context, ports.AgentConfig) (ports.AgentProfileReadiness, error) {
	return ports.AgentProfileReadiness{Ready: f.ready, Detail: f.detail}, nil
}

func TestEnrichMissionRolesFailsClosedWithoutProfile(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectAgentPreferences{DefaultWorker: "deepseek-harness"})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessDeepSeekHarness: {Installed: true, RequiresProfile: true, ProfileReady: boolPtr(false)},
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

func TestEnrichMissionRolesMarksUninstalledHarnessNotReady(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectAgentPreferences{})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessCodex: {Installed: false},
	}
	got := EnrichMissionRoles(base, facts)
	if got.Coordinator.Harness != domain.HarnessCodex || got.Coordinator.Ready {
		t.Fatalf("uninstalled default should be not-ready, not substituted: %+v", got.Coordinator)
	}
}

func TestEnrichMissionRolesPassesReadyRolesThrough(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectAgentPreferences{})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessCodex: {Installed: true},
	}
	got := EnrichMissionRoles(base, facts)
	for name, role := range map[string]domain.ResolvedAgentRole{
		"analyzer": got.Analyzer, "coordinator": got.Coordinator,
		"worker": got.Worker, "verifier": got.Verifier,
	} {
		if !role.Ready {
			t.Fatalf("%s should stay ready when installed with no profile gate: %+v", name, role)
		}
	}
}

func TestServiceResolveMissionRolesProbesLiveInventory(t *testing.T) {
	svc := NewWithAgents([]agentregistry.HarnessAgent{
		{Harness: domain.HarnessDeepSeekHarness, Manifest: adapters.Manifest{ID: "deepseek-harness"}, Agent: &deepseekharness.Plugin{}},
		{Harness: domain.HarnessCodex, Manifest: adapters.Manifest{ID: "codex"}, Agent: fakeAgent{}},
	})
	roles := svc.ResolveMissionRoles(domain.ProjectAgentPreferences{DefaultWorker: "deepseek-harness"})
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

func boolPtr(b bool) *bool { return &b }
