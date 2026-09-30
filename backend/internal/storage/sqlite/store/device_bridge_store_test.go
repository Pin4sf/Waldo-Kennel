package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
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

// The old port cannot carry device identity. After 0164 every unscoped
// outbox/journal entry point is disabled, rather than guessing a paired device.
func TestDeviceBridgeLegacyPortsFailClosed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	check := func(err error) {
		t.Helper()
		if !errors.Is(err, domain.ErrDeviceBridgeInvalid) {
			t.Fatalf("unscoped method: %v", err)
		}
	}
	_, e := s.EnqueueDeviceBridgeOutbox(ctx, domain.DeviceBridgeOutboxMessage{})
	check(e)
	_, _, e = s.GetDeviceBridgeOutboxByMessageID(ctx, "id")
	check(e)
	_, e = s.ListPendingDeviceBridgeOutbox(ctx, 10)
	check(e)
	_, e = s.MarkDeviceBridgeOutboxSent(ctx, 1, now)
	check(e)
	_, e = s.MarkDeviceBridgeOutboxReceipted(ctx, 1, now)
	check(e)
	_, e = s.CountPendingDeviceBridgeOutbox(ctx)
	check(e)
	_, _, e = s.RecordDeviceBridgeInbound(ctx, domain.DeviceBridgeInboxCommand{})
	check(e)
	_, _, e = s.GetDeviceBridgeInboundByIdempotencyKey(ctx, "key")
	check(e)
	_, e = s.TransitionDeviceBridgeInbound(ctx, "cmd", 1, domain.DeviceBridgeInboxAcked, "", now)
	check(e)
}
