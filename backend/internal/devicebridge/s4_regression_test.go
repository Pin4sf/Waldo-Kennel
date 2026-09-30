package devicebridge

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"io"
	"testing"
	"time"
)

type s4RegressionStore struct {
	*mockSessionStore
	admitted bool
	retained []byte
}

func (s *s4RegressionStore) Admit(ctx context.Context, scope Scope, raw []byte, now time.Time) (Admission, error) {
	first := !s.admitted
	s.admitted = true
	return Admission{State: AckAccepted, New: first}, nil
}
func (s *s4RegressionStore) CommitResult(ctx context.Context, scope Scope, command, result []byte) error {
	s.retained = append([]byte(nil), result...)
	return s.mockSessionStore.CommitResult(ctx, scope, command, result)
}
func (s *s4RegressionStore) ResultForCommand(ctx context.Context, scope Scope, command []byte) ([]byte, bool, error) {
	if scope != s.scope {
		return nil, false, ErrTerminalFrame
	}
	return append([]byte(nil), s.retained...), len(s.retained) > 0, nil
}

func TestS4RegressionRedeliveryAfterReceiptReusesResult(t *testing.T) {
	c, base := sessionFixture(t)
	store := &s4RegressionStore{mockSessionStore: base}
	c.Store = store
	socket := &mockSocket{}
	if err := c.process(context.Background(), socket, []byte(commandFixture)); err != nil {
		t.Fatal(err)
	}
	if len(socket.frames) != 2 {
		t.Fatalf("initial ack/result frame count=%d", len(socket.frames))
	}
	original, err := VerifyFrame(socket.frames[1], c.Key.Public().(ed25519.PublicKey), c.clock())
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := parseJSONObject([]byte(receiptFixture))
	if err != nil {
		t.Fatal(err)
	}
	receipt["payload"].(map[string]any)["result_message_id"] = original["message_id"]
	raw, err := marshalCanonical(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.process(context.Background(), socket, raw); err != nil {
		t.Fatal(err)
	}
	if len(base.pending) != 0 {
		t.Fatal("matching receipt did not clear outbox")
	}
	if err := c.process(context.Background(), socket, []byte(commandFixture)); err != nil {
		t.Fatal(err)
	}
	if len(socket.frames) != 4 {
		t.Fatalf("redelivery after receipt must emit ack plus retained result; total frames=%d, want4", len(socket.frames))
	}
	resend, err := VerifyFrame(socket.frames[3], c.Key.Public().(ed25519.PublicKey), c.clock())
	if err != nil {
		t.Fatal(err)
	}
	a, _ := LogicalFingerprint(original)
	b, _ := LogicalFingerprint(resend)
	if a != b || original["nonce"] == resend["nonce"] {
		t.Fatal("retained result identity or fresh nonce violated")
	}
	effects := 0
	for _, event := range base.events {
		if event == "handle" {
			effects++
		}
	}
	if effects != 1 {
		t.Fatalf("handler calls=%d, want1", effects)
	}
}

func TestS4RegressionStartupDrainDoesNotResendSameRowTwice(t *testing.T) {
	c, store := sessionFixture(t)
	store.pending = []PendingResult{{Seq: 1, Frame: signedResult(t, c)}}
	socket := &mockSocket{}
	if err := c.RunConnection(context.Background(), socket); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	results := 0
	for _, raw := range socket.frames {
		if bytes.Contains(raw, []byte(`"type":"result"`)) {
			results++
		}
	}
	if results != 1 {
		t.Fatalf("same startup outbox row sent %d times, want1", results)
	}
}
