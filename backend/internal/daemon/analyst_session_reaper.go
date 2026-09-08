package daemon

import (
	"context"
	"strings"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// intakeAnalysisSweepInterval is how often a running daemon re-checks durable
// expiry. Well under the request TTL, so an abandoned ask reaches a verdict
// within a fraction of its own lifetime rather than at the next restart.
const intakeAnalysisSweepInterval = 2 * time.Minute

// sessionReaper ends a bounded proposing session through the ordinary session
// service, adding no new teardown path of its own.
//
// Waldo now reasons through its own model rather than by spawning a coding
// agent, so nothing new creates these sessions. The reaper is retained for
// compatibility recovery: historical intakes whose analyst session was still
// running when the daemon restarted are cleaned up through this same path.
type sessionReaper struct {
	sessions attemptSessionControl
}

var _ ports.AnalystSessionReaper = sessionReaper{}

func (r sessionReaper) Kill(ctx context.Context, sessionID string) error {
	if r.sessions == nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	_, err := r.sessions.Kill(ctx, domain.SessionID(sessionID))
	return err
}
