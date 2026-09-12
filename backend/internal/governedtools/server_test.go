package governedtools

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/governedcheck"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

func testPolicy(write, exec bool) domain.AttemptExecutionPolicy {
	names := []string{domain.CapabilityWorktreeRead}
	grants := []domain.CapabilityGrant{{ID: "read", Name: domain.CapabilityWorktreeRead, Scope: "worktree/*"}}
	var checks []domain.ApprovedCheck
	if write {
		names = append(names, domain.CapabilityWorktreeWrite)
		grants = append(grants, domain.CapabilityGrant{ID: "write", Name: domain.CapabilityWorktreeWrite, Scope: "worktree/*"})
	}
	if exec {
		names = []string{domain.CapabilityWorktreeExec, domain.CapabilityWorktreeRead, domain.CapabilityWorktreeWrite}
		grants = []domain.CapabilityGrant{{ID: "exec", Name: domain.CapabilityWorktreeExec, Scope: "worktree/*"}, {ID: "read", Name: domain.CapabilityWorktreeRead, Scope: "worktree/*"}, {ID: "write", Name: domain.CapabilityWorktreeWrite, Scope: "worktree/*"}}
		checks = []domain.ApprovedCheck{{ID: "check-1", CriterionID: "criterion-1", Argv: []string{"go", "test", "./..."}, TimeoutSeconds: 30}}
	}
	return domain.AttemptExecutionPolicy{OutcomeID: "out-1", PlanRevisionID: "plan-1", WorkUnitID: "wu-1", ContractRevisionNumber: 1, RunBriefCoreDigest: "brief", RequiredCapabilities: names, Grants: grants, ApprovedChecks: checks}
}

func testServer(t *testing.T, root string, write, exec bool) Server {
	t.Helper()
	policy, err := testPolicy(write, exec).BindWorkspaceRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	server := Server{Policy: policy, WorkspaceRoot: root, root: opened}
	if exec {
		server.SessionID = "test-session"
		store := &recordingUncertaintySink{}
		server.UncertaintySink = store
		server.UncertaintySource = store
	}
	return server
}

func TestRepositoryToolsEnforceReadWriteAndLeaseBoundary(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("evidence"), 0o644); err != nil {
		t.Fatal(err)
	}
	readOnly := testServer(t, root, false, false)
	if got, err := readOnly.call(context.Background(), "read_text_file", map[string]interface{}{"path": "source.txt"}); err != nil || got != "evidence" {
		t.Fatalf("read = %q, %v", got, err)
	}
	if _, err := readOnly.call(context.Background(), "write_text_file", map[string]interface{}{"path": "report.md", "content": "no"}); err == nil {
		t.Fatal("read-only policy allowed write")
	}
	if _, err := readOnly.call(context.Background(), "read_text_file", map[string]interface{}{"path": "../outside"}); err == nil {
		t.Fatal("reader escaped workspace")
	}
	writer := testServer(t, root, true, false)
	if _, err := writer.call(context.Background(), "write_text_file", map[string]interface{}{"path": "report.md", "content": "bounded"}); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(root, "report.md")); err != nil || string(b) != "bounded" {
		t.Fatalf("report = %q, %v", b, err)
	}
}

func TestApprovedCheckToolExecutesOnlyFrozenVector(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	server := testServer(t, root, true, true)
	server.RunCheck = func(_ context.Context, req governedcheck.Request) (governedcheck.Result, error) {
		got = append([]string(nil), req.Argv...)
		return governedcheck.Result{ExitCode: 0, EnforcedBy: "test-fence", Output: "PASS"}, nil
	}
	text, err := server.call(context.Background(), "run_approved_check", map[string]interface{}{"check_id": "check-1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "go test ./..." || !strings.Contains(text, "enforced_by=test-fence") {
		t.Fatalf("vector=%q output=%q", got, text)
	}
	if _, err := server.call(context.Background(), "run_approved_check", map[string]interface{}{"check_id": "other"}); err == nil {
		t.Fatal("unapproved check executed")
	}
}

func TestRepositoryToolsProtectGitCustodyAndSymlinkBoundary(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	server := testServer(t, root, true, false)
	for _, path := range []string{".git", ".git/config", "nested/.git/config", ".GIT/config", "nested/.GiT/config"} {
		if _, err := server.call(context.Background(), "write_text_file", map[string]interface{}{"path": path, "content": "corrupt"}); err == nil {
			t.Fatalf("write to custody path %q succeeded", path)
		}
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := server.call(context.Background(), "write_text_file", map[string]interface{}{"path": "escape/pwned", "content": "no"}); err == nil {
		t.Fatal("write followed an escaping directory symlink")
	}
	if _, err := os.Stat(filepath.Join(outside, "pwned")); !os.IsNotExist(err) {
		t.Fatalf("outside file exists after refused write: %v", err)
	}
}

func TestRepositoryToolsDoNotWidenWriteOnlyPolicyIntoRead(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	policy, err := (domain.AttemptExecutionPolicy{
		OutcomeID: "out-1", PlanRevisionID: "plan-1", WorkUnitID: "wu-1", ContractRevisionNumber: 1,
		RunBriefCoreDigest: "brief", RequiredCapabilities: []string{domain.CapabilityWorktreeWrite},
		Grants: []domain.CapabilityGrant{{ID: "write", Name: domain.CapabilityWorktreeWrite, Scope: "worktree/*"}},
	}).BindWorkspaceRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	server := Server{Policy: policy, WorkspaceRoot: root, root: opened}
	for _, advertised := range server.tools() {
		name, _ := advertised["name"].(string)
		if name == "list_repository" || name == "read_text_file" {
			t.Fatalf("write-only policy advertised %q", name)
		}
	}
	if _, err := server.call(context.Background(), "read_text_file", map[string]interface{}{"path": "source.txt"}); err == nil {
		t.Fatal("write-only policy allowed direct read invocation")
	}
	if _, err := server.call(context.Background(), "write_text_file", map[string]interface{}{"path": "report.md", "content": "bounded"}); err != nil {
		t.Fatalf("write-only policy lost approved write: %v", err)
	}
}

func TestRepositoryWriteResistsConcurrentDirectorySymlinkSwap(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	server := testServer(t, root, true, false)
	dir := filepath.Join(root, "changing")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 300; i++ {
			_ = os.RemoveAll(dir)
			_ = os.Symlink(outside, dir)
			_ = os.Remove(dir)
			_ = os.Mkdir(dir, 0o755)
		}
	}()
	for i := 0; i < 300; i++ {
		_, _ = server.call(context.Background(), "write_text_file", map[string]interface{}{"path": "changing/pwned", "content": "bounded"})
	}
	<-done
	if _, err := os.Stat(filepath.Join(outside, "pwned")); !os.IsNotExist(err) {
		t.Fatalf("symlink swap redirected write outside root: %v", err)
	}
}

func TestRepositoryWritePreservesExistingExecutableMode(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "verify.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "--quiet"}, {"add", "verify.sh"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	server := testServer(t, root, true, false)
	if _, err := server.call(context.Background(), "write_text_file", map[string]interface{}{
		"path": "verify.sh", "content": "#!/bin/sh\nexit 0\n",
	}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("mode after atomic overwrite = %04o, want 0755", got)
	}
	cmd := exec.Command("git", "diff", "--summary", "--", "verify.sh")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git diff mode check: %v: %s", err, output)
	} else if strings.Contains(string(output), "mode change") {
		t.Fatalf("atomic overwrite changed Git executable mode: %s", output)
	}
}

