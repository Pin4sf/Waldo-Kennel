package devicebridge

import (
	"crypto/rand"
	"io"
	"time"
)

// NewMessageID creates an uppercase 26-character ULID for a new logical
// frame. Resends reuse the old ID and refresh only signing fields.
func NewMessageID(now time.Time, random io.Reader) (string, error) {
	ms := now.UnixMilli()
	if ms < 0 || uint64(ms) >= 1<<48 {
		return "", ErrInvalidSigningInput
	}
	if random == nil {
		random = rand.Reader
	}
	var bytes [16]byte
	for i := 5; i >= 0; i-- {
		bytes[i] = byte(ms)
		ms >>= 8
	}
	if _, err := io.ReadFull(random, bytes[6:]); err != nil {
		return "", err
	}
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	var encoded [ULIDLength]byte
	for i := 0; i < len(encoded); i++ {
		var digit byte
		for j := 0; j < 5; j++ {
			bit := i*5 + j - 2
			digit <<= 1
			if bit >= 0 {
				digit |= (bytes[bit/8] >> (7 - uint(bit%8))) & 1
			}
		}
		encoded[i] = alphabet[digit]
	}
	return string(encoded[:]), nil
}
