package intelligence

import (
	"context"
	"fmt"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

const interruptedRunDetail = "The daemon restarted before reasoning reached a terminal result; retry explicitly."

// ReconcileInterruptedRuns consumes durable requested/running IntelligenceRuns
// during boot. A synchronous call cannot remain visibly active after its owner
// process died, and the paid call is never retried automatically because its
// side effects/billing may be ambiguous.
func ReconcileInterruptedRuns(ctx context.Context, store ports.IntelligenceRunStore, now func() time.Time) (int, error) {
	if store == nil {
		return 0, fmt.Errorf("intelligence run store is unavailable")
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	runs, err := store.ListNonTerminalIntelligenceRuns(ctx)
	if err != nil {
		return 0, fmt.Errorf("list interrupted intelligence runs: %w", err)
	}
	recovered := 0
	for _, run := range runs {
		completed := now().UTC()
		if err := store.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunExpired, "", "INTELLIGENCE_INTERRUPTED", interruptedRunDetail, &completed); err != nil {
			return recovered, fmt.Errorf("reconcile intelligence run %s: %w", run.ID, err)
		}
		recovered++
	}
	return recovered, nil
}