type recordingUncertaintySink struct {
	facts   []ports.GovernedCheckUncertainty
	clears  []domain.SessionID
	current *ports.GovernedCheckUncertainty
}

func (s *recordingUncertaintySink) ClearGovernedCheckUncertainty(_ context.Context, sessionID domain.SessionID) error {
	s.clears = append(s.clears, sessionID)
	s.current = nil
	return nil
}

func (s *recordingUncertaintySink) RecordGovernedCheckUncertainty(_ context.Context, fact ports.GovernedCheckUncertainty) error {
	s.facts = append(s.facts, fact)
	s.current = &fact
	return nil
}

func (s *recordingUncertaintySink) GovernedCheckUncertainty(_ context.Context, sessionID domain.SessionID) (ports.GovernedCheckUncertainty, bool, error) {
	if s.current == nil || s.current.SessionID != sessionID {
		return ports.GovernedCheckUncertainty{}, false, nil
	}
	return *s.current, true, nil
}

func TestServeRecordsUnknownTerminationAndRefusesLaterEffects(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "report.md"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	policy, err := testPolicy(true, true).BindWorkspaceRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	requests := []map[string]any{
		{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "run_approved_check", "arguments": map[string]any{"check_id": "check-1"}}},
		{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "write_text_file", "arguments": map[string]any{"path": "report.md", "content": "changed"}}},
		{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": map[string]any{"name": "run_approved_check", "arguments": map[string]any{"check_id": "check-1"}}},
	}
	var input bytes.Buffer
	for _, request := range requests {
		if err := json.NewEncoder(&input).Encode(request); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	sink := &recordingUncertaintySink{}
	runs := 0
	server := Server{
		Policy: policy, WorkspaceRoot: root, SessionID: "session-1", In: &input, Out: &output,
		UncertaintySink: sink, UncertaintySource: sink,
		RunCheck: func(context.Context, governedcheck.Request) (governedcheck.Result, error) {
			runs++
			return governedcheck.Result{ExitCode: -1, EnforcedBy: "test-fence", TerminationUnknown: true}, nil
		},
	}
	if err := server.Serve(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runs != 1 {
		t.Fatalf("check invocations = %d, want exactly 1 after uncertainty latch", runs)
	}
	if len(sink.facts) != 2 || sink.facts[1].SessionID != "session-1" || !sink.facts[1].TerminationUnknown || len(sink.clears) != 0 {
		t.Fatalf("recorded uncertainty = %+v", sink.facts)
	}
	if text := output.String(); !strings.Contains(text, "termination_unknown=true") || strings.Count(text, `"isError":true`) != 3 {
		t.Fatalf("MCP output did not surface and latch uncertainty: %s", text)
	}
	content, err := os.ReadFile(filepath.Join(root, "report.md"))
	if err != nil || string(content) != "original" {
		t.Fatalf("post-uncertainty write changed report: %q, %v", content, err)
	}

	var restartInput, restartOutput bytes.Buffer
	if err := json.NewEncoder(&restartInput).Encode(requests[1]); err != nil {
		t.Fatal(err)
	}
	restarted := Server{
		Policy: policy, WorkspaceRoot: root, SessionID: "session-1",
		In: &restartInput, Out: &restartOutput, UncertaintySink: sink, UncertaintySource: sink,
	}
	if err := restarted.Serve(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(restartOutput.String(), "effects blocked by unknown check termination") {
		t.Fatalf("restarted MCP server did not restore uncertainty latch: %s", restartOutput.String())
	}
}
