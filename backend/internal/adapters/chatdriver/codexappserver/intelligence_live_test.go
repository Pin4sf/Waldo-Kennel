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

// TestLiveCodexIntelligence drives the exact one-shot structured planning
// boundary used by intake and interactive planning. It is opt-in because it
// uses the owner's signed-in Codex installation and spends a real model turn.
func TestLiveCodexIntelligence(t *testing.T) {
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

	response, err := client.Complete(ctx, ports.LLMRequest{
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
	if err != nil {
		var failure *ports.ReasoningFailure
		if errors.As(err, &failure) {
			t.Fatalf("native intelligence: %v (cause: %v)", err, failure.Err)
		}
		t.Fatalf("native intelligence: %v", err)
	}
	var got struct {
		LineCount int `json:"lineCount"`
	}
	if err := json.Unmarshal(response.JSON, &got); err != nil || got.LineCount != 2 {
		t.Fatalf("native intelligence response = %s, err=%v", response.JSON, err)
	}
	if response.NativeSessionRef == "" || response.EffectiveModel == "" {
		t.Fatalf("missing native provenance: %+v", response)
	}
}
