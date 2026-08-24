package specgen_test

import (
	"bytes"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apispec"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apispec/specgen"
)

// TestBuild_MatchesEmbedded is the drift guard: the committed (embedded)
// openapi.yaml must equal fresh Build() output. If this fails, run
// `go generate ./...` and commit the result.
func TestBuild_MatchesEmbedded(t *testing.T) {
	got, err := specgen.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	embedded := apispec.Default().YAML()
	if !bytes.Equal(normalizeYAML(got), normalizeYAML(embedded)) {
		t.Fatalf("embedded openapi.yaml is stale — run `go generate ./...` and commit.\n"+
			"len(fresh)=%d len(embedded)=%d", len(got), len(embedded))
	}
}

// TestBuild_HarnessEnumsMatchAdmissionPolicy pins the wire enums to the
// domain admission policy: fresh worker, delegation, and switch targets admit
// Codex and DeepSeek Harness, while reviewer surfaces stay Codex-only until a
// DeepSeek reviewer adapter exists.
func TestBuild_HarnessEnumsMatchAdmissionPolicy(t *testing.T) {
	got, err := specgen.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Enum []string `yaml:"enum"`
				} `yaml:"properties"`
			} `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(got, &doc); err != nil {
		t.Fatalf("parse generated OpenAPI: %v", err)
	}
	for schema, wantEnum := range map[string][]string{
		"SpawnSessionRequest": {"codex", "deepseek-harness"},
		// Switch targets require verified continuation identity, so only Codex
		// is admitted there even though DeepSeek is a selectable worker.
		"SwitchAgentRequest":        {"codex"},
		"DelegateTaskRequest":       {"codex", "deepseek-harness"},
		"SetSessionReviewerRequest": {"codex"},
		"TriggerReviewRequest":      {"codex"},
	} {
		field := map[string]string{
			"SpawnSessionRequest":       "harness",
			"SwitchAgentRequest":        "targetHarness",
			"SetSessionReviewerRequest": "harness",
			"DelegateTaskRequest":       "agent",
			"TriggerReviewRequest":      "harness",
		}[schema]
		gotEnum := doc.Components.Schemas[schema].Properties[field].Enum
		if !reflect.DeepEqual(gotEnum, wantEnum) {
			t.Fatalf("%s.%s enum = %v, want %v", schema, field, gotEnum, wantEnum)
		}
	}
}

// TestBuild_Deterministic guards against nondeterministic output (which would
// make the drift check flaky in CI).
func TestBuild_Deterministic(t *testing.T) {
	a, err := specgen.Build()
	if err != nil {
		t.Fatalf("Build #1: %v", err)
	}
	b, err := specgen.Build()
	if err != nil {
		t.Fatalf("Build #2: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("Build() is not deterministic across calls")
	}
}

func normalizeYAML(in []byte) []byte {
	return bytes.ReplaceAll(in, []byte("\r\n"), []byte("\n"))
}
