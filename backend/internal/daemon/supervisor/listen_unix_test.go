//go:build !windows

package supervisor

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestListen_basic verifies that Listen remains usable when running.json lives
// under a path longer than sockaddr_un permits. The published address must be
// short, private, and dialable; it is not derived beside the run-file.
func TestListen_basic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	longDir := filepath.Join(dir, strings.Repeat("profile-", 20))
	if err := os.MkdirAll(longDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	runFile := filepath.Join(longDir, "running.json")

	ln, addr, err := Listen(runFile)
	if err != nil {
		t.Fatalf("Listen: unexpected error: %v", err)
	}
	defer ln.Close()

	if len([]byte(addr)) > maxUnixSocketPathBytes {
		t.Errorf("addr length = %d, want <= %d: %q", len([]byte(addr)), maxUnixSocketPathBytes, addr)
	}
	if filepath.Dir(addr) == filepath.Dir(runFile) {
		t.Errorf("addr = %q unexpectedly lives beside long run-file %q", addr, runFile)
	}

	// Socket file must exist after Listen.
	info, err := os.Stat(addr)
	if err != nil {
		t.Errorf("socket file missing after Listen: %v", err)
	} else if info.Mode().Perm() != 0o600 {
		t.Errorf("socket permissions = %o, want 600", info.Mode().Perm())
	}

	// Dialing the returned address must succeed.
	conn, err := net.Dial("unix", addr)
	if err != nil {
		t.Fatalf("Dial(%q): %v", addr, err)
	}
	conn.Close()
}

// TestListen_doesNotReplaceProfileObject verifies that the old profile-local
// path is not touched. This also makes the safety boundary explicit: the
// supervisor does not remove unrelated filesystem objects to make startup work.
func TestListen_doesNotReplaceProfileObject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	runFile := filepath.Join(dir, "running.json")
	sockPath := filepath.Join(dir, "supervise.sock")

	// Pre-create a regular file to simulate a stale socket.
	if err := os.WriteFile(sockPath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("pre-create stale file: %v", err)
	}

	ln, _, err := Listen(runFile)
	if err != nil {
		t.Fatalf("Listen with stale socket: unexpected error: %v", err)
	}
	ln.Close()
	if contents, readErr := os.ReadFile(sockPath); readErr != nil || string(contents) != "stale" {
		t.Errorf("profile-local object changed: contents=%q err=%v", contents, readErr)
	}
}

// TestListen_unlinkOnClose verifies that closing the listener removes the
// socket file from the filesystem.
func TestListen_unlinkOnClose(t *testing.T) {
	t.Parallel()
	ln, addr, err := Listen(filepath.Join(t.TempDir(), "running.json"))
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	if err := ln.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := os.Stat(addr); !os.IsNotExist(err) {
		t.Errorf("socket file still present after Close (err=%v); expected not-exist", err)
	}
}

func TestListen_closeDoesNotRemoveReplacedPath(t *testing.T) {
	t.Parallel()
	ln, addr, err := Listen(filepath.Join(t.TempDir(), "running.json"))
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	backup := addr + ".moved"
	if err := os.Rename(addr, backup); err != nil {
		ln.Close()
		t.Fatalf("rename socket: %v", err)
	}
	if err := os.WriteFile(addr, []byte("replacement"), 0o600); err != nil {
		ln.Close()
		t.Fatalf("create replacement: %v", err)
	}
	if err := ln.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if contents, err := os.ReadFile(addr); err != nil || string(contents) != "replacement" {
		t.Errorf("replacement path changed: contents=%q err=%v", contents, err)
	}
	_ = os.Remove(backup)
}
