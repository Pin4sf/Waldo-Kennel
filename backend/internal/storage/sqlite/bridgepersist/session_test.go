package bridgepersist_test

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/bridgepersist"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
)

func bridgeCommand(t *testing.T, s devicebridge.Scope, now time.Time, id string) []byte {
	t.Helper()
	raw, e := json.Marshal(map[string]any{"class": "machine_state_query", "command_id": id, "contract_version": "0.2.3", "device_id": s.DeviceID, "expires_at": now.Unix() + 600, "idempotency_key": id, "message_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV", "owner_id": s.OwnerID, "payload": map[string]any{"query_id": "q", "query_kind": "session_status"}, "revision": 1, "type": "command"})
	if e != nil {
		t.Fatal(e)
	}
	return raw
}
func bridgeResult(t *testing.T, s devicebridge.Scope, now time.Time, id string) []byte {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"command_id": id, "contract_version": "0.2.3", "device_id": s.DeviceID, "idempotency_key": id, "message_id": "01ARZ3NDEKTSV4RRFFQ69G5FAX", "owner_id": s.OwnerID, "payload": map[string]any{"status": "answered", "answer": map[string]any{"query_id": "q", "query_kind": "session_status", "state": "unknown"}}, "revision": 1, "type": "result"})
	key := ed25519.NewKeyFromSeed(make([]byte, 32))
	signed, e := devicebridge.SignFrame(raw, key, now, [devicebridge.NonceBytes]byte{})
	if e != nil {
		t.Fatal(e)
	}
	return signed
}
func bridgeReceipt(t *testing.T, s devicebridge.Scope, id, msg string) []byte {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"command_id": id, "contract_version": "0.2.3", "device_id": s.DeviceID, "idempotency_key": id, "message_id": msg, "owner_id": s.OwnerID, "payload": map[string]any{"received_at": 1, "result_message_id": "01ARZ3NDEKTSV4RRFFQ69G5FAX"}, "revision": 1, "type": "receipt"})
	return raw
}
func TestGateBSessionRestartDedupReceiptRevocation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, e := openBridge(t, dir)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now().UTC().Truncate(time.Second)
	scopes := []devicebridge.Scope{{DeviceID: "dev1", OwnerID: "owner1"}, {DeviceID: "dev2", OwnerID: "owner2"}}
	for _, scope := range scopes {
		if e = s.CreateDeviceBridgeDevice(ctx, deviceBridgeTestDevice(now, scope.DeviceID, scope.OwnerID)); e != nil {
			t.Fatal(e)
		}
		if _, e = s.TransitionDeviceBridgeDevice(ctx, domain.DeviceBridgeDeviceID(scope.DeviceID), domain.DeviceBridgeStatePaired, now); e != nil {
			t.Fatal(e)
		}
		c := bridgeCommand(t, scope, now, "cmd")
		r := bridgeResult(t, scope, now, "cmd")
		a, e := s.Admit(ctx, scope, c, now)
		if e != nil || !a.New {
			t.Fatalf("admit: %+v %v", a, e)
		}
		if e = s.CommitResult(ctx, scope, c, r); e != nil {
			t.Fatal(e)
		}
		if e = s.CommitResult(ctx, scope, c, r); e != nil {
			t.Fatal("repeat result:", e)
		}
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = openBridge(t, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	b := s
	scope := scopes[0]
	p, e := b.Pending(ctx, scope)
	if e != nil || len(p) != 1 {
		t.Fatalf("restart pending %v %v", p, e)
	}
	a, e := b.Admit(ctx, scope, bridgeCommand(t, scope, now, "cmd"), now)
	if e != nil || a.New {
		t.Fatalf("restart dedup %+v %v", a, e)
	}
	r := bridgeReceipt(t, scope, "cmd", "01ARZ3NDEKTSV4RRFFQ69G5FAY")
	wrong := bridgeReceipt(t, scope, "wrong", "01ARZ3NDEKTSV4RRFFQ69G5FAY")
	if e = b.ApplyReceipt(ctx, scope, wrong, now); e == nil {
		t.Fatal("wrong receipt accepted")
	}
	if n, e := b.Depth(ctx, scope); e != nil || n != 1 {
		t.Fatalf("rollback depth %d %v", n, e)
	}
	if e = b.ApplyReceipt(ctx, scope, r, now); e != nil {
		t.Fatal(e)
	}
	if e = b.ApplyReceipt(ctx, scope, r, now); e != nil {
		t.Fatal("duplicate:", e)
	}
	if n, e := b.Depth(ctx, scope); e != nil || n != 0 {
		t.Fatalf("receipt depth %d %v", n, e)
	}
	if n, e := b.Depth(ctx, scopes[1]); e != nil || n != 1 {
		t.Fatalf("cross device isolation %d %v", n, e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = openBridge(t, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	b = s
	if e = b.ApplyReceipt(ctx, scope, r, now); e != nil {
		t.Fatal("restart tombstone:", e)
	}
	a, e = b.Admit(ctx, scope, bridgeCommand(t, scope, now, "cmd"), now)
	if e != nil || a.New {
		t.Fatal("journal redelivery", a, e)
	}
	if n, e := b.Depth(ctx, scope); e != nil || n != 1 {
		t.Fatal("journaled result not re-spooled", n, e)
	}
	// Re-receipt under a fresh backend message ID also succeeds.
	if e = b.ApplyReceipt(ctx, scope, bridgeReceipt(t, scope, "cmd", "01ARZ3NDEKTSV4RRFFQ69G5FAZ"), now); e != nil {
		t.Fatal(e)
	}
	if e = b.ClearPaired(ctx, scope); e != nil {
		t.Fatal(e)
	}
	if _, e = b.Depth(ctx, scope); !errors.Is(e, domain.ErrDeviceBridgeInvalid) {
		t.Fatalf("revoked readable: %v", e)
	}
	if d, ok, e := s.GetDeviceBridgeDevice(ctx, domain.DeviceBridgeDeviceID(scope.DeviceID)); e != nil || !ok || d.State != domain.DeviceBridgeStateRevoked {
		t.Fatalf("audit retained: %+v %v", d, e)
	}
	if _, e = s.CountPendingDeviceBridgeOutbox(ctx); !errors.Is(e, domain.ErrDeviceBridgeInvalid) {
		t.Fatal("legacy APIs still active")
	}
}
func TestGateBSessionConflictExpiredAndUnresulted(t *testing.T) {
	ctx := context.Background()
	s, err := openBridge(t, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Now().UTC().Truncate(time.Second)
	scope := devicebridge.Scope{DeviceID: "dev", OwnerID: "owner"}
	if e := s.CreateDeviceBridgeDevice(ctx, deviceBridgeTestDevice(now, "dev", "owner")); e != nil {
		t.Fatal(e)
	}
	_, _ = s.TransitionDeviceBridgeDevice(ctx, "dev", domain.DeviceBridgeStatePaired, now)
	b := s
	c := bridgeCommand(t, scope, now, "cmd")
	a, e := b.Admit(ctx, scope, c, now)
	if e != nil || !a.New {
		t.Fatal(a, e)
	}
	rows, e := b.AcceptedUnresulted(ctx, scope)
	if e != nil || len(rows) != 1 {
		t.Fatal(rows, e)
	}
	var f map[string]any
	_ = json.Unmarshal(c, &f)
	f["command_id"] = "different"
	raw, _ := json.Marshal(f)
	a, e = b.Admit(ctx, scope, raw, now)
	if e != nil || a.Reason != devicebridge.ReasonIdempotencyConflict {
		t.Fatal("message id conflict", a, e)
	}
	a, e = b.Admit(ctx, scope, c, now.Add(601*time.Second))
	if e != nil || a.State != devicebridge.AckExpired {
		t.Fatal("expiry", a, e)
	}
	// Command redelivery with a new message ID still maps to the same outcome.
	_ = json.Unmarshal(c, &f)
	f["message_id"] = "01ARZ3NDEKTSV4RRFFQ69G5FB0"
	raw, _ = json.Marshal(f)
	a, e = b.Admit(ctx, scope, raw, now)
	if e != nil || a.New || a.State != devicebridge.AckAccepted {
		t.Fatal("new message dedup", a, e)
	}
	// No new identities outside authenticated scope.
	if _, e = b.Admit(ctx, devicebridge.Scope{DeviceID: "dev", OwnerID: "wrong"}, c, now); e == nil {
		t.Fatal("wrong owner accepted")
	}
	t.Log(fmt.Sprintf("accepted frame survives durable journal; conflict/expiry/scope checked"))
}

func openBridge(t *testing.T, dir string) (*bridgepersist.BridgeSession, error) {
	t.Helper()
	migration, e := sqlite.Open(dir)
	if e != nil {
		return nil, e
	}
	if e = migration.Close(); e != nil {
		return nil, e
	}
	db, e := sql.Open("sqlite", "file:"+filepath.Join(dir, "kennel.db")+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)")
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	return bridgepersist.New(db), nil
}
func deviceBridgeTestDevice(now time.Time, id, owner string) domain.DeviceBridgeDevice {
	return domain.DeviceBridgeDevice{DeviceID: domain.DeviceBridgeDeviceID(id), OwnerID: owner, Label: "Mac", PublicKey: strings.Repeat("ab", 32), KeyCustodyRef: "custody", ContractVersion: "0.2.3", CapabilityClasses: []string{"machine_state_query"}, State: domain.DeviceBridgeStatePairing, CreatedAt: now, UpdatedAt: now}
}

func TestGateBRetainedResultForS4PostReceiptReplay(t *testing.T) {
	ctx := context.Background()
	s, e := openBridge(t, t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	now := time.Now().UTC().Truncate(time.Second)
	scope := devicebridge.Scope{DeviceID: "dev", OwnerID: "owner"}
	if e = s.CreateDeviceBridgeDevice(ctx, deviceBridgeTestDevice(now, "dev", "owner")); e != nil {
		t.Fatal(e)
	}
	_, e = s.TransitionDeviceBridgeDevice(ctx, "dev", domain.DeviceBridgeStatePaired, now)
	if e != nil {
		t.Fatal(e)
	}
	command := bridgeCommand(t, scope, now, "cmd")
	result := bridgeResult(t, scope, now, "cmd")
	if _, found, e := s.ResultForCommand(ctx, scope, command); e != nil || found {
		t.Fatal("unknown result", found, e)
	}
	a, e := s.Admit(ctx, scope, command, now)
	if e != nil || !a.New {
		t.Fatal(a, e)
	}
	if _, found, e := s.ResultForCommand(ctx, scope, command); e != nil || found {
		t.Fatal("accepted unresulted", found, e)
	}
	if e = s.CommitResult(ctx, scope, command, result); e != nil {
		t.Fatal(e)
	}
	if e = s.ApplyReceipt(ctx, scope, bridgeReceipt(t, scope, "cmd", "01ARZ3NDEKTSV4RRFFQ69G5FAY"), now); e != nil {
		t.Fatal(e)
	}
	retained, found, e := s.ResultForCommand(ctx, scope, command)
	if e != nil || !found || string(retained) != string(result) {
		t.Fatal("post receipt replay", found, e)
	}
	var frame map[string]any
	_ = json.Unmarshal(command, &frame)
	frame["message_id"] = "01ARZ3NDEKTSV4RRFFQ69G5FB0"
	redelivery, _ := json.Marshal(frame)
	a, e = s.Admit(ctx, scope, redelivery, now)
	if e != nil || a.New {
		t.Fatal("duplicate would re-execute", a, e)
	}
	retained, found, e = s.ResultForCommand(ctx, scope, redelivery)
	if e != nil || !found || string(retained) != string(result) {
		t.Fatal("new message post receipt replay", found, e)
	}
	frame["expires_at"] = now.Unix() + 601
	conflict, _ := json.Marshal(frame)
	if _, _, e = s.ResultForCommand(ctx, scope, conflict); !errors.Is(e, domain.ErrDeviceBridgeConflict) {
		t.Fatal("widened command replay", e)
	}
	if _, _, e = s.ResultForCommand(ctx, devicebridge.Scope{DeviceID: "dev", OwnerID: "wrong"}, command); e == nil {
		t.Fatal("wrong owner replay")
	}
}
