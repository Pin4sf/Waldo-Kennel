package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func deviceBridgeTestDevice(now time.Time, id, owner string) domain.DeviceBridgeDevice {
	return domain.DeviceBridgeDevice{
		DeviceID: domain.DeviceBridgeDeviceID(id), OwnerID: owner, Label: "Test Mac",
		PublicKey: strings.Repeat("cd", 32), KeyCustodyRef: "secretstore:device-bridge/" + id,
		ContractVersion: "0.1",
		CapabilityClasses: []string{
			string(domain.DeviceBridgeClassMachineStateQuery),
			string(domain.DeviceBridgeClassNotifyLocal),
		},
		State: domain.DeviceBridgeStatePairing, CreatedAt: now, UpdatedAt: now,
	}
}

func TestDeviceBridgeDeviceLifecycleAndOnePairedPerOwner(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := s.CreateDeviceBridgeDevice(ctx, deviceBridgeTestDevice(now, "dev-1", "owner-1")); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.CreateDeviceBridgeDevice(ctx, deviceBridgeTestDevice(now, "dev-1", "owner-1")); !errors.Is(err, domain.ErrDeviceBridgeConflict) {
		t.Fatalf("duplicate device id must conflict, got %v", err)
	}
	got, found, err := s.GetDeviceBridgeDevice(ctx, "dev-1")
	if err != nil || !found || got.State != domain.DeviceBridgeStatePairing {
		t.Fatalf("get: found=%v state=%v err=%v", found, got.State, err)
	}
	if _, found, _ := s.GetPairedDeviceBridgeDevice(ctx, "owner-1"); found {
		t.Fatal("no paired device before transition")
	}
	won, err := s.TransitionDeviceBridgeDevice(ctx, "dev-1", domain.DeviceBridgeStatePaired, now.Add(time.Minute))
	if err != nil || !won {
		t.Fatalf("pair: won=%v err=%v", won, err)
	}
	if won, _ := s.TransitionDeviceBridgeDevice(ctx, "dev-1", domain.DeviceBridgeStatePaired, now.Add(2*time.Minute)); won {
		t.Fatal("second pair transition must lose the CAS")
	}
	paired, found, err := s.GetPairedDeviceBridgeDevice(ctx, "owner-1")
	if err != nil || !found || paired.PairedAt == nil {
		t.Fatalf("paired lookup: found=%v pairedAt=%v err=%v", found, paired.PairedAt, err)
	}

	// A second device for the same owner cannot reach paired while dev-1 is live.
	if err := s.CreateDeviceBridgeDevice(ctx, deviceBridgeTestDevice(now, "dev-2", "owner-1")); err != nil {
		t.Fatalf("create second: %v", err)
	}
	if _, err := s.TransitionDeviceBridgeDevice(ctx, "dev-2", domain.DeviceBridgeStatePaired, now.Add(3*time.Minute)); !errors.Is(err, domain.ErrDeviceBridgeConflict) {
		t.Fatalf("second paired device per owner must conflict at the database, got %v", err)
	}

	// Heartbeat moves last_seen forward and never resurrects a revoked device.
	touched, err := s.TouchDeviceBridgeDeviceLastSeen(ctx, "dev-1", now.Add(4*time.Minute))
	if err != nil || !touched {
		t.Fatalf("heartbeat: touched=%v err=%v", touched, err)
	}
	if touched, _ := s.TouchDeviceBridgeDeviceLastSeen(ctx, "dev-1", now.Add(3*time.Minute)); touched {
		t.Fatal("heartbeat must never move last_seen backward")
	}
	if won, _ := s.TransitionDeviceBridgeDevice(ctx, "dev-1", domain.DeviceBridgeStateRevoked, now.Add(5*time.Minute)); !won {
		t.Fatal("revoke should win")
	}
	if touched, _ := s.TouchDeviceBridgeDeviceLastSeen(ctx, "dev-1", now.Add(6*time.Minute)); touched {
		t.Fatal("heartbeat on a revoked device must not apply")
	}
	if _, found, _ := s.GetPairedDeviceBridgeDevice(ctx, "owner-1"); found {
		t.Fatal("revoked device must not read as paired")
	}
}

