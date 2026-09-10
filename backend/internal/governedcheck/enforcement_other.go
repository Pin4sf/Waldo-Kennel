//go:build !darwin

package governedcheck

import (
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// platformEnforcement reports that this host has no mechanism able to apply an
// approved policy to a check process.
//
// Linux would use Landlock or a bubblewrap-style namespace and Windows a job
// object with a restricted token; neither is implemented here. Until one is,
// checks fail closed on these platforms rather than running unconfined.
func platformEnforcement(domain.AttemptExecutionPolicy) (Enforcement, error) {
	return nil, fmt.Errorf("%w: no sandbox mechanism is implemented for this platform", ErrEnforcementUnavailable)
}
