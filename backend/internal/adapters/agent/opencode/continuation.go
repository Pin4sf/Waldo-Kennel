package opencode

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

var (
	_ ports.AgentContinuationCapabilityProvider = (*Plugin)(nil)
	_ ports.AgentNativeSessionConfigProvider    = (*Plugin)(nil)
	_ ports.AgentNativeSessionProber            = (*Plugin)(nil)
)

// opencode deliberately does not implement ports.AgentTranscriptLocator.
// Claude Code and Codex each write a per-conversation transcript file that Kennel
// can point at, but opencode keeps message history in rows of its SQLite state
// (session_message), not in a readable per-session file. The locator contract
// returns a filesystem path, so implementing it would mean inventing a path
// that does not exist. Its absence is already the honest answer: the switch
// path records the source transcript as unavailable rather than advertising a
// handoff artifact Kennel cannot produce.

// ContinuationCapabilities reports that opencode assigns its own conversation
// ids. Kennel learns the id from the workspace activity plugin, which forwards
// opencode's native session.id through `kennel hooks opencode session-start`,
// and GetRestoreCommand replays it with `--session`. Kennel never selects the id,
// so caller-assigned continuation is deliberately not claimed.
func (p *Plugin) ContinuationCapabilities() ports.ContinuationCapabilities {
	return ports.ContinuationCapabilities{
		FreshNativeSessionID: ports.FreshNativeSessionIDProviderAssigned,
	}
}

// NativeSessionConfigDir returns the opencode state root whose session records
// back a resume. The launch environment wins over the daemon's own, so a
// session pinned to its own OPENCODE_DATA_DIR is probed against exactly the
// state it will run with rather than the user's default account.
func (p *Plugin) NativeSessionConfigDir(ctx context.Context, env map[string]string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	dir, ok := opencodeDataDirFrom(env)
	if !ok {
		return "", errors.New("opencode: resolve data dir: no OPENCODE_DATA_DIR, XDG_DATA_HOME, or home directory")
	}
	return dir, nil
}

// ProbeNativeSession reports whether opencode still holds resumable backing
// state for a native session id. Evidence is the `session` row in opencode's
// SQLite state: present and not archived is resumable, archived is not (it
// mirrors Codex, where an archived conversation stays readable history but is
// no longer an active conversation to continue).
//
// Every case Kennel cannot prove returns Unknown rather than Unavailable, because
// Unavailable makes the switch path discard a resume handle: a missing state
// file, a schema older than the columns read here, and a locked or unreadable
// database are all "no authoritative answer", not "the session is gone".
func (p *Plugin) ProbeNativeSession(ctx context.Context, ref ports.NativeSessionRef) (ports.NativeSessionAvailability, error) {
	if err := ctx.Err(); err != nil {
		return ports.NativeSessionAvailabilityUnknown, err
	}
	configDir := strings.TrimSpace(ref.ConfigDir)
	if configDir == "" {
		return ports.NativeSessionAvailabilityUnknown, nil
	}
	sessionID, err := validateOpenCodeNativeSessionID(ref.NativeSessionID)
	if err != nil {
		return ports.NativeSessionAvailabilityUnknown, err
	}

	path := filepath.Join(configDir, opencodeStateDBName)
	if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
		return ports.NativeSessionAvailabilityUnknown, nil
	} else if statErr != nil {
		return ports.NativeSessionAvailabilityUnknown, fmt.Errorf("opencode: stat session state: %w", statErr)
	}

	db, err := openOpenCodeStateDB(path)
	if err != nil {
		return ports.NativeSessionAvailabilityUnknown, err
	}
	defer func() { _ = db.Close() }()

	var archived sql.NullInt64
	row := db.QueryRowContext(ctx, `SELECT time_archived FROM session WHERE id = ?`, sessionID)
	switch err := row.Scan(&archived); {
	case errors.Is(err, sql.ErrNoRows):
		return ports.NativeSessionAvailabilityUnavailable, nil
	case err != nil:
		// A state file whose shape predates the columns read here cannot answer
		// the question; it must not read as a deleted conversation.
		if isMissingSchemaErr(err) {
			return ports.NativeSessionAvailabilityUnknown, nil
		}
		return ports.NativeSessionAvailabilityUnknown, fmt.Errorf("opencode: read session state: %w", err)
	}
	if archived.Valid {
		return ports.NativeSessionAvailabilityUnavailable, nil
	}
	return ports.NativeSessionAvailabilityAvailable, nil
}

func isMissingSchemaErr(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "no such table") || strings.Contains(text, "no such column")
}

// validateOpenCodeNativeSessionID rejects ids that cannot be a native session
// handle. opencode mints opaque prefixed ids (ses_...), so the check stays a
// shape guard rather than a format assertion that a future id scheme would
// break.
func validateOpenCodeNativeSessionID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 256 || strings.ContainsAny(value, `/\`+"\x00") {
		return "", errors.New("opencode: invalid native session id")
	}
	return value, nil
}