func TestDeviceBridgeOutboxDrainReceiptAndConflicts(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	enqueue := func(id, cmd string, rev int64) int64 {
		seq, err := s.EnqueueDeviceBridgeOutbox(ctx, domain.DeviceBridgeOutboxMessage{
			MessageID: id, CommandID: cmd, Revision: rev, Class: domain.DeviceBridgeClassResult,
			Payload: `{"status":"answered"}`, CreatedAt: now,
		})
		if err != nil {
			t.Fatalf("enqueue %s: %v", id, err)
		}
		return seq
	}
	seq1 := enqueue("msg-1", "cmd-1", 1)
	seq2 := enqueue("msg-2", "cmd-1", 2)
	seq3 := enqueue("msg-3", "cmd-2", 1)
	if !(seq1 < seq2 && seq2 < seq3) {
		t.Fatalf("seq must be monotonic: %d %d %d", seq1, seq2, seq3)
	}
	if _, err := s.EnqueueDeviceBridgeOutbox(ctx, domain.DeviceBridgeOutboxMessage{
		MessageID: "msg-1", CommandID: "cmd-9", Revision: 1, Class: domain.DeviceBridgeClassAck,
		Payload: `{}`, CreatedAt: now,
	}); !errors.Is(err, domain.ErrDeviceBridgeConflict) {
		t.Fatalf("duplicate message id must conflict, got %v", err)
	}
	if _, err := s.EnqueueDeviceBridgeOutbox(ctx, domain.DeviceBridgeOutboxMessage{
		MessageID: "msg-4", CommandID: "cmd-1", Revision: 1, Class: domain.DeviceBridgeClassAck,
		Payload: `{}`, CreatedAt: now,
	}); !errors.Is(err, domain.ErrDeviceBridgeConflict) {
		t.Fatalf("duplicate (command, revision) must conflict, got %v", err)
	}

	pending, err := s.ListPendingDeviceBridgeOutbox(ctx, 10)
	if err != nil || len(pending) != 3 || pending[0].MessageID != "msg-1" || pending[2].MessageID != "msg-3" {
		t.Fatalf("drain order: %+v err=%v", pending, err)
	}
	if depth, _ := s.CountPendingDeviceBridgeOutbox(ctx); depth != 3 {
		t.Fatalf("depth=%d", depth)
	}
	if sent, _ := s.MarkDeviceBridgeOutboxSent(ctx, seq1, now.Add(time.Minute)); !sent {
		t.Fatal("mark sent should win")
	}
	// Sent is not delivered: the entry stays in the drain view until receipt.
	if pending, _ := s.ListPendingDeviceBridgeOutbox(ctx, 10); len(pending) != 3 {
		t.Fatalf("sent entry must stay in drain until receipt, len=%d", len(pending))
	}
	if ok, _ := s.MarkDeviceBridgeOutboxReceipted(ctx, seq1, now.Add(2*time.Minute)); !ok {
		t.Fatal("receipt should win")
	}
	if ok, _ := s.MarkDeviceBridgeOutboxReceipted(ctx, seq1, now.Add(3*time.Minute)); ok {
		t.Fatal("second receipt must lose the CAS")
	}
	pending, _ = s.ListPendingDeviceBridgeOutbox(ctx, 10)
	if len(pending) != 2 || pending[0].MessageID != "msg-2" {
		t.Fatalf("drain after receipt: %+v", pending)
	}
	got, found, _ := s.GetDeviceBridgeOutboxByMessageID(ctx, "msg-1")
	if !found || got.State != domain.DeviceBridgeOutboxReceipted || got.ReceiptedAt == nil {
		t.Fatalf("receipted read-back: %+v", got)
	}
}

