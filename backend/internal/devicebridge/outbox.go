package devicebridge

import (
	"context"
	"errors"
	"time"
)

var ErrUnknownReceiptMessage = errors.New("receipt names unknown result")

// Scope is mandatory on every client/store operation. There is deliberately no
// adapter to the legacy globally keyed 0162 journal/outbox. Gate B must supply
// and prove a durable implementation before daemon activation.
type Scope struct{ DeviceID, OwnerID string }

func (s Scope) valid() bool { return validIdentifier(s.DeviceID) && validIdentifier(s.OwnerID) }
func (s Scope) matches(frame map[string]any) bool {
	return frame["device_id"] == s.DeviceID && frame["owner_id"] == s.OwnerID
}

type PendingResult struct {
	Seq   int64
	Frame []byte
}
type Admission struct {
	State, Reason string
	New           bool
}

// SessionStore operations are transactional durable boundaries, not callbacks
// that may fall back to globally scoped identities. Implementations must:
//   - dedup (device,message_id) with stable fingerprint and journal
//     (device,command_id,revision), unique (device,idempotency_key);
//   - journal acceptance before effects and atomically commit journal outcome
//     plus exactly one result outbox row per device/command/revision;
//   - atomically validate receipt binding, remove its outbox row and persist a
//     tombstone; matching duplicate tombstones no-op, unknown/conflict fail;
//   - retain tombstones until expires_at+300 AND no matching in-flight result,
//     cap 100000/device, reject new commands instead of evicting live records.
//
// Audit/journal records survive ClearPaired. Pending includes sent rows until
// receipted and returns oldest durable seq first. These are Gate B proof gates.
type SessionStore interface {
	Pending(context.Context, Scope) ([]PendingResult, error)
	Depth(context.Context, Scope) (int64, error)
	AcceptedUnresulted(context.Context, Scope) ([][]byte, error)
	// ResultForCommand returns the retained, validated signed result for this
	// journaled command, even after receipt removes its outbox row. Nil means
	// accepted but unresulted. It must validate the command fingerprint.
	ResultForCommand(context.Context, Scope, []byte) ([]byte, bool, error)
	Admit(context.Context, Scope, []byte, time.Time) (Admission, error)
	CommitResult(context.Context, Scope, []byte, []byte) error
	ApplyReceipt(context.Context, Scope, []byte, time.Time) error
	ClearPaired(context.Context, Scope) error
}
