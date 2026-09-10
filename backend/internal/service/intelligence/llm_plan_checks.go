package intelligence

import (
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// planDraftChecks narrows model-proposed check commands into draft shape.
//
// Nothing here decides whether a command is allowed; that is the control
// plane's job at compile time. This only trims and drops entries too empty to
// mean anything, so an obviously malformed suggestion does not reach
// validation as a confusing error about a blank argument.
func planDraftChecks(proposed []struct {
	CriterionAlias string   `json:"criterionAlias"`
	Argv           []string `json:"argv"`
	TimeoutSeconds int64    `json:"timeoutSeconds"`
}) []domain.PlanDraftCheck {
	checks := make([]domain.PlanDraftCheck, 0, len(proposed))
	for _, item := range proposed {
		argv := trimAll(item.Argv)
		alias := strings.TrimSpace(item.CriterionAlias)
		if alias == "" || len(argv) == 0 {
			continue
		}
		checks = append(checks, domain.PlanDraftCheck{
			CriterionAlias: alias, Argv: argv, TimeoutSeconds: item.TimeoutSeconds,
		})
	}
	return checks
}
