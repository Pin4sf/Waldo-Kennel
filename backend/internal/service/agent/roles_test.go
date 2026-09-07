package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/cursor"
	agentregistry "github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/registry"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
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
	ready            bool
	readyWithProfile bool
	captured         *ports.AgentConfig
}

func (f captureProfileAgent) ProfileReadiness(_ context.Context, cfg ports.AgentConfig) (ports.AgentProfileReadiness, error) {
	if f.captured != nil {
		*f.captured = cfg
	}
	ready := f.ready || f.readyWithProfile && cfg.Profile != ""
	return ports.AgentProfileReadiness{Ready: ready, Detail: "captured"}, nil
}

// TestServiceResolveMissionRolesUsesProjectConfigForReadiness proves the
// Project-config profile reaches the readiness gate: a worker override
// carrying AgentConfig.Profile flips the honored Cursor preference to
// ready, and the adapter observed exactly that profile.
func TestServiceResolveMissionRolesUsesProjectConfigForReadiness(t *testing.T) {
	captured := ports.AgentConfig{}
	plugin := captureProfileAgent{fakeAgent: fakeAgent{}, ready: false, captured: &captured}
	svc := NewWithAgents([]agentregistry.HarnessAgent{
		{Harness: domain.HarnessCursor, Manifest: adapters.Manifest{ID: "cursor"}, Agent: plugin},
	})
	cfg := domain.ProjectConfig{
		Worker: domain.RoleOverride{Harness: domain.HarnessCursor, AgentConfig: domain.AgentConfig{Profile: "waldo-profile"}},
	}
	roles := svc.ResolveMissionRoles(context.Background(),
		domain.ProjectAgentPreferences{DefaultWorker: "cursor"}, cfg)
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
		{Harness: domain.HarnessCursor, Manifest: adapters.Manifest{ID: "cursor"}, Agent: plugin},
	})
	cfg := domain.ProjectConfig{
		AgentConfig: domain.AgentConfig{Model: "shared-model", Mode: "medium", Profile: "shared-profile", Permissions: domain.PermissionModeAuto},
		Worker: domain.RoleOverride{Harness: domain.HarnessCodex, AgentConfig: domain.AgentConfig{
			Model: "codex-model", Mode: "high", Profile: "codex-side-profile",
		}},
	}
	roles := svc.ResolveMissionRoles(context.Background(),
		domain.ProjectAgentPreferences{DefaultWorker: "cursor"}, cfg)
	if got := captured.Profile; got != "" {
		t.Fatalf("readiness probed profile %q from a mismatched override; want empty", got)
	}
	if got := captured.Model; got != "" {
		t.Fatalf("readiness probed model %q from a mismatched override; want empty", got)
	}
	if got := captured.Mode; got != "" {
		t.Fatalf("readiness probed mode %q from a mismatched override; want empty", got)
	}
	if got := captured.Permissions; got != domain.PermissionModeAuto {
		t.Fatalf("readiness dropped provider-neutral permissions %q; want %q", got, domain.PermissionModeAuto)
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
	plugin := captureProfileAgent{fakeAgent: fakeAgent{}, ready: false, readyWithProfile: true, captured: &captured}
	svc := NewWithAgents([]agentregistry.HarnessAgent{
		{Harness: domain.HarnessCursor, Manifest: adapters.Manifest{ID: "cursor"}, Agent: plugin},
	})
	cfg := domain.ProjectConfig{
		AgentConfig: domain.AgentConfig{Model: "shared-model", Mode: "medium", Profile: "shared-profile", Permissions: domain.PermissionModeAuto},
		Worker: domain.RoleOverride{AgentConfig: domain.AgentConfig{
			Model: "orphan-model", Mode: "high", Profile: "orphan-profile",
		}},
	}
	roles := svc.ResolveMissionRoles(context.Background(),
		domain.ProjectAgentPreferences{DefaultWorker: "cursor"}, cfg)
	if got := captured.Profile; got != "" {
		t.Fatalf("unset-harness override leaked profile %q into readiness; want empty", got)
	}
	if got := captured.Model; got != "" {
		t.Fatalf("readiness probed model %q from an unset harness; want empty", got)
	}
	if got := captured.Mode; got != "" {
		t.Fatalf("readiness probed mode %q from an unset harness; want empty", got)
	}
	if got := captured.Permissions; got != domain.PermissionModeAuto {
		t.Fatalf("readiness dropped provider-neutral permissions %q; want %q", got, domain.PermissionModeAuto)
	}
	if roles.Worker.Ready {
		t.Fatalf("unset-harness shared profile must fail closed: %+v", roles.Worker)
	}
	if !strings.Contains(strings.ToLower(roles.Worker.Reason), "profile") {
		t.Fatalf("reason should name the missing-profile gate: %q", roles.Worker.Reason)
	}
}

