// Package dshtest provides scripted fake `dsh` executables and shared
// scenario helpers so DeepSeek Harness admission/lifecycle behavior can be
// exercised from ANY package's tests without the real CLI.
//
// Issue #60 requires failure-injection fixtures that are importable across
// packages; this package is that import surface. The scripts are plain POSIX
// shell and behave deterministically:
//
//   - `--dump-config` composes (exit 0) unless DSH_TEST_COMPOSE_FAIL=1,
//     or hangs until killed when DSH_TEST_HANG=1 — probing truth;
//   - an actual launch prints a startup banner, sleeps
//     DSH_TEST_RUN_SECONDS (default 0.2), then exits DSH_TEST_EXIT_CODE
//     (default 0) — mid-run loss is DSH_TEST_EXIT_CODE=1.
//
// Environment variables are read at execution time, so one binary serves both
// readiness probes and launches with per-test overrides via t.Setenv.
package dshtest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Env knobs read by every scripted binary at execution time.
const (
	EnvComposeFail = "DSH_TEST_COMPOSE_FAIL" // =1 makes --dump-config exit 3
	EnvHang        = "DSH_TEST_HANG"         // =1 makes --dump-config sleep 30s
	EnvRunSeconds  = "DSH_TEST_RUN_SECONDS"  // launch sleep length
	EnvExitCode    = "DSH_TEST_EXIT_CODE"    // launch exit code (1 = mid-run loss)
)

// Binary is a scripted fake dsh executable living under its own directory so
// callers can prepend that directory to PATH.
type Binary struct {
	Dir string
}

// ScriptedBinary writes a fake `dsh` executable into a fresh temp dir and
// returns it. Unix-only: shell-script fakes cannot express Windows launch
// semantics, mirroring the in-package scaffold's platform gate.
func ScriptedBinary(t testing.TB) *Binary {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("scripted fake dsh binaries are unix-only")
	}
	dir := t.TempDir()
	b := &Binary{Dir: dir}
	writeScript(t, filepath.Join(dir, "dsh"), launchAndDumpScript())
	return b
}

// WriteScript writes body as an executable shell script named name under a
// fresh temp dir and returns its full path. Exported for in-package and
// cross-package scaffolds that need bespoke bodies.
func WriteScript(t testing.TB, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("scripted fake binaries are unix-only")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	WriteScriptAt(t, path, body)
	return path
}

// WriteScriptAt writes body as an executable shell script at exactly path
// (the caller owns the directory layout).
func WriteScriptAt(t testing.TB, path, body string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("scripted fake binaries are unix-only")
	}
	writeScript(t, path, body)
}

// PrependToPATH puts the binary's directory ahead of the current PATH so the
// adapter's standard resolution finds the fake before any real install.
func (b *Binary) PrependToPATH(t testing.TB) {
	t.Helper()
	t.Setenv("PATH", b.Dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func writeScript(t testing.TB, path, body string) {
	t.Helper()
	script := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake dsh script: %v", err)
	}
}

func launchAndDumpScript() string {
	return `
dump=0
profile=""
prev=""
for a in "$@"; do
  if [ "$prev" = "--profile" ]; then profile="$a"; fi
  if [ "$a" = "--dump-config" ]; then dump=1; fi
  prev="$a"
done
if [ "$dump" = "1" ]; then
  if [ "$DSH_TEST_HANG" = "1" ]; then sleep 30; fi
  if [ "$DSH_TEST_COMPOSE_FAIL" = "1" ]; then echo "invalid profile layer" >&2; exit 3; fi
  echo "# == composed profile $profile"
  exit 0
fi
echo "dsh session started profile=$profile"
sleep "${DSH_TEST_RUN_SECONDS:-0.2}"
exit "${DSH_TEST_EXIT_CODE:-0}"`
}
