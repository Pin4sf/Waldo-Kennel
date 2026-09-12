//go:build !windows

package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAgentProcessSuperviseReportsNonzeroExitWithCapabilityAndPreservesOutput(t *testing.T) {
	t.Setenv("KENNEL_SUPERVISOR_CAPABILITY", "supervisor-token")
	cfg := setConfigEnv(t)
	srv, capture := activityServer(t, http.StatusOK, `{"ok":true}`)
	writeRunFileFor(t, cfg, srv)

	out, errOut, err := executeCLI(t, Deps{
		In:           strings.NewReader(""),
		ProcessAlive: func(int) bool { return true },
	}, "agent-process", "supervise", "--session", "kennel-7", "--launch", "launch-3", "--", "sh", "-c", `test -z "$KENNEL_SUPERVISOR_CAPABILITY" && printf supervised; exit 23`)
	if err != nil {
		t.Fatalf("supervise returned child exit as command failure: %v\nstderr=%s", err, errOut)
	}
	if out != "supervised" {
		t.Fatalf("stdout = %q, want supervised", out)
	}
	var req setActivityAPIRequest
	if err := json.Unmarshal([]byte(capture.body), &req); err != nil {
		t.Fatal(err)
	}
	if req.State != "exited" || req.Event != "process-exited" || req.LaunchID != "launch-3" {
		t.Fatalf("exit report = %+v", req)
	}
	if req.ProcessExit == nil || req.ProcessExit.ExitCode == nil || *req.ProcessExit.ExitCode != 23 || req.ProcessExit.Reason != "failed" {
		t.Fatalf("process exit = %+v, want code 23 failed", req.ProcessExit)
	}
	if capture.supervisorCapability != "supervisor-token" {
		t.Fatalf("supervisor capability header = %q", capture.supervisorCapability)
	}
}

func TestAgentProcessSuperviseReportsNaturalZeroExit(t *testing.T) {
	t.Setenv("KENNEL_SUPERVISOR_CAPABILITY", "supervisor-token")
	cfg := setConfigEnv(t)
	srv, capture := activityServer(t, http.StatusOK, `{"ok":true}`)
	writeRunFileFor(t, cfg, srv)

	_, _, err := executeCLI(t, Deps{
		In:           strings.NewReader(""),
		ProcessAlive: func(int) bool { return true },
	}, "agent-process", "supervise", "--session", "kennel-7", "--launch", "launch-3", "--", "true")
	if err != nil {
		t.Fatal(err)
	}
	var req setActivityAPIRequest
	if err := json.Unmarshal([]byte(capture.body), &req); err != nil {
		t.Fatal(err)
	}
	if req.ProcessExit == nil || req.ProcessExit.ExitCode == nil || *req.ProcessExit.ExitCode != 0 || req.ProcessExit.Reason != "exited" {
		t.Fatalf("process exit = %+v, want code 0 exited", req.ProcessExit)
	}
}

func TestAgentProcessSuperviseRejectsInvalidGeneration(t *testing.T) {
	_, _, err := executeCLI(t, Deps{}, "agent-process", "supervise", "--session", "kennel-7", "--launch", "../stale", "--", "true")
	if err == nil {
		t.Fatal("invalid launch id should be rejected before starting the child")
	}
}