// TestServiceResolveMissionRolesMatchingHarnessRetainsSharedProfile proves
// spawn parity: a shared profile is retained when the stored role harness
// matches the resolved Cursor Harness.
func TestServiceResolveMissionRolesMatchingHarnessRetainsSharedProfile(t *testing.T) {
	captured := ports.AgentConfig{}
	plugin := captureProfileAgent{fakeAgent: fakeAgent{}, ready: false, readyWithProfile: true, captured: &captured}
	svc := NewWithAgents([]agentregistry.HarnessAgent{
		{Harness: domain.HarnessCursor, Manifest: adapters.Manifest{ID: "cursor"}, Agent: plugin},
	})
	cfg := domain.ProjectConfig{
		AgentConfig: domain.AgentConfig{Model: "shared-model", Mode: "medium", Profile: "shared-profile", Permissions: domain.PermissionModeAuto},
		Worker:      domain.RoleOverride{Harness: domain.HarnessCursor},
	}
	roles := svc.ResolveMissionRoles(context.Background(),
		domain.ProjectAgentPreferences{DefaultWorker: "cursor"}, cfg)
	if got := captured.Profile; got != "shared-profile" {
		t.Fatalf("matching harness dropped shared profile %q; want shared-profile", got)
	}
	if got := captured.Model; got != "shared-model" {
		t.Fatalf("matching harness dropped shared model %q; want shared-model", got)
	}
	if got := captured.Mode; got != "medium" {
		t.Fatalf("matching harness dropped shared mode %q; want medium", got)
	}
	if got := captured.Permissions; got != domain.PermissionModeAuto {
		t.Fatalf("matching harness dropped provider-neutral permissions %q; want %q", got, domain.PermissionModeAuto)
	}
	if !roles.Worker.Ready {
		t.Fatalf("matching Cursor Harness with a shared profile must be ready: %+v", roles.Worker)
	}
}

func TestEnrichMissionRolesFailsClosedWithoutProfile(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectConfig{AgentPreferences: domain.ProjectAgentPreferences{DefaultWorker: "cursor"}})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessCursor: {Installed: true, RequiresProfile: true, ProfileReady: boolPtr(false), AuthApplicable: true, Auth: ports.AgentAuthStatusAuthorized},
	}
	got := EnrichMissionRoles(base, facts)
	if got.Worker.Harness != domain.HarnessCursor {
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
		base := domain.ResolveMissionRoles(domain.ProjectConfig{AgentPreferences: domain.ProjectAgentPreferences{}})
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
	base := domain.ResolveMissionRoles(domain.ProjectConfig{AgentPreferences: domain.ProjectAgentPreferences{DefaultWorker: "cursor"}})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessCursor: {Installed: true, RequiresProfile: true, ProfileReady: boolPtr(false), AuthApplicable: true, Auth: ports.AgentAuthStatusUnauthorized},
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

// TestEnrichMissionRolesCursorReadyAfterProfile proves the canonical unlock
// path: Cursor Harness deliberately does not implement the auth checker, so
// once it is installed and its profile is ready — exactly what the #60
// Profile contract will compose — the honored worker preference becomes ready.
func TestEnrichMissionRolesCursorReadyAfterProfile(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectConfig{AgentPreferences: domain.ProjectAgentPreferences{DefaultWorker: "cursor"}})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessCursor: {Installed: true, RequiresProfile: true, ProfileReady: boolPtr(true), AuthApplicable: false, Auth: ports.AgentAuthStatusUnknown},
	}
	got := EnrichMissionRoles(base, facts)
	if got.Worker.Harness != domain.HarnessCursor {
		t.Fatalf("preference harness must be preserved: %+v", got.Worker)
	}
	if !got.Worker.Ready {
		t.Fatalf("profile-ready Cursor worker must become ready: %+v", got.Worker)
	}
	lower := strings.ToLower(got.Worker.Reason)
	for _, blocker := range []string{"not installed", "profile", "authorization"} {
		if strings.Contains(lower, blocker) {
			t.Fatalf("ready worker must not carry a blocking reason: %q", got.Worker.Reason)
		}
	}
}

