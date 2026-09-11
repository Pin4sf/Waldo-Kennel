package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// TestLiveCodexPacketIntelligence proves the launchable native fallback: Codex
// reasons over the bounded packet while repository-tool conformance remains a
// separate, explicit live gate.
func TestLiveCodexPacketIntelligence(t *testing.T) {
	if os.Getenv("KENNEL_CODEX_LIVE") != "1" {
		t.Skip("set KENNEL_CODEX_LIVE=1 to run against a real codex app-server")
	}
	bin := os.Getenv("KENNEL_CODEX_BIN")
	if bin == "" {
		bin = "codex"
	}
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("codex binary %q not on PATH: %v", bin, err)
	}
	driver := New(livePlugin{bin: bin}, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))
	client := NewIntelligenceClient(driver, IntelligenceConfig{Timeout: 4 * time.Minute})
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	response, err := client.Complete(ctx, ports.LLMRequest{
		System: "Return the requested structured observation.", User: "The supplied fact is that the launch marker equals 7. Return that integer.",
		SchemaName: "live_packet_observation",
		Schema:     map[string]any{"type": "object", "additionalProperties": false, "required": []any{"marker"}, "properties": map[string]any{"marker": map[string]any{"type": "integer"}}},
	})
	if err != nil {
		t.Fatalf("native packet intelligence: %v", err)
	}
	var got struct {
		Marker int `json:"marker"`
	}
	if err := json.Unmarshal(response.JSON, &got); err != nil || got.Marker != 7 {
		t.Fatalf("native packet response = %s, err=%v", response.JSON, err)
	}
	if response.NativeSessionRef == "" || response.EffectiveModel == "" {
		t.Fatalf("missing native provenance: %+v", response)
	}
}

// TestLiveCodexRepositoryToolsRemainUnavailable keeps the optional live suite
// truthful: the packet path above is launchable, while native repository tools
// remain disabled until their permission and behavioral conformance is proven.
func TestLiveCodexRepositoryToolsRemainUnavailable(t *testing.T) {
	if os.Getenv("KENNEL_CODEX_LIVE") != "1" {
		t.Skip("set KENNEL_CODEX_LIVE=1 to run against a real codex app-server")
	}
	bin := os.Getenv("KENNEL_CODEX_BIN")
	if bin == "" {
		bin = "codex"
	}
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("codex binary %q not on PATH: %v", bin, err)
	}

	workspace := t.TempDir()
	seedGitWorkspace(t, workspace)
	driver := New(livePlugin{bin: bin}, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))
	client := NewIntelligenceClient(driver, IntelligenceConfig{Timeout: 4 * time.Minute})
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	_, err := client.Complete(ctx, ports.LLMRequest{
		System:     "Return the requested structured planning observation. Do not implement anything.",
		User:       "Inspect hello.txt and report its non-empty line count.",
		SchemaName: "live_planning_observation",
		Schema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []any{"lineCount"},
			"properties": map[string]any{
				"lineCount": map[string]any{"type": "integer"},
			},
		},
		ContextAccess: ports.ReasoningContextAccess{Mode: ports.ReasoningContextRepositoryRead, Root: workspace},
	})
	if err == nil {
		t.Fatal("expected native repository tools to remain unavailable")
	}
	var failure *ports.ReasoningFailure
	if !errors.As(err, &failure) || failure.Kind != ports.ReasoningUnavailable {
		t.Fatalf("repository tool failure = %v", err)
	}
}
