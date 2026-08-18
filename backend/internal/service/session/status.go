package session

import (
	"strings"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/pkg/contract"
)

// noSignalGrace is how long after spawn/restore a session may stay silent
// before its idle reading is downgraded to StatusNoSignal. It covers the
// agent's TUI boot plus the gap to the first activity-bearing hook callback
// (for Codex that is UserPromptSubmit, seconds after the auto-submitted spawn
// prompt — its SessionStart hook fires earlier but carries no activity state);
// past it, a silent session is indistinguishable from one with a broken hook
// pipeline, and the dashboard must not claim a confident "idle".
const noSignalGrace = 90 * time.Second

func deriveStatus(rec domain.SessionRecord, prs []domain.PRFacts, now time.Time, signalCapable bool) domain.SessionStatus {
	status := domain.SessionStatus(contract.DeriveStatus(
		toContractSessionFacts(rec, signalCapable),
		toContractPRFacts(prs),
		now,
		noSignalGrace,
	))

	// Coordinated work has a lifecycle that cannot be inferred from terminal
	// activity alone. In particular, a completed agent turn is "idle" even when
	// its work is ready for the user. Orchestrators and workers therefore report
	// explicit, durable markers in their latest assistant updates. Keep real
	// termination and SCM evidence authoritative; markers fill only the
	// provider-activity gap for sessions without attributed PR facts.
	if status == domain.StatusTerminated || status == domain.StatusWorking || status == domain.StatusNeedsInput || len(prs) > 0 {
		return status
	}
	if reported, ok := reportedCoordinationStatus(rec); ok {
		return reported
	}
	return status
}

const outcomeStatusMarker = "KENNEL_OUTCOME_STATUS:"
const workStatusMarker = "KENNEL_WORK_STATUS:"

func isOutcomeSession(rec domain.SessionRecord) bool {
	return rec.Kind == domain.KindOrchestrator && strings.HasPrefix(strings.TrimSpace(rec.DisplayName), "Outcome:")
}

// reportedCoordinationStatus reads the last marker so a response that summarizes a
// previous phase before announcing the current one cannot move backwards.
func reportedCoordinationStatus(rec domain.SessionRecord) (domain.SessionStatus, bool) {
	marker := workStatusMarker
	if isOutcomeSession(rec) {
		marker = outcomeStatusMarker
	} else if rec.Kind != domain.KindWorker {
		return "", false
	}
	return reportedStatusMarker(rec.Metadata.LatestAssistantUpdate, marker)
}

func reportedStatusMarker(update, marker string) (domain.SessionStatus, bool) {
	index := strings.LastIndex(update, marker)
	if index < 0 {
		return "", false
	}
	value := strings.TrimSpace(update[index+len(marker):])
	if end := strings.IndexAny(value, "\r\n \t"); end >= 0 {
		value = value[:end]
	}
	switch strings.ToLower(value) {
	case "working":
		return domain.StatusWorking, true
	case "needs_you":
		return domain.StatusNeedsInput, true
	case "reviewing":
		return domain.StatusReviewPending, true
	case "ready_to_merge", "finished":
		return domain.StatusMergeable, true
	default:
		return "", false
	}
}

func deriveSCMStatus(prs []domain.PRFacts) domain.SessionStatus {
	return domain.SessionStatus(contract.DeriveSCMStatus(toContractPRFacts(prs)))
}

func toContractSessionFacts(rec domain.SessionRecord, signalCapable bool) contract.SessionFacts {
	return contract.SessionFacts{
		Activity:       contract.ActivityState(rec.Activity.State),
		LastActivityAt: rec.Activity.LastActivityAt,
		HasSignal:      !rec.FirstSignalAt.IsZero(),
		SignalExpected: signalCapable && rec.Mode != domain.SessionModeChat,
		IsTerminated:   rec.IsTerminated,
	}
}

func toContractPRFacts(prs []domain.PRFacts) []contract.PRFacts {
	facts := make([]contract.PRFacts, len(prs))
	for i, pr := range prs {
		facts[i] = contract.PRFacts{
			URL:            pr.URL,
			Draft:          pr.Draft,
			Merged:         pr.Merged,
			Closed:         pr.Closed,
			CI:             pr.CI,
			Review:         pr.Review,
			Mergeability:   pr.Mergeability,
			ReviewComments: pr.ReviewComments,
			SourceBranch:   pr.SourceBranch,
			TargetBranch:   pr.TargetBranch,
		}
	}
	return facts
}