func TestEnrichMissionRolesMarksUninstalledHarnessNotReady(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectConfig{AgentPreferences: domain.ProjectAgentPreferences{}})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessCodex: {Installed: false, AuthApplicable: true, Auth: ports.AgentAuthStatusAuthorized},
	}
	got := EnrichMissionRoles(base, facts)
	if got.Coordinator.Harness != domain.HarnessCodex || got.Coordinator.Ready {
		t.Fatalf("uninstalled default should be not-ready, not substituted: %+v", got.Coordinator)
	}
}

func TestEnrichMissionRolesPassesReadyRolesThrough(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectConfig{AgentPreferences: domain.ProjectAgentPreferences{}})
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
		{Harness: domain.HarnessCursor, Manifest: adapters.Manifest{ID: "cursor"}, Agent: &cursor.Plugin{}},
		{Harness: domain.HarnessCodex, Manifest: adapters.Manifest{ID: "codex"}, Agent: fakeAuthAgent{fakeAgent: fakeAgent{}, status: ports.AgentAuthStatusAuthorized}},
	})
	roles := svc.ResolveMissionRoles(context.Background(), domain.ProjectAgentPreferences{DefaultWorker: "cursor"}, domain.ProjectConfig{})
	// The real dsh binary is absent on this machine's test PATH and no profile
	// is configured: the preference is honored but readiness fails closed.
	if roles.Worker.Harness != domain.HarnessCursor {
		t.Fatalf("preference harness must be preserved: %+v", roles.Worker)
	}
	if roles.Worker.Ready {
		t.Fatalf("cursor worker must fail closed without binary+profile: %+v", roles.Worker)
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
	prefs := domain.ProjectAgentPreferences{DefaultWorker: "cursor"}
	build := func(order []domain.AgentHarness) *Service {
		items := make([]agentregistry.HarnessAgent, 0, len(order))
		for _, harness := range order {
			items = append(items, agentregistry.HarnessAgent{
				Harness:  harness,
				Manifest: adapters.Manifest{ID: string(harness)},
				Agent:    fakeAuthAgent{fakeAgent: fakeAgent{}, status: ports.AgentAuthStatusAuthorized},
			})
		}
		// The real Cursor adapter carries its own profile gate; keep one in
		// the inventory so the worker role exercises the profile path too.
		return NewWithAgents(append(items, agentregistry.HarnessAgent{
			Harness:  domain.HarnessCursor,
			Manifest: adapters.Manifest{ID: "cursor"},
			Agent:    &cursor.Plugin{},
		}))
	}
	orderings := [][]domain.AgentHarness{
		{domain.HarnessCodex, domain.HarnessClaudeCode, domain.HarnessOpenCode},
		{domain.HarnessOpenCode, domain.HarnessCodex, domain.HarnessClaudeCode},
	}
	var baseline domain.ResolvedMissionRoles
	for i, order := range orderings {
		got := build(order).ResolveMissionRoles(context.Background(), prefs, domain.ProjectConfig{})
		if got.Coordinator.Harness != domain.HarnessCodex || got.Coordinator.Source != domain.RoleSourceDefault {
			t.Fatalf("ordering %d coordinator = %+v, want canonical codex/default", i, got.Coordinator)
		}
		if got.Worker.Harness != domain.HarnessCursor || got.Worker.Source != domain.RoleSourcePreference {
			t.Fatalf("ordering %d worker = %+v, want honored cursor-harness preference", i, got.Worker)
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

// An agent that runs without a provider grant must not be gated on finding one.
// opencode ships usable free models, so an install with no credential is
// working, and an inconclusive probe is its ordinary state rather than a
// missing precondition.
func TestEnrichMissionRolesOptionalAuthReadyWithoutGrant(t *testing.T) {
	for name, auth := range map[string]ports.AgentAuthStatus{
		"unknown":    ports.AgentAuthStatusUnknown,
		"authorized": ports.AgentAuthStatusAuthorized,
	} {
		base := domain.ResolveMissionRoles(domain.ProjectConfig{AgentPreferences: domain.ProjectAgentPreferences{DefaultWorker: "opencode"}})
		facts := map[domain.AgentHarness]RoleInventoryFact{
			domain.HarnessOpenCode: {Installed: true, AuthApplicable: true, AuthOptional: true, Auth: auth},
		}
		got := EnrichMissionRoles(base, facts)
		if got.Worker.Harness != domain.HarnessOpenCode {
			t.Fatalf("%s: preference harness must be preserved: %+v", name, got.Worker)
		}
		if !got.Worker.Ready {
			t.Fatalf("%s: optional-auth worker must be ready: %+v", name, got.Worker)
		}
		if strings.Contains(strings.ToLower(got.Worker.Reason), "authorization") {
			t.Fatalf("%s: reason must not name an authorization gate: %q", name, got.Worker.Reason)
		}
	}
}

// Optional auth softens only the inconclusive case. A probe that affirmatively
// reports a refused grant is evidence, not absence of it, so it still blocks.
func TestEnrichMissionRolesOptionalAuthStillBlocksRefusedGrant(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectConfig{AgentPreferences: domain.ProjectAgentPreferences{DefaultWorker: "opencode"}})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessOpenCode: {Installed: true, AuthApplicable: true, AuthOptional: true, Auth: ports.AgentAuthStatusUnauthorized},
	}
	got := EnrichMissionRoles(base, facts)
	if got.Worker.Ready {
		t.Fatalf("a refused grant must still block even when auth is optional: %+v", got.Worker)
	}
	if !strings.Contains(strings.ToLower(got.Worker.Reason), "authorization") {
		t.Fatalf("reason should name the authorization gate: %q", got.Worker.Reason)
	}
}

// Optional auth must not weaken an agent that genuinely needs a grant, and it
// must not bypass the other independent gates.
func TestEnrichMissionRolesOptionalAuthDoesNotWeakenOtherGates(t *testing.T) {
	base := domain.ResolveMissionRoles(domain.ProjectConfig{AgentPreferences: domain.ProjectAgentPreferences{DefaultWorker: "opencode"}})
	facts := map[domain.AgentHarness]RoleInventoryFact{
		domain.HarnessOpenCode: {Installed: false, AuthApplicable: true, AuthOptional: true, Auth: ports.AgentAuthStatusUnknown},
	}
	got := EnrichMissionRoles(base, facts)
	if got.Worker.Ready {
		t.Fatalf("an uninstalled optional-auth worker must fail closed: %+v", got.Worker)
	}
	if !strings.Contains(strings.ToLower(got.Worker.Reason), "not installed") {
		t.Fatalf("reason should name the install gate: %q", got.Worker.Reason)
	}
}

// The shipped opencode adapter must actually declare optional auth; without it
// the readiness gate above would still block a working install.
func TestOpenCodeAdapterDeclaresOptionalAuth(t *testing.T) {
	for _, item := range agentregistry.Harnessed() {
		if item.Harness != domain.HarnessOpenCode {
			continue
		}
		if !isAuthOptional(item) {
			t.Fatal("opencode adapter must declare AuthOptional: it runs without a provider grant")
		}
		return
	}
	t.Fatal("opencode adapter not found in the shipped registry")
}

// Agents that need a grant must keep the strict default: saying nothing about
// optional auth means an inconclusive probe still fails closed.
func TestCodexKeepsStrictAuthDefault(t *testing.T) {
	for _, item := range agentregistry.Harnessed() {
		if item.Harness != domain.HarnessCodex {
			continue
		}
		if isAuthOptional(item) {
			t.Fatal("codex must keep the strict auth default")
		}
		return
	}
	t.Fatal("codex adapter not found in the shipped registry")
}
