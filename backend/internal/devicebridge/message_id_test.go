package devicebridge

import (
	"bytes"
	"testing"
	"time"
)

func TestNewMessageIDIsULIDAndFreshPerSend(t *testing.T) {
	now := time.UnixMilli(1790200800000)
	a, err := NewMessageID(now, bytes.NewReader(bytes.Repeat([]byte{0}, 10)))
	if err != nil || !validULID(a) {
		t.Fatalf("id = %q, %v", a, err)
	}
	b, err := NewMessageID(now, bytes.NewReader(bytes.Repeat([]byte{1}, 10)))
	if err != nil || !validULID(b) || a == b {
		t.Fatalf("fresh id = %q, %v", b, err)
	}
	if _, err := NewMessageID(time.UnixMilli(-1), nil); err != ErrInvalidSigningInput {
		t.Fatal("negative ULID time accepted")
	}
}
