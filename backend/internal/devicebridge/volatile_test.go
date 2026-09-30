package devicebridge

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"testing"
	"time"
)

func TestVolatileAckAndHeartbeatUseFreshMessageIDs(t *testing.T) {
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{8}, ed25519.SeedSize))
	command, err := ParseBackendFrame([]byte(commandFixture))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1790200800, 0)
	a, err := NewVolatileAck(command, AckAccepted, "", key, now, bytes.NewReader(bytes.Repeat([]byte{1}, 26)))
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewVolatileAck(command, AckAccepted, "", key, now, bytes.NewReader(bytes.Repeat([]byte{2}, 26)))
	if err != nil {
		t.Fatal(err)
	}
	fa, err := VerifyFrame(a, key.Public().(ed25519.PublicKey), now)
	if err != nil {
		t.Fatal(err)
	}
	fb, err := VerifyFrame(b, key.Public().(ed25519.PublicKey), now)
	if err != nil || fa["message_id"] == fb["message_id"] {
		t.Fatal("new ack reused message ID")
	}
	if _, err := NewVolatileAck(command, AckAccepted, ReasonExpired, key, now, bytes.NewReader(bytes.Repeat([]byte{1}, 26))); !errors.Is(err, ErrInvalidFrameShape) {
		t.Fatal("accepted ack with reason")
	}
	heartbeat, err := NewHeartbeat("dev_1", "owner_1", []string{ClassMachineStateQuery}, 3, key, now, bytes.NewReader(bytes.Repeat([]byte{3}, 26)))
	if err != nil {
		t.Fatal(err)
	}
	verified, err := VerifyFrame(heartbeat, key.Public().(ed25519.PublicKey), now)
	if err != nil || verified["type"] != TypeHeartbeat {
		t.Fatal("heartbeat verification failed")
	}
}
