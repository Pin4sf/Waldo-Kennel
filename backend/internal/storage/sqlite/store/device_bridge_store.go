package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
)

var _ ports.DeviceBridgeStore = (*Store)(nil)

// Device bridge store (ADR 0018, contract v0.1). Bridge state is daemon
// infrastructure, not work-state: these tables carry no change_log triggers
// and hold no secret material. Every write goes through the single writer
// connection; CAS transitions report whether the caller won the transition.

func (s *Store) CreateDeviceBridgeDevice(ctx context.Context, rec domain.DeviceBridgeDevice) error {
	if rec.State != domain.DeviceBridgeStatePairing {
		return domain.ErrDeviceBridgeInvalid
	}
	if err := rec.Validate(); err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.qw.InsertDeviceBridgeDevice(ctx, deviceBridgeDeviceInsert(rec)); err != nil {
		if isSQLiteUnique(err) || isSQLitePrimaryKey(err) {
			return domain.ErrDeviceBridgeConflict
		}
		return fmt.Errorf("create device bridge device: %w", err)
	}
	return nil
}

func (s *Store) GetDeviceBridgeDevice(ctx context.Context, id domain.DeviceBridgeDeviceID) (domain.DeviceBridgeDevice, bool, error) {
	row, err := s.qr.GetDeviceBridgeDevice(ctx, string(id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.DeviceBridgeDevice{}, false, nil
	}
	if err != nil {
		return domain.DeviceBridgeDevice{}, false, fmt.Errorf("get device bridge device: %w", err)
	}
	rec, err := deviceBridgeDeviceFromGen(row)
	return rec, true, err
}

func (s *Store) GetPairedDeviceBridgeDevice(ctx context.Context, ownerID string) (domain.DeviceBridgeDevice, bool, error) {
	row, err := s.qr.GetPairedDeviceBridgeDevice(ctx, ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.DeviceBridgeDevice{}, false, nil
	}
	if err != nil {
		return domain.DeviceBridgeDevice{}, false, fmt.Errorf("get paired device bridge device: %w", err)
	}
	rec, err := deviceBridgeDeviceFromGen(row)
	return rec, true, err
}

// TransitionDeviceBridgeDevice is the device lifecycle CAS: pairing -> paired,
// pairing|paired -> revoked. The database enforces at most one paired device
// per owner (partial unique index), so a second pairing-to-paired transition
// for the same owner fails here as a conflict, not in application logic.
func (s *Store) TransitionDeviceBridgeDevice(ctx context.Context, id domain.DeviceBridgeDeviceID, to domain.DeviceBridgeState, now time.Time) (bool, error) {
	if now.IsZero() {
		return false, domain.ErrDeviceBridgeInvalid
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	switch to {
	case domain.DeviceBridgeStatePaired:
		n, err := s.qw.TransitionDeviceBridgeDeviceToPaired(ctx, gen.TransitionDeviceBridgeDeviceToPairedParams{Now: sql.NullTime{Time: now.UTC(), Valid: true}, DeviceID: string(id)})
		if err != nil {
			if isSQLiteUnique(err) || isSQLitePrimaryKey(err) {
				return false, domain.ErrDeviceBridgeConflict
			}
			return false, fmt.Errorf("pair device bridge device: %w", err)
		}
		return n > 0, nil
	case domain.DeviceBridgeStateRevoked:
		n, err := s.qw.TransitionDeviceBridgeDeviceToRevoked(ctx, gen.TransitionDeviceBridgeDeviceToRevokedParams{Now: sql.NullTime{Time: now.UTC(), Valid: true}, DeviceID: string(id)})
		if err != nil {
			return false, fmt.Errorf("revoke device bridge device: %w", err)
		}
		return n > 0, nil
	default:
		return false, domain.ErrDeviceBridgeInvalid
	}
}

// TouchDeviceBridgeDeviceLastSeen only moves last_seen forward and only for a
// paired device: a heartbeat never resurrects a revoked device or rewrites
// history.
func (s *Store) TouchDeviceBridgeDeviceLastSeen(ctx context.Context, id domain.DeviceBridgeDeviceID, now time.Time) (bool, error) {
	if now.IsZero() {
		return false, domain.ErrDeviceBridgeInvalid
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	n, err := s.qw.TouchDeviceBridgeDeviceLastSeen(ctx, gen.TouchDeviceBridgeDeviceLastSeenParams{Now: sql.NullTime{Time: now.UTC(), Valid: true}, DeviceID: string(id)})
	if err != nil {
		return false, fmt.Errorf("touch device bridge device last seen: %w", err)
	}
	return n > 0, nil
}

// EnqueueDeviceBridgeOutbox spools one outbound message and returns its
// durable seq cursor. A duplicate message ID or (command ID, revision)
// inserts nothing and is a conflict: the caller reuses the existing entry,
// so a redelivered logical message never spools a second effect.
func (s *Store) EnqueueDeviceBridgeOutbox(ctx context.Context, msg domain.DeviceBridgeOutboxMessage) (int64, error) {
	msg.State = domain.DeviceBridgeOutboxPending
	msg.SentAt = nil
	msg.ReceiptedAt = nil
	if err := msg.Validate(); err != nil {
		return 0, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	n, err := s.qw.EnqueueDeviceBridgeOutbox(ctx, gen.EnqueueDeviceBridgeOutboxParams{
		MessageID: msg.MessageID, CommandID: msg.CommandID, Revision: msg.Revision,
		Class: string(msg.Class), Payload: msg.Payload, CreatedAt: msg.CreatedAt.UTC(),
	})
	if err != nil {
		if isSQLiteUnique(err) || isSQLitePrimaryKey(err) {
			return 0, domain.ErrDeviceBridgeConflict
		}
		return 0, fmt.Errorf("enqueue device bridge outbox: %w", err)
	}
	if n != 1 {
		return 0, domain.ErrDeviceBridgeConflict
	}
	row, err := s.qw.GetDeviceBridgeOutboxByMessageID(ctx, msg.MessageID)
	if err != nil {
		return 0, fmt.Errorf("read spooled device bridge outbox: %w", err)
	}
	return row.Seq, nil
}

func (s *Store) GetDeviceBridgeOutboxByMessageID(ctx context.Context, messageID string) (domain.DeviceBridgeOutboxMessage, bool, error) {
	row, err := s.qr.GetDeviceBridgeOutboxByMessageID(ctx, messageID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.DeviceBridgeOutboxMessage{}, false, nil
	}
	if err != nil {
		return domain.DeviceBridgeOutboxMessage{}, false, fmt.Errorf("get device bridge outbox: %w", err)
	}
	return deviceBridgeOutboxFromGen(row), true, nil
}

// ListPendingDeviceBridgeOutbox is the drain view: unreceipted messages in
// seq order, oldest first. A `sent` entry stays in this view until a backend
// receipt clears it, so a disconnect mid-drain never loses a message whose
// delivery state is unknown.
func (s *Store) ListPendingDeviceBridgeOutbox(ctx context.Context, limit int) ([]domain.DeviceBridgeOutboxMessage, error) {
	if limit < 1 {
		limit = 100
	}
	rows, err := s.qr.ListPendingDeviceBridgeOutbox(ctx, int64(limit))
	if err != nil {
		return nil, fmt.Errorf("list pending device bridge outbox: %w", err)
	}
	msgs := make([]domain.DeviceBridgeOutboxMessage, 0, len(rows))
	for _, row := range rows {
		msgs = append(msgs, deviceBridgeOutboxFromGen(row))
	}
	return msgs, nil
}

func (s *Store) MarkDeviceBridgeOutboxSent(ctx context.Context, seq int64, now time.Time) (bool, error) {
	if now.IsZero() {
		return false, domain.ErrDeviceBridgeInvalid
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	n, err := s.qw.MarkDeviceBridgeOutboxSent(ctx, gen.MarkDeviceBridgeOutboxSentParams{Now: sql.NullTime{Time: now.UTC(), Valid: true}, Seq: seq})
	if err != nil {
		return false, fmt.Errorf("mark device bridge outbox sent: %w", err)
	}
	return n > 0, nil
}

// MarkDeviceBridgeOutboxReceipted is the only path that clears an entry from
// drain: the backend's durable receipt, never a local send.
func (s *Store) MarkDeviceBridgeOutboxReceipted(ctx context.Context, seq int64, now time.Time) (bool, error) {
	if now.IsZero() {
		return false, domain.ErrDeviceBridgeInvalid
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	n, err := s.qw.MarkDeviceBridgeOutboxReceipted(ctx, gen.MarkDeviceBridgeOutboxReceiptedParams{Now: sql.NullTime{Time: now.UTC(), Valid: true}, Seq: seq})
	if err != nil {
		return false, fmt.Errorf("mark device bridge outbox receipted: %w", err)
	}
	return n > 0, nil
}

func (s *Store) CountPendingDeviceBridgeOutbox(ctx context.Context) (int64, error) {
	return s.qr.CountPendingDeviceBridgeOutbox(ctx)
}

// RecordDeviceBridgeInbound journals one inbound command replay-safely. The
// first record wins. A redelivery carrying the same idempotency key and the
// same payload fingerprint returns the existing record with created=false -
// one effect, no duplicate work. The same key with a different fingerprint is
// a conflict and is never last-writer-wins.
func (s *Store) RecordDeviceBridgeInbound(ctx context.Context, cmd domain.DeviceBridgeInboxCommand) (domain.DeviceBridgeInboxCommand, bool, error) {
	if cmd.State != domain.DeviceBridgeInboxReceived {
		return domain.DeviceBridgeInboxCommand{}, false, domain.ErrDeviceBridgeInvalid
	}
	if err := cmd.Validate(); err != nil {
		return domain.DeviceBridgeInboxCommand{}, false, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	n, err := s.qw.InsertDeviceBridgeInbound(ctx, gen.InsertDeviceBridgeInboundParams{
		CommandID: cmd.CommandID, Revision: cmd.Revision, IdempotencyKey: cmd.IdempotencyKey,
		PayloadFingerprint: cmd.PayloadFingerprint.String(), Class: string(cmd.Class), Payload: cmd.Payload,
		State: string(cmd.State), CreatedAt: cmd.CreatedAt.UTC(), UpdatedAt: cmd.UpdatedAt.UTC(),
	})
	if err != nil {
		return domain.DeviceBridgeInboxCommand{}, false, fmt.Errorf("record device bridge inbound: %w", err)
	}
	row, err := s.qw.GetDeviceBridgeInboundByIdempotencyKey(ctx, cmd.IdempotencyKey)
	if n == 0 && errors.Is(err, sql.ErrNoRows) {
		return domain.DeviceBridgeInboxCommand{}, false, domain.ErrDeviceBridgeConflict
	}
	if err != nil {
		return domain.DeviceBridgeInboxCommand{}, false, fmt.Errorf("read device bridge inbound: %w", err)
	}
	stored, err := deviceBridgeInboundFromGen(row)
	if err != nil {
		return domain.DeviceBridgeInboxCommand{}, false, err
	}
	if n == 1 {
		return stored, true, nil
	}
	if stored.CommandID != cmd.CommandID || stored.Revision != cmd.Revision || stored.Class != cmd.Class || stored.PayloadFingerprint != cmd.PayloadFingerprint {
		return domain.DeviceBridgeInboxCommand{}, false, domain.ErrDeviceBridgeConflict
	}
	return stored, false, nil
}

func (s *Store) GetDeviceBridgeInboundByIdempotencyKey(ctx context.Context, key string) (domain.DeviceBridgeInboxCommand, bool, error) {
	row, err := s.qr.GetDeviceBridgeInboundByIdempotencyKey(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.DeviceBridgeInboxCommand{}, false, nil
	}
	if err != nil {
		return domain.DeviceBridgeInboxCommand{}, false, fmt.Errorf("get device bridge inbound: %w", err)
	}
	rec, err := deviceBridgeInboundFromGen(row)
	return rec, true, err
}

// TransitionDeviceBridgeInbound is the journal lifecycle CAS over the closed
// transitions: received -> acked/rejected/expired, acked -> resulted.
func (s *Store) TransitionDeviceBridgeInbound(ctx context.Context, commandID string, revision int64, to domain.DeviceBridgeInboxState, rejectReason string, now time.Time) (bool, error) {
	if strings.TrimSpace(commandID) == "" || revision < 1 || now.IsZero() {
		return false, domain.ErrDeviceBridgeInvalid
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	switch to {
	case domain.DeviceBridgeInboxAcked, domain.DeviceBridgeInboxRejected, domain.DeviceBridgeInboxExpired:
		if to == domain.DeviceBridgeInboxRejected && rejectReason == "" {
			return false, domain.ErrDeviceBridgeInvalid
		}
		if to != domain.DeviceBridgeInboxRejected {
			rejectReason = ""
		}
		n, err := s.qw.TransitionDeviceBridgeInboundReceived(ctx, gen.TransitionDeviceBridgeInboundReceivedParams{
			ToState: string(to), RejectReason: rejectReason, Now: now.UTC(), CommandID: commandID, Revision: revision,
		})
		if err != nil {
			return false, fmt.Errorf("transition device bridge inbound: %w", err)
		}
		return n > 0, nil
	case domain.DeviceBridgeInboxResulted:
		n, err := s.qw.TransitionDeviceBridgeInboundAcked(ctx, gen.TransitionDeviceBridgeInboundAckedParams{Now: now.UTC(), CommandID: commandID, Revision: revision})
		if err != nil {
			return false, fmt.Errorf("transition device bridge inbound: %w", err)
		}
		return n > 0, nil
	default:
		return false, domain.ErrDeviceBridgeInvalid
	}
}

func deviceBridgeDeviceInsert(r domain.DeviceBridgeDevice) gen.InsertDeviceBridgeDeviceParams {
	classes, _ := json.Marshal(r.CapabilityClasses)
	return gen.InsertDeviceBridgeDeviceParams{
		DeviceID: string(r.DeviceID), OwnerID: r.OwnerID, Label: r.Label, PublicKey: r.PublicKey,
		KeyCustodyRef: r.KeyCustodyRef, ContractVersion: r.ContractVersion, CapabilityClasses: string(classes),
		State: string(r.State), PairedAt: timePtrToNull(r.PairedAt), RevokedAt: timePtrToNull(r.RevokedAt), LastSeenAt: timePtrToNull(r.LastSeenAt),
		CreatedAt: r.CreatedAt.UTC(), UpdatedAt: r.UpdatedAt.UTC(),
	}
}

func deviceBridgeDeviceFromGen(row gen.DeviceBridgeDevice) (domain.DeviceBridgeDevice, error) {
	var classes []string
	if err := json.Unmarshal([]byte(row.CapabilityClasses), &classes); err != nil {
		return domain.DeviceBridgeDevice{}, fmt.Errorf("decode device bridge capabilities: %w", err)
	}
	rec := domain.DeviceBridgeDevice{
		DeviceID: domain.DeviceBridgeDeviceID(row.DeviceID), OwnerID: row.OwnerID, Label: row.Label,
		PublicKey: row.PublicKey, KeyCustodyRef: row.KeyCustodyRef, ContractVersion: row.ContractVersion,
		CapabilityClasses: classes, State: domain.DeviceBridgeState(row.State),
		PairedAt: nullTimeToPtr(row.PairedAt), RevokedAt: nullTimeToPtr(row.RevokedAt), LastSeenAt: nullTimeToPtr(row.LastSeenAt),
		CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	}
	if err := rec.Validate(); err != nil {
		return domain.DeviceBridgeDevice{}, err
	}
	return rec, nil
}

func deviceBridgeOutboxFromGen(row gen.DeviceBridgeOutbox) domain.DeviceBridgeOutboxMessage {
	return domain.DeviceBridgeOutboxMessage{
		Seq: row.Seq, MessageID: row.MessageID, CommandID: row.CommandID, Revision: row.Revision,
		Class: domain.DeviceBridgeMessageClass(row.Class), Payload: row.Payload, State: domain.DeviceBridgeOutboxState(row.State),
		CreatedAt: row.CreatedAt.UTC(), SentAt: nullTimeToPtr(row.SentAt), ReceiptedAt: nullTimeToPtr(row.ReceiptedAt),
	}
}

func deviceBridgeInboundFromGen(row gen.DeviceBridgeInboxJournal) (domain.DeviceBridgeInboxCommand, error) {
	rec := domain.DeviceBridgeInboxCommand{
		CommandID: row.CommandID, Revision: row.Revision, IdempotencyKey: row.IdempotencyKey,
		PayloadFingerprint: domain.SHA256Digest(row.PayloadFingerprint), Class: domain.DeviceBridgeMessageClass(row.Class),
		Payload: row.Payload, State: domain.DeviceBridgeInboxState(row.State), RejectReason: row.RejectReason,
		CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	}
	if err := rec.Validate(); err != nil {
		return domain.DeviceBridgeInboxCommand{}, err
	}
	return rec, nil
}
