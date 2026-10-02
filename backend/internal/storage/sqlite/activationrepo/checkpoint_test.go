package activationrepo

import (
	"errors"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"reflect"
	"strings"
	"testing"
)

func safeCheckpoint(owner string) devicebridge.PairingRecoveryError {
	return devicebridge.PairingRecoveryError{OwnerID: owner, KeyCustodyRef: "opaque", PublicKey: "public"}
}
func TestCheckpointStoresRefKeepsPending(t *testing.T) {
	dir := t.TempDir()
	a, b := connections(t, dir)
	r := New(a)
	v := reserve(t, r, "owner")
	cp := safeCheckpoint("owner")
	must(t, r.Checkpoint(bg, "owner", v.Attempt, cp))
	must(t, a.Close())
	must(t, b.Close())
	a, _ = connections(t, dir)
	got := load(t, New(a), "owner")
	if got.Phase != "pending" || got.Attempt != v.Attempt || !reflect.DeepEqual(got.Checkpoint, &cp) {
		t.Fatalf("lost checkpoint %+v", got)
	}
}
func TestCheckpointWrongAttemptOrOwnerRejected(t *testing.T) {
	for _, kind := range []string{"token", "owner", "cp-owner", "blocked", "paired", "revoked", "empty-ref", "empty-key"} {
		t.Run(kind, func(t *testing.T) {
			r, _ := repos(t)
			v := reserve(t, r, "owner")
			cp := safeCheckpoint("owner")
			owner, attempt := "owner", v.Attempt
			switch kind {
			case "token":
				attempt = "wrong"
			case "owner":
				owner = "other"
			case "cp-owner":
				cp.OwnerID = "other"
			case "blocked":
				must(t, r.Block(bg, "owner", v.Attempt, nil))
			case "paired", "revoked":
				d := pairedDevice("owner")
				must(t, r.Complete(bg, "owner", v.Attempt, d))
				if kind == "revoked" {
					must(t, r.Revoke(bg, "owner", d.DeviceID))
				}
			case "empty-ref":
				cp.KeyCustodyRef = ""
			case "empty-key":
				cp.PublicKey = ""
			}
			before := snapshot(t, r.db)
			if r.Checkpoint(bg, owner, attempt, cp) == nil {
				t.Fatal("accepted")
			}
			if !reflect.DeepEqual(before, snapshot(t, r.db)) {
				t.Fatal("changed")
			}
		})
	}
}
func TestCheckpointSecondDifferentRefRefused(t *testing.T) {
	r, _ := repos(t)
	v := reserve(t, r, "owner")
	cp := safeCheckpoint("owner")
	must(t, r.Checkpoint(bg, "owner", v.Attempt, cp))
	before := snapshot(t, r.db)
	must(t, r.Checkpoint(bg, "owner", v.Attempt, cp))
	if !reflect.DeepEqual(before, snapshot(t, r.db)) {
		t.Fatal("idempotency changed row")
	}
	cp.KeyCustodyRef = "other"
	want(t, r.Checkpoint(bg, "owner", v.Attempt, cp), domain.ErrDeviceBridgeConflict)
	if !reflect.DeepEqual(before, snapshot(t, r.db)) {
		t.Fatal("overwrote key")
	}
}
func TestCheckpointNeverStoresCodeOrPrivateKey(t *testing.T) {
	r, _ := repos(t)
	v := reserve(t, r, "owner")
	cp := safeCheckpoint("owner")
	code := strings.Repeat("A", 43)
	private := "RECOGNISABLE-PRIVATE-KEY-MATERIAL"
	cp.Cause = errors.New(code + private)
	cp.DeviceID = code
	must(t, r.Checkpoint(bg, "owner", v.Attempt, cp))
	seed(t, r, "paired")
	for _, s := range snapshot(t, r.db) {
		if strings.Contains(s, code) || strings.Contains(s, private) {
			t.Fatal("secret stored")
		}
	}
}