func TestDeviceBridgeInboundReplaySafety(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	fp := domain.DigestSHA256([]byte(`{"query_kind":"session_status"}`))
	cmd := domain.DeviceBridgeInboxCommand{
		CommandID: "cmd-1", Revision: 1, IdempotencyKey: "idem-1",
		PayloadFingerprint: fp, Class: domain.DeviceBridgeClassMachineStateQuery,
		Payload: `{"query_kind":"session_status"}`, State: domain.DeviceBridgeInboxReceived,
		CreatedAt: now, UpdatedAt: now,
	}
	stored, created, err := s.RecordDeviceBridgeInbound(ctx, cmd)
	if err != nil || !created || stored.IdempotencyKey != "idem-1" {
		t.Fatalf("first record: created=%v err=%v", created, err)
	}
	invalidInitialState := cmd
	invalidInitialState.State = domain.DeviceBridgeInboxAcked
	if _, _, err := s.RecordDeviceBridgeInbound(ctx, invalidInitialState); !errors.Is(err, domain.ErrDeviceBridgeInvalid) {
		t.Fatalf("new inbound command must start received, got %v", err)
	}
	// Identical redelivery maps to the same row: one effect.
	again, created, err := s.RecordDeviceBridgeInbound(ctx, cmd)
	if err != nil || created || again.CommandID != stored.CommandID {
		t.Fatalf("replay: created=%v err=%v", created, err)
	}
	// Same key, different fingerprint: conflict, never last-writer-wins.
	conflicting := cmd
	conflicting.Payload = `{"query_kind":"attempt_status"}`
	conflicting.PayloadFingerprint = domain.DigestSHA256([]byte(conflicting.Payload))
	if _, _, err := s.RecordDeviceBridgeInbound(ctx, conflicting); !errors.Is(err, domain.ErrDeviceBridgeConflict) {
		t.Fatalf("conflicting replay must reject, got %v", err)
	}
	otherIdentity := cmd
	otherIdentity.CommandID = "cmd-other"
	if _, _, err := s.RecordDeviceBridgeInbound(ctx, otherIdentity); !errors.Is(err, domain.ErrDeviceBridgeConflict) {
		t.Fatalf("same idempotency key with another command must conflict, got %v", err)
	}
	otherKey := cmd
	otherKey.IdempotencyKey = "idem-other"
	if _, _, err := s.RecordDeviceBridgeInbound(ctx, otherKey); !errors.Is(err, domain.ErrDeviceBridgeConflict) {
		t.Fatalf("same command revision with another idempotency key must conflict, got %v", err)
	}

	if won, _ := s.TransitionDeviceBridgeInbound(ctx, "cmd-1", 1, domain.DeviceBridgeInboxAcked, "", now.Add(time.Minute)); !won {
		t.Fatal("received -> acked should win")
	}
	if won, _ := s.TransitionDeviceBridgeInbound(ctx, "cmd-1", 1, domain.DeviceBridgeInboxAcked, "", now.Add(2*time.Minute)); won {
		t.Fatal("second acked transition must lose the CAS")
	}
	if won, _ := s.TransitionDeviceBridgeInbound(ctx, "cmd-1", 1, domain.DeviceBridgeInboxResulted, "", now.Add(3*time.Minute)); !won {
		t.Fatal("acked -> resulted should win")
	}
	got, found, _ := s.GetDeviceBridgeInboundByIdempotencyKey(ctx, "idem-1")
	if !found || got.State != domain.DeviceBridgeInboxResulted {
		t.Fatalf("journal read-back: %+v", got)
	}

	// Rejected requires a reason and is terminal from received.
	cmd2 := cmd
	cmd2.CommandID, cmd2.IdempotencyKey = "cmd-2", "idem-2"
	if _, _, err := s.RecordDeviceBridgeInbound(ctx, cmd2); err != nil {
		t.Fatalf("record cmd-2: %v", err)
	}
	if _, err := s.TransitionDeviceBridgeInbound(ctx, "cmd-2", 1, domain.DeviceBridgeInboxRejected, "", now.Add(time.Minute)); !errors.Is(err, domain.ErrDeviceBridgeInvalid) {
		t.Fatalf("reject without reason must be invalid, got %v", err)
	}
	if won, _ := s.TransitionDeviceBridgeInbound(ctx, "cmd-2", 1, domain.DeviceBridgeInboxRejected, "expired", now.Add(time.Minute)); !won {
		t.Fatal("received -> rejected should win")
	}
}

