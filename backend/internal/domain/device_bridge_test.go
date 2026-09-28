package domain

import (
	"strings"
	"testing"
	"time"
)

var deviceBridgeDigest = SHA256Digest(strings.Repeat("ab", 32))

func deviceBridgeTestDevice(now time.Time) DeviceBridgeDevice {
	return DeviceBridgeDevice{
		DeviceID: "dev-1", OwnerID: "owner-1", Label: "Ashish's MacBook",
		PublicKey: strings.Repeat("cd", 32), KeyCustodyRef: "secretstore:device-bridge/dev-1",
		ContractVersion: "0.1", CapabilityClasses: []string{string(DeviceBridgeClassMachineStateQuery), string(DeviceBridgeClassNotifyLocal)},
		State: DeviceBridgeStatePairing, CreatedAt: now, UpdatedAt: now,
	}
}

func TestDeviceBridgeDeviceValidateLifecycleFields(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	d := deviceBridgeTestDevice(now)
	if err := d.Validate(); err != nil {
		t.Fatalf("pairing device should validate: %v", err)
	}
	d.State = DeviceBridgeStatePaired
	if err := d.Validate(); err != ErrDeviceBridgeInvalid {
		t.Fatalf("paired without PairedAt must reject, got %v", err)
	}
	paired := now.Add(time.Minute)
	d.PairedAt = &paired
	if err := d.Validate(); err != nil {
		t.Fatalf("paired device should validate: %v", err)
	}
	revoked := now.Add(2 * time.Minute)
	d.State = DeviceBridgeStateRevoked
	d.RevokedAt = &revoked
	if err := d.Validate(); err != nil {
		t.Fatalf("revoked device should validate: %v", err)
	}
	d.RevokedAt = nil
	if err := d.Validate(); err != ErrDeviceBridgeInvalid {
		t.Fatalf("revoked without RevokedAt must reject, got %v", err)
	}
}

func TestDeviceBridgeDeviceRejectsSecretLikeCustodyOmissions(t *testing.T) {
	now := time.Now().UTC()
	d := deviceBridgeTestDevice(now)
	d.KeyCustodyRef = ""
	if err := d.Validate(); err != ErrDeviceBridgeInvalid {
		t.Fatalf("missing custody ref must reject, got %v", err)
	}
	d = deviceBridgeTestDevice(now)
	d.CapabilityClasses = []string{"shell"}
	if err := d.Validate(); err != ErrDeviceBridgeInvalid {
		t.Fatalf("unknown capability class must reject, got %v", err)
	}
}

func TestDeviceBridgeMessageClassDirection(t *testing.T) {
	if !DeviceBridgeClassMachineStateQuery.InboundCommand() || !DeviceBridgeClassNotifyLocal.InboundCommand() {
		t.Fatal("query and notify must be inbound commands")
	}
	if DeviceBridgeClassAck.InboundCommand() || DeviceBridgeClassResult.InboundCommand() || DeviceBridgeClassReceipt.InboundCommand() {
		t.Fatal("ack/result/receipt must not be journaled inbound commands")
	}
	if !DeviceBridgeClassAck.OutboxMessage() || !DeviceBridgeClassResult.OutboxMessage() {
		t.Fatal("ack/result must be outbox messages")
	}
	if DeviceBridgeClassMachineStateQuery.OutboxMessage() || DeviceBridgeClassReceipt.OutboxMessage() {
		t.Fatal("query/receipt must not spool outbound")
	}
	if DeviceBridgeMessageClass("approve").Valid() {
		t.Fatal("no approval class exists in v0.1")
	}
}

func TestDeviceBridgeOutboxValidate(t *testing.T) {
	now := time.Now().UTC()
	m := DeviceBridgeOutboxMessage{
		MessageID: "msg-1", CommandID: "cmd-1", Revision: 1,
		Class: DeviceBridgeClassResult, Payload: `{"status":"answered"}`,
		State: DeviceBridgeOutboxPending, CreatedAt: now,
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("pending outbox message should validate: %v", err)
	}
	m.Class = DeviceBridgeClassMachineStateQuery
	if err := m.Validate(); err != ErrDeviceBridgeInvalid {
		t.Fatalf("inbound class in outbox must reject, got %v", err)
	}
	m.Class = DeviceBridgeClassAck
	m.Revision = 0
	if err := m.Validate(); err != ErrDeviceBridgeInvalid {
		t.Fatalf("revision 0 must reject, got %v", err)
	}
}

func TestDeviceBridgeInboxValidate(t *testing.T) {
	now := time.Now().UTC()
	c := DeviceBridgeInboxCommand{
		CommandID: "cmd-1", Revision: 1, IdempotencyKey: "idem-1",
		PayloadFingerprint: deviceBridgeDigest, Class: DeviceBridgeClassMachineStateQuery,
		Payload: `{"query_kind":"session_status"}`, State: DeviceBridgeInboxReceived,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("inbound command should validate: %v", err)
	}
	c.PayloadFingerprint = ""
	if err := c.Validate(); err != ErrDeviceBridgeInvalid {
		t.Fatalf("missing fingerprint must reject, got %v", err)
	}
	c.PayloadFingerprint = deviceBridgeDigest
	c.Class = DeviceBridgeClassAck
	if err := c.Validate(); err != ErrDeviceBridgeInvalid {
		t.Fatalf("outbound class in inbox must reject, got %v", err)
	}
	c.Class = DeviceBridgeClassMachineStateQuery
	c.State = DeviceBridgeInboxRejected
	if err := c.Validate(); err != ErrDeviceBridgeInvalid {
		t.Fatalf("rejected command without a reason must reject, got %v", err)
	}
	c.RejectReason = "unsupported query"
	if err := c.Validate(); err != nil {
		t.Fatalf("rejected command with a reason should validate: %v", err)
	}
}
