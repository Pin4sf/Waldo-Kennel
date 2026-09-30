package devicebridge

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestFrameSigningCanonicalAndRejectsTampering(t *testing.T) {
	seed := bytes.Repeat([]byte{7}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	var nonce [16]byte
	for i := range nonce {
		nonce[i] = byte(i + 1)
	}
	now := time.Unix(1790200800, 0)
	unsigned := []byte(`{"contract_version":"0.2.3","device_id":"dev_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAZ","owner_id":"owner_1","payload":{"declared_capabilities":["machine_state_query"],"outbox_depth":2},"type":"heartbeat"}`)
	signed, err := SignFrame(unsigned, privateKey, now, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(signed, []byte(`{"contract_version":"0.2.3","device_id":"dev_1"`)) || bytes.HasSuffix(signed, []byte("\n")) {
		t.Fatal("signed frame is not compact canonical JSON")
	}
	verified, err := VerifyFrame(signed, publicKey, now)
	if err != nil || verified["message_id"] != "01ARZ3NDEKTSV4RRFFQ69G5FAZ" {
		t.Fatalf("valid frame verification = %v, %v", verified, err)
	}
	tampered := bytes.Replace(signed, []byte(`"outbox_depth":2`), []byte(`"outbox_depth":3`), 1)
	if _, err := VerifyFrame(tampered, publicKey, now); !errors.Is(err, ErrInvalidFrameSignature) {
		t.Fatalf("tampered frame error = %v", err)
	}
	if _, err := VerifyFrame(signed, publicKey, now.Add(301*time.Second)); !errors.Is(err, ErrInvalidFrameSignature) {
		t.Fatalf("stale frame error = %v", err)
	}
}

func TestFrameGoldenSignatures(t *testing.T) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	key := ed25519.NewKeyFromSeed(seed)
	var nonce [NonceBytes]byte
	for i := range nonce {
		nonce[i] = byte(i)
	}
	for _, tc := range []struct{ name, unsigned, signature string }{
		{"ack", `{"command_id":"cmd_1","contract_version":"0.2.3","device_id":"dev_1","idempotency_key":"idem_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAW","owner_id":"owner_1","payload":{"state":"accepted"},"revision":1,"type":"ack"}`, "c2GwY1y5gWNCSPdf9Ufb8IIRgMGx_t9wpOi02pbKlHcC-QdOb7i7R5wp-cwmk-u2Lb-mO6ah2hugIFYflTQNBw"},
		{"result", `{"command_id":"cmd_1","contract_version":"0.2.3","device_id":"dev_1","idempotency_key":"idem_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAX","owner_id":"owner_1","payload":{"answer":{"query_id":"query_1","query_kind":"session_status","state":"idle"},"status":"answered"},"revision":1,"type":"result"}`, "Ur-XO9PNQy3Kszxe7A_SJp7GssLVSKp6wkZUqEZAYHkKGon0IqbLhA-vhD1ya_o-HQQkb2wOB9Sgyyd8PWpDAA"},
		{"heartbeat", `{"contract_version":"0.2.3","device_id":"dev_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAZ","owner_id":"owner_1","payload":{"declared_capabilities":["machine_state_query","notify_local"],"outbox_depth":0},"type":"heartbeat"}`, "1DzrSOApCr5iU_sTuJFozlndCY-A1EhY0XVtYGf07nP_OoW_Q8bI1a9OakAvlfJBBfBQb1NDevyyH4Q3tbWzDA"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := SignFrame([]byte(tc.unsigned), key, time.Unix(1790200800, 0), nonce)
			if err != nil {
				t.Fatal(err)
			}
			var frame map[string]any
			if err := json.Unmarshal(wire, &frame); err != nil {
				t.Fatal(err)
			}
			if frame["signature"] != tc.signature {
				t.Fatalf("signature = %v", frame["signature"])
			}
			sig, _ := base64.RawURLEncoding.DecodeString(tc.signature)
			parsed, err := parseJSONObject(wire)
			if err != nil {
				t.Fatal(err)
			}
			base, err := frameBaseString(parsed)
			if err != nil || !ed25519.Verify(key.Public().(ed25519.PublicKey), []byte(base), sig) {
				t.Fatal("golden signature verification failed")
			}
			if _, err := VerifyFrame(wire, key.Public().(ed25519.PublicKey), time.Unix(1790200800, 0)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCanonicalJSONRejectsAmbiguity(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte(`{"x":1.0}`), []byte(`{"x":1e0}`), []byte(`{"x":"\ud800"}`),
		[]byte("{\"x\":\"\xff\"}"), []byte(`{"x":1,"x":2}`),
	} {
		value, err := parseJSONObject(raw)
		if err == nil {
			canonical, err := marshalCanonical(value)
			if err == nil && bytes.Equal(raw, canonical) {
				t.Fatalf("accepted %q", raw)
			}
		}
	}
	value, err := parseJSONObject([]byte(`{"𐀀":"plane","":"private","é":"café","é":"combining","text":"<>&/\n\t\"\\"}`))
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := marshalCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := hex.DecodeString("7b2265cc81223a22636f6d62696e696e67222c2274657874223a223c3e262f5c6e5c745c225c5c222c22c3a9223a22636166c3a9222c22ee8080223a2270726976617465222c22f0908080223a22706c616e65227d")
	if !bytes.Equal(canonical, want) {
		t.Fatalf("canonical = %x", canonical)
	}
}

func TestFrameSigningRejectsDuplicateKeysAtAnyDepth(t *testing.T) {
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, ed25519.SeedSize))
	now := time.Unix(1790200800, 0)
	var nonce [16]byte
	for _, raw := range [][]byte{
		[]byte(`{"type":"heartbeat","type":"result","message_id":"m","payload":{}}`),
		[]byte(`{"type":"heartbeat","message_id":"m","payload":{"a":1,"a":2}}`),
	} {
		if _, err := SignFrame(raw, key, now, nonce); !errors.Is(err, ErrInvalidSigningInput) {
			t.Fatalf("duplicate-key sign error = %v", err)
		}
	}
}

func TestFrameSignatureChangesWithFreshNonce(t *testing.T) {
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{3}, ed25519.SeedSize))
	now := time.Unix(1790200800, 0)
	unsigned := []byte(`{"command_id":"cmd_1","contract_version":"0.2.3","device_id":"dev_1","idempotency_key":"idem_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAX","owner_id":"owner_1","payload":{"answer":{"query_id":"query_1","query_kind":"session_status","state":"idle"},"status":"answered"},"revision":1,"type":"result"}`)
	var nonceA, nonceB [16]byte
	nonceB[0] = 1
	a, err := SignFrame(unsigned, key, now, nonceA)
	if err != nil {
		t.Fatal(err)
	}
	b, err := SignFrame(unsigned, key, now, nonceB)
	if err != nil {
		t.Fatal(err)
	}
	var frameA, frameB map[string]any
	if err := json.Unmarshal(a, &frameA); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &frameB); err != nil {
		t.Fatal(err)
	}
	if frameA["message_id"] != frameB["message_id"] || frameA["signature"] == frameB["signature"] {
		t.Fatal("fresh nonce must change signature without changing logical message identity")
	}
}
