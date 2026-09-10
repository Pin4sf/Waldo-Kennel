package intelligence

import (
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// planDraftChecks narrows model-proposed check commands into draft shape.
//
// The argument vector is copied verbatim. These are program arguments, not
// labels: trimming them would change what the command does, and dropping an
// empty one would shift every argument after it into a different position —
// silently turning the proposal into a different command than the one that
// will be reviewed. Whether an empty argument is acceptable is a decision the
// control plane makes explicitly when it compiles the proposal; it is not
// something normalization may quietly resolve.
//
// The criterion alias is a label, so trimming it is safe and useful.
func planDraftChecks(proposed []struct {
	CriterionAlias string   `json:"criterionAlias"`
	Argv           []string `json:"argv"`
	TimeoutSeconds int64    `json:"timeoutSeconds"`
}) []domain.PlanDraftCheck {
	checks := make([]domain.PlanDraftCheck, 0, len(proposed))
	for _, item := range proposed {
		alias := strings.TrimSpace(item.CriterionAlias)
		if alias == "" && len(item.Argv) == 0 {
			// Nothing was proposed here at all. Anything less empty than this
			// reaches validation and is refused with a reason.
			continue
		}
		checks = append(checks, domain.PlanDraftCheck{
			CriterionAlias: alias,
			Argv:           append([]string(nil), item.Argv...),
			TimeoutSeconds: item.TimeoutSeconds,
		})
	}
	return checks
}