func TestDeviceBridgeStateSurvivesRestart(t *testing.T) {
	dataDir := t.TempDir()
	s, err := sqlitetest.Open(dataDir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := s.CreateDeviceBridgeDevice(ctx, deviceBridgeTestDevice(now, "dev-r", "owner-r")); err != nil {
		t.Fatalf("create: %v", err)
	}
	if won, _ := s.TransitionDeviceBridgeDevice(ctx, "dev-r", domain.DeviceBridgeStatePaired, now.Add(time.Minute)); !won {
		t.Fatal("pair should win")
	}
	seq, err := s.EnqueueDeviceBridgeOutbox(ctx, domain.DeviceBridgeOutboxMessage{
		MessageID: "msg-r", CommandID: "cmd-r", Revision: 1, Class: domain.DeviceBridgeClassResult,
		Payload: `{"status":"delivered"}`, CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	fp := domain.DigestSHA256([]byte("{}"))
	if _, _, err := s.RecordDeviceBridgeInbound(ctx, domain.DeviceBridgeInboxCommand{
		CommandID: "cmd-r", Revision: 0, IdempotencyKey: "idem-bad", PayloadFingerprint: fp,
		Class: domain.DeviceBridgeClassNotifyLocal, Payload: `{}",`,
		State: domain.DeviceBridgeInboxReceived, CreatedAt: now, UpdatedAt: now,
	}); err == nil {
		t.Fatal("revision 0 must fail validation before insert")
	}
	if _, _, err := s.RecordDeviceBridgeInbound(ctx, domain.DeviceBridgeInboxCommand{
		CommandID: "cmd-r", Revision: 1, IdempotencyKey: "idem-r", PayloadFingerprint: fp,
		Class: domain.DeviceBridgeClassNotifyLocal, Payload: `{"title":"hi"}`,
		State: domain.DeviceBridgeInboxReceived, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("record inbound: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	dev, found, err := reopened.GetPairedDeviceBridgeDevice(ctx, "owner-r")
	if err != nil || !found || dev.State != domain.DeviceBridgeStatePaired || dev.PairedAt == nil {
		t.Fatalf("device after restart: found=%v state=%v err=%v", found, dev.State, err)
	}
	pending, err := reopened.ListPendingDeviceBridgeOutbox(ctx, 10)
	if err != nil || len(pending) != 1 || pending[0].Seq != seq {
		t.Fatalf("outbox after restart: %+v err=%v", pending, err)
	}
	cmd, found, err := reopened.GetDeviceBridgeInboundByIdempotencyKey(ctx, "idem-r")
	if err != nil || !found || cmd.State != domain.DeviceBridgeInboxReceived {
		t.Fatalf("journal after restart: found=%v err=%v", found, err)
	}
	// A replayed delivery after restart still maps to the same row.
	if _, created, err := reopened.RecordDeviceBridgeInbound(ctx, domain.DeviceBridgeInboxCommand{
		CommandID: "cmd-r", Revision: 1, IdempotencyKey: "idem-r", PayloadFingerprint: fp,
		Class: domain.DeviceBridgeClassNotifyLocal, Payload: `{"title":"hi"}`,
		State: domain.DeviceBridgeInboxReceived, CreatedAt: now, UpdatedAt: now,
	}); err != nil || created {
		t.Fatalf("post-restart replay must not create a second effect: created=%v err=%v", created, err)
	}
}
