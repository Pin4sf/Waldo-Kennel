package devicebridge

import (
	"crypto/ed25519"
	"encoding/json"
	"io"
	"strconv"
	"time"
)

// NewVolatileAck creates a newly identified, signed ack for this send. It is
// never passed to the durable result outbox; a lost ack is repaired by backend
// redelivery and the journaled command state.
func NewVolatileAck(command map[string]any, state, reason string, key ed25519.PrivateKey, now time.Time, random io.Reader) ([]byte, error) {
	if validateFrameShape(command, false) != nil || command["type"] != TypeCommand {
		return nil, ErrInvalidFrameShape
	}
	messageID, err := NewMessageID(now, random)
	if err != nil {
		return nil, err
	}
	nonce, err := NewNonce(random)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{"state": state}
	if reason != "" {
		payload["reason"] = reason
	}
	frame := map[string]any{
		"contract_version": ContractVersion, "type": TypeAck, "message_id": messageID,
		"device_id": command["device_id"], "owner_id": command["owner_id"],
		"command_id": command["command_id"], "revision": command["revision"],
		"idempotency_key": command["idempotency_key"], "payload": payload,
	}
	if err := validateUnsignedDeviceShape(frame); err != nil {
		return nil, ErrInvalidFrameShape
	}
	unsigned, err := marshalCanonical(frame)
	if err != nil {
		return nil, err
	}
	return SignFrame(unsigned, key, now, nonce)
}

// NewHeartbeat creates one volatile heartbeat, including honest unreceipted
// outbox depth, with a new message identity on every send.
func NewHeartbeat(deviceID, ownerID string, capabilities []string, depth int64, key ed25519.PrivateKey, now time.Time, random io.Reader) ([]byte, error) {
	if !validIdentifier(deviceID) || !validIdentifier(ownerID) || !validCapabilities(capabilities) || depth < 0 || depth > OutboxDepthMax {
		return nil, ErrInvalidFrameShape
	}
	messageID, err := NewMessageID(now, random)
	if err != nil {
		return nil, err
	}
	nonce, err := NewNonce(random)
	if err != nil {
		return nil, err
	}
	capabilitiesJSON := make([]any, len(capabilities))
	for i, capability := range capabilities {
		capabilitiesJSON[i] = capability
	}
	frame := map[string]any{
		"contract_version": ContractVersion, "type": TypeHeartbeat, "message_id": messageID,
		"device_id": deviceID, "owner_id": ownerID,
		"payload": map[string]any{"declared_capabilities": capabilitiesJSON, "outbox_depth": json.Number(strconv.FormatInt(depth, 10))},
	}
	unsigned, err := marshalCanonical(frame)
	if err != nil {
		return nil, err
	}
	return SignFrame(unsigned, key, now, nonce)
}
