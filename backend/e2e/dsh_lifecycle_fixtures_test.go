//go:build !windows

package e2e

// DeepSeek Harness lifecycle fixtures for issue #60, consuming the exported
// dshtest package so these scenarios stay importable and deterministic:
// no real dsh CLI, no model calls — a scripted fake binary stands in for the
// provider process while every canonical fact flows through the real daemon.

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/deepseekharness"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/deepseekharness/dshtest"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// Scenario (gap ②): the provider process dies mid-run. The exit surfaces as
// an observable FACT through launch/exec, while profile composition truth is
// unchanged before and after — Kennel never infers death beyond what the
// process reported.
func TestDSHMidRunLossSurfacesExitFactWithoutDeathInference(t *testing.T) {
	requireProofE2E(t)
	bin := dshtest.ScriptedBinary(t)
	bin.PrependToPATH(t)
	t.Setenv(dshtest.EnvExitCode, "1") // mid-run loss: launch exits nonzero

	plugin := &deepseekharness.Plugin{}
	cfg := ports.AgentConfig{Mode: "waldo-profile"}

	readyBefore, err := plugin.ProfileReadiness(context.Background(), cfg)
	if err != nil || !readyBefore.Ready {
		t.Fatalf("composition should be ready pre-launch: %+v err=%v", readyBefore, err)
	}

	argv, err := plugin.GetLaunchCommand(context.Background(), ports.LaunchConfig{Config: cfg})
	if err != nil {
		t.Fatalf("launch command: %v", err)
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	out, runErr := cmd.CombinedOutput()
	if runErr == nil {
		t.Fatalf("mid-run loss must surface a nonzero exit: output=%s", out)
	}
	if !strings.Contains(string(out), "session started") {
		t.Fatalf("expected startup banner before loss: %s", out)
	}

	readyAfter, err := plugin.ProfileReadiness(context.Background(), cfg)
	if err != nil || !readyAfter.Ready {
		t.Fatalf("composition truth must not inherit runtime death: %+v err=%v", readyAfter, err)
	}
}

// Scenario (required-capability-missing): starting a deepseek-harness attempt
// without a composed profile fails closed AT THE API BOUNDARY, before any
// durable attempt row exists — while plan proposal/approval stay available.
func TestDSHAttemptWithoutProfileFailsClosedBeforeAnyRow(t *testing.T) {
	requireProofE2E(t)
	bin := dshtest.ScriptedBinary(t)
	bin.PrependToPATH(t)

	d := startDaemon(t, t.TempDir())
	defer d.stop()

	projectID := seedProject(t, d, "dsh-noprofile")
	var created struct {
		Outcome struct {
			ID              string `json:"id"`
			CurrentRevision struct {
				ID     string `json:"id"`
				Number int64  `json:"number"`
			} `json:"currentRevision"`
		} `json:"outcome"`
	}
	d.mustCall("POST", "/projects/"+projectID+"/outcomes", 201, map[string]any{
		"title":           "Missing capability must refuse admission",
		"goal":            "One deterministic criterion.",
		"successCriteria": []string{"Ledger delivers."},
		"review":          "Deterministic check plus owner walkthrough.",
		"requestKey":      "seed-dsh-noprofile",
	}, &created)
	o := created.Outcome

	var planEnv struct {
		Plan struct {
			ID string `json:"id"`
		} `json:"plan"`
	}
	d.mustCall("POST", "/outcomes/"+o.ID+"/plans", 201,
		map[string]any{"expectedContractRevision": o.CurrentRevision.Number}, &planEnv)
	d.mustCall("POST", "/outcomes/"+o.ID+"/plans/"+planEnv.Plan.ID+"/approval", 200,
		map[string]any{"expectedContractRevision": o.CurrentRevision.Number}, nil)

	// No profile is configured anywhere: the spawn-identical gate must refuse.
	status, apiErr := d.callExpectingError("POST", "/outcomes/"+o.ID+"/attempts", map[string]any{
		"planRevisionId": planEnv.Plan.ID,
		"harness":        "deepseek-harness",
		"requestKey":     "attempt-refused",
	})
	if status != 409 || apiErr.Code != "AGENT_PROFILE_NOT_READY" {
		t.Fatalf("want 409 AGENT_PROFILE_NOT_READY, got %d %+v", status, apiErr)
	}
	if !strings.Contains(apiErr.Message+"|"+fmt.Sprint(apiErr.Details), "no dsh profile selected") {
		t.Fatalf("refusal should name the missing profile: %+v", apiErr)
	}

	var list struct {
		Attempts []json.RawMessage `json:"attempts"`
	}
	d.mustCall("GET", "/outcomes/"+o.ID+"/attempts", 200, nil, &list)
	if len(list.Attempts) != 0 {
		t.Fatalf("refused admission must not leave durable rows: %d", len(list.Attempts))
	}
}
