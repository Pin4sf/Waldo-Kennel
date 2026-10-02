// Package bridgeruntime connects an already paired identity. It never creates
// identities, redeems codes, or authorizes owner commands.
package bridgeruntime

import (
	"context"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
)

// UnknownQueries intentionally ships no machine evidence. Notifications have
// no Display adapter either, so connected means unknown / delivery_unknown.
type UnknownQueries struct{}

var _ devicebridge.QueryEvidenceSource = UnknownQueries{}

func (UnknownQueries) QueryState(context.Context, devicebridge.Scope, string, string) (string, bool, error) {
	return "", false, nil
}
