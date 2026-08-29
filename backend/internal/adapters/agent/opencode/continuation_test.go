package opencode

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

func TestContinuationCapabilitiesAreProviderAssigned(t *testing.T) {
	caps := New().ContinuationCapabilities()
	if caps.FreshNativeSessionID != ports.FreshNativeSessionIDProviderAssigned {
		t.Fatalf("FreshNativeSessionID = %q, want provider_assigned", caps.FreshNativeSessionID)
	}
}

// opencode keeps history in SQLite rows rather than a per-session file, so the
// adapter must not claim the transcript-locator capability. Claiming it would
// make the switch path advertise a handoff artifact that has no path.
func TestPluginDoesNotClaimTranscriptLocation(t *testing.T) {
	if _, ok := any(New()).(ports.AgentTranscriptLocator); ok {
		t.Fatal("opencode must not implement AgentTranscriptLocator: history has no per-session file")
	}
}

// isolateOpenCodeEnv clears the state-root variables in the process
// environment. Resolution falls back to the process environment for any key the
// launch environment does not carry, so without this a developer's own
// OPENCODE_DATA_DIR would decide the result instead of the test's input.
func isolateOpenCodeEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"OPENCODE_DATA_DIR", "XDG_DATA_HOME", "HOME"} {
		t.Setenv(key, "")
	}
}

func TestNativeSessionConfigDirPrefersLaunchEnvironment(t *testing.T) {
	isolateOpenCodeEnv(t)
	plugin := New()
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "explicit data dir", env: map[string]string{"OPENCODE_DATA_DIR": "/tmp/oc-state"}, want: "/tmp/oc-state"},
		{name: "xdg data home", env: map[string]string{"XDG_DATA_HOME": "/tmp/xdg"}, want: filepath.Join("/tmp/xdg", "opencode")},
		{name: "home fallback", env: map[string]string{"HOME": "/tmp/home"}, want: filepath.Join("/tmp/home", ".local", "share", "opencode")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plugin.NativeSessionConfigDir(context.Background(), tt.env)
			if err != nil {
				t.Fatalf("NativeSessionConfigDir: %v", err)
			}
			if got != tt.want {
				t.Fatalf("NativeSessionConfigDir = %q, want %q", got, tt.want)
			}
		})
	}
}

// OPENCODE_DATA_DIR must win over XDG_DATA_HOME, matching opencode's own
// precedence, so a session pinned to its own state is probed against that state.
func TestNativeSessionConfigDirPrecedence(t *testing.T) {
	isolateOpenCodeEnv(t)
	got, err := New().NativeSessionConfigDir(context.Background(), map[string]string{
		"OPENCODE_DATA_DIR": "/tmp/explicit",
		"XDG_DATA_HOME":     "/tmp/xdg",
		"HOME":              "/tmp/home",
	})
	if err != nil {
		t.Fatalf("NativeSessionConfigDir: %v", err)
	}
	if got != "/tmp/explicit" {
		t.Fatalf("NativeSessionConfigDir = %q, want /tmp/explicit", got)
	}
}

// writeOpenCodeState builds a database with opencode's real session columns so
// the probe is exercised against the shape it reads in production.
func writeOpenCodeState(t *testing.T, dir string, rows ...sessionRow) {
	t.Helper()
	path := filepath.Join(dir, opencodeStateDBName)
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("open state: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE session (
		id text PRIMARY KEY,
		project_id text NOT NULL,
		directory text NOT NULL,
		title text NOT NULL,
		time_created integer NOT NULL,
		time_archived integer
	)`); err != nil {
		t.Fatalf("create session table: %v", err)
	}
	for _, row := range rows {
		if _, err := db.Exec(
			`INSERT INTO session (id, project_id, directory, title, time_created, time_archived) VALUES (?, 'global', '/tmp', 'test', 1, ?)`,
			row.id, row.archived,
		); err != nil {
			t.Fatalf("insert session %s: %v", row.id, err)
		}
	}
}

type sessionRow struct {
	id       string
	archived any
}

func TestProbeNativeSession(t *testing.T) {
	dir := t.TempDir()
	writeOpenCodeState(t, dir,
		sessionRow{id: "ses_live", archived: nil},
		sessionRow{id: "ses_archived", archived: int64(1786625559067)},
	)

	plugin := New()
	tests := []struct {
		name      string
		sessionID string
		configDir string
		want      ports.NativeSessionAvailability
	}{
		{name: "active session resumes", sessionID: "ses_live", configDir: dir, want: ports.NativeSessionAvailabilityAvailable},
		{name: "archived session is not resumable", sessionID: "ses_archived", configDir: dir, want: ports.NativeSessionAvailabilityUnavailable},
		{name: "absent session is authoritative", sessionID: "ses_missing", configDir: dir, want: ports.NativeSessionAvailabilityUnavailable},
		// Everything AO cannot prove stays Unknown: Unavailable would make the
		// switch path discard a resume handle that may still be good.
		{name: "missing state file is unknown", sessionID: "ses_live", configDir: t.TempDir(), want: ports.NativeSessionAvailabilityUnknown},
		{name: "no config dir is unknown", sessionID: "ses_live", configDir: "", want: ports.NativeSessionAvailabilityUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plugin.ProbeNativeSession(context.Background(), ports.NativeSessionRef{
				NativeSessionID: tt.sessionID,
				ConfigDir:       tt.configDir,
			})
			if err != nil {
				t.Fatalf("ProbeNativeSession: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ProbeNativeSession = %q, want %q", got, tt.want)
			}
		})
	}
}

// A state file older than the columns the probe reads cannot answer the
// question, so it must degrade to Unknown rather than report a live session as
// deleted.
func TestProbeNativeSessionUnknownOnOlderSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, opencodeStateDBName)
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("open state: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE session (id text PRIMARY KEY)`); err != nil {
		t.Fatalf("create legacy session table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO session (id) VALUES ('ses_live')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	_ = db.Close()

	got, err := New().ProbeNativeSession(context.Background(), ports.NativeSessionRef{
		NativeSessionID: "ses_live",
		ConfigDir:       dir,
	})
	if err != nil {
		t.Fatalf("ProbeNativeSession: %v", err)
	}
	if got != ports.NativeSessionAvailabilityUnknown {
		t.Fatalf("ProbeNativeSession = %q, want unknown", got)
	}
}

func TestProbeNativeSessionRejectsMalformedID(t *testing.T) {
	dir := t.TempDir()
	writeOpenCodeState(t, dir, sessionRow{id: "ses_live", archived: nil})

	for _, id := range []string{"", "   ", "ses/../escape", "ses\x00null"} {
		got, err := New().ProbeNativeSession(context.Background(), ports.NativeSessionRef{
			NativeSessionID: id,
			ConfigDir:       dir,
		})
		if err == nil {
			t.Fatalf("ProbeNativeSession(%q) = nil error, want rejection", id)
		}
		if got != ports.NativeSessionAvailabilityUnknown {
			t.Fatalf("ProbeNativeSession(%q) = %q, want unknown", id, got)
		}
	}
}

func TestProbeNativeSessionHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New().ProbeNativeSession(ctx, ports.NativeSessionRef{
		NativeSessionID: "ses_live",
		ConfigDir:       t.TempDir(),
	}); err == nil {
		t.Fatal("ProbeNativeSession = nil error on canceled context, want cancellation")
	}
}
