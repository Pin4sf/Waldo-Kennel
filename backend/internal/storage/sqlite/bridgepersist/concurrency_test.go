package bridgepersist_test

import (
	"context"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"sync"
	"testing"
	"time"
)

func TestGateBConcurrentAdmissionOneWinner(t *testing.T) {
	dir := t.TempDir()
	a, e := openBridge(t, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	now := time.Now().UTC().Truncate(time.Second)
	ctx := context.Background()
	scope := devicebridge.Scope{DeviceID: "dev", OwnerID: "owner"}
	if e = a.CreateDeviceBridgeDevice(ctx, deviceBridgeTestDevice(now, "dev", "owner")); e != nil {
		t.Fatal(e)
	}
	_, e = a.TransitionDeviceBridgeDevice(ctx, "dev", domain.DeviceBridgeStatePaired, now)
	if e != nil {
		t.Fatal(e)
	}
	b, e := openBridge(t, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	var wg sync.WaitGroup
	results := make(chan devicebridge.Admission, 2)
	errs := make(chan error, 2)
	for _, store := range []interface {
		Admit(context.Context, devicebridge.Scope, []byte, time.Time) (devicebridge.Admission, error)
	}{a, b} {
		wg.Add(1)
		go func(store interface {
			Admit(context.Context, devicebridge.Scope, []byte, time.Time) (devicebridge.Admission, error)
		}) {
			defer wg.Done()
			r, e := store.Admit(ctx, scope, bridgeCommand(t, scope, now, "cmd"), now)
			results <- r
			errs <- e
		}(store)
	}
	wg.Wait()
	close(results)
	close(errs)
	wins := 0
	for r := range results {
		if r.New {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("admission winners=%d", wins)
	}
	// SQLite can fail a contending deferred transaction with BUSY; it must not
	// claim admission. The caller retries admission, never the local effect.
	for e := range errs {
		if e != nil {
			t.Log("fail-closed contention:", e)
		}
	}
	r, e := b.Admit(ctx, scope, bridgeCommand(t, scope, now, "cmd"), now)
	if e != nil || r.New {
		t.Fatal("post-contention replay", r, e)
	}
}
