package ports

import (
	"context"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// DeviceBridgeStore durably tracks the paired-device principal, the outbound
// bridge outbox, and the replay-safe inbound command journal (ADR 0018,
// contract v0.1). No private key material, bearer secret, or owner proof
// crosses this boundary; the store is infrastructure state and emits no
// change_log events.
type DeviceBridgeStore interface {
	// CreateDeviceBridgeDevice inserts a new device record in the pairing
	// state. A duplicate device ID is ErrDeviceBridgeConflict.
	CreateDeviceBridgeDevice(ctx context.Context, rec domain.DeviceBridgeDevice) error
	GetDeviceBridgeDevice(ctx context.Context, id domain.DeviceBridgeDeviceID) (domain.DeviceBridgeDevice, bool, error)
	// GetPairedDeviceBridgeDevice returns the owner's currently paired device,
	// if any. At most one device per owner may be paired at a time.
	GetPairedDeviceBridgeDevice(ctx context.Context, ownerID string) (domain.DeviceBridgeDevice, bool, error)
	// TransitionDeviceBridgeDevice is a compare-and-swap on state over the
	// closed lifecycle (pairing -> paired, pairing|paired -> revoked),
	// stamping PairedAt/RevokedAt as the target requires, and reports
	// whether this call won the transition.
	TransitionDeviceBridgeDevice(ctx context.Context, id domain.DeviceBridgeDeviceID, to domain.DeviceBridgeState, now time.Time) (bool, error)
	// TouchDeviceBridgeDeviceLastSeen records a heartbeat; it only ever moves
	// last_seen forward and never resurrects a revoked device.
	TouchDeviceBridgeDeviceLastSeen(ctx context.Context, id domain.DeviceBridgeDeviceID, now time.Time) (bool, error)

	// EnqueueDeviceBridgeOutbox spools one outbound message and returns its
	// durable seq cursor. A duplicate message ID or (command ID, revision) is
	// ErrDeviceBridgeConflict: redelivery reuses the existing entry instead.
	EnqueueDeviceBridgeOutbox(ctx context.Context, msg domain.DeviceBridgeOutboxMessage) (int64, error)
	// GetDeviceBridgeOutboxByMessageID reads one spooled message by its
	// stable message ID.
	GetDeviceBridgeOutboxByMessageID(ctx context.Context, messageID string) (domain.DeviceBridgeOutboxMessage, bool, error)
	// ListPendingDeviceBridgeOutbox returns unreceipted messages in seq order
	// (oldest first) for drain-on-reconnect.
	ListPendingDeviceBridgeOutbox(ctx context.Context, limit int) ([]domain.DeviceBridgeOutboxMessage, error)
	// MarkDeviceBridgeOutboxSent transitions pending -> sent.
	MarkDeviceBridgeOutboxSent(ctx context.Context, seq int64, now time.Time) (bool, error)
	// MarkDeviceBridgeOutboxReceipted transitions pending|sent -> receipted on
	// backend receipt. This is the only path that clears an entry from drain.
	MarkDeviceBridgeOutboxReceipted(ctx context.Context, seq int64, now time.Time) (bool, error)
	// CountPendingDeviceBridgeOutbox reports drain depth for honest state.
	CountPendingDeviceBridgeOutbox(ctx context.Context) (int64, error)

	// RecordDeviceBridgeInbound journals one inbound command replay-safely:
	// the first record wins; redelivery with the same idempotency key and the
	// same payload fingerprint returns the existing record with created=false;
	// the same key with a different fingerprint is ErrDeviceBridgeConflict.
	RecordDeviceBridgeInbound(ctx context.Context, cmd domain.DeviceBridgeInboxCommand) (domain.DeviceBridgeInboxCommand, bool, error)
	GetDeviceBridgeInboundByIdempotencyKey(ctx context.Context, key string) (domain.DeviceBridgeInboxCommand, bool, error)
	// TransitionDeviceBridgeInbound is a compare-and-swap on journal state
	// over the closed lifecycle (received -> acked/rejected/expired,
	// acked -> resulted), and reports whether this call won the transition.
	TransitionDeviceBridgeInbound(ctx context.Context, commandID string, revision int64, to domain.DeviceBridgeInboxState, rejectReason string, now time.Time) (bool, error)
}
