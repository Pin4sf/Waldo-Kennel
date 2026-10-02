package bridgeruntime

import (
	"context"
	"errors"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/activationrepo"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

type dialFunc func(context.Context, string, http.Header) (devicebridge.Socket, error)

func (f dialFunc) Dial(c context.Context, target string, h http.Header) (devicebridge.Socket, error) {
	return f(c, target, h)
}
func TestSupervisorBackoffBoundedAndResets(t *testing.T) {
	for _, jitter := range []float64{0, 0.5, 1} {
		f, d := factoryHarness(t)
		clock := time.Now()
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		delays := []time.Duration{}
		f.deps.Clock = func() time.Time { return clock }
		f.deps.Jitter = func() float64 { return jitter }
		f.deps.Sleep = func(_ context.Context, pause time.Duration) error {
			delays = append(delays, pause)
			clock = clock.Add(pause)
			if len(delays) == 12 {
				cancel()
			}
			return nil
		}
		raw, e := f.New(ctx, "https://example.test", d, nil)
		if e != nil {
			t.Fatal(e)
		}
		s := raw.(*Session)
		s.connect = func(context.Context) error {
			calls++
			if calls == 11 {
				clock = clock.Add(61 * time.Second)
			}
			return errors.New("offline")
		}
		if e = s.Run(ctx); e != nil {
			t.Fatal(e)
		}
		base := baseBackoff
		for i, pause := range delays {
			if i == 10 {
				base = baseBackoff
			}
			low := time.Duration(float64(base) * 0.8)
			high := time.Duration(float64(base) * 1.2)
			if high > maxBackoff {
				high = maxBackoff
			}
			if pause < low || pause > high || pause > maxBackoff {
				t.Fatalf("delay %d: %v", i, pause)
			}
			base *= 2
			if base > maxBackoff {
				base = maxBackoff
			}
		}
	}
}
func TestSupervisorNeverRetriesAfterAuthRejection(t *testing.T) {
	for _, clearErr := range []error{nil, errors.New("persistence unavailable")} {
		f, d := factoryHarness(t)
		store := f.deps.SessionStore.(*fakeStore)
		store.clearErr = clearErr
		var calls, sleeps, unpaired int
		f.deps.Dialer = dialFunc(func(context.Context, string, http.Header) (devicebridge.Socket, error) {
			calls++
			return nil, devicebridge.ErrAuthenticationRejected
		})
		f.deps.Sleep = func(context.Context, time.Duration) error { sleeps++; return nil }
		raw, e := f.New(context.Background(), "https://example.test", d, func(state devicebridge.ConnectionState) {
			if state == devicebridge.ConnectionUnpaired {
				unpaired++
			}
		})
		if e != nil {
			t.Fatal(e)
		}
		if e = raw.Run(context.Background()); e != nil || calls != 1 || sleeps != 0 || store.cleared != 1 || clearErr == nil && unpaired != 1 || clearErr != nil && unpaired != 0 {
			t.Fatalf("auth retry %v %d %d %d", e, calls, sleeps, unpaired)
		}
	}
}
func TestSupervisorStopsWhenRevoked(t *testing.T) {
	repo, store, d, keys, dir := realIdentity(t)
	f := NewFactory(Dependencies{Fence: repo, Keys: keys, SessionStore: store, KeyDirectory: dir, ReadyProbe: func(context.Context) error { return nil }, Jitter: func() float64 { return 0.5 }})
	var calls int
	f.deps.Dialer = dialFunc(func(context.Context, string, http.Header) (devicebridge.Socket, error) {
		calls++
		return nil, errors.New("offline")
	})
	f.deps.Sleep = func(context.Context, time.Duration) error {
		return repo.Revoke(context.Background(), d.OwnerID, d.DeviceID)
	}
	raw, e := f.New(context.Background(), "https://example.test", d, nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = raw.Run(context.Background()); e != nil || calls != 1 {
		t.Fatal("reconnected revoked identity")
	}
	if _, e = f.New(context.Background(), "https://example.test", d, nil); !errors.Is(e, activationrepo.ErrRevoked) {
		t.Fatal(e)
	}
}
func TestSupervisorStopsOnContextCancelDuringBackoff(t *testing.T) {
	f, d := factoryHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	entered := make(chan struct{})
	f.deps.Dialer = dialFunc(func(context.Context, string, http.Header) (devicebridge.Socket, error) {
		calls.Add(1)
		return nil, errors.New("offline")
	})
	f.deps.Sleep = func(ctx context.Context, pause time.Duration) error {
		close(entered)
		return interruptibleSleep(ctx, pause)
	}
	raw, e := f.New(ctx, "https://example.test", d, nil)
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan struct{})
	go func() { raw.Run(ctx); close(done) }()
	<-entered
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("backoff ignored cancellation")
	}
	if calls.Load() != 1 {
		t.Fatal("extra reconnect")
	}
}
