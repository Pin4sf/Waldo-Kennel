package opencode

import "github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"

// DeriveActivityState maps an opencode plugin hook event onto an Kennel activity
// state. The opencode plugin (assets/kennel-activity.ts) normalizes opencode's
// native events to "session-start" / "user-prompt-submit" / "stop" before
// invoking `kennel hooks opencode <event>`. The bool is false when the event
// carries no activity signal.
func DeriveActivityState(event string, _ []byte) (domain.ActivityState, bool) {
	switch event {
	case "session-start":
		return domain.ActivityActive, true
	case "user-prompt-submit":
		return domain.ActivityActive, true
	case "stop":
		return domain.ActivityIdle, true
	default:
		return "", false
	}
}
