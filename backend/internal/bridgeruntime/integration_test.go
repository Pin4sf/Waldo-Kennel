package bridgeruntime

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"errors"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/bridgeactivation"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/secretstore"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/activationrepo"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/bridgepersist"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func realIdentity(t *testing.T) (*activationrepo.Repository, *bridgepersist.BridgeSession, domain.DeviceBridgeDevice, *secretstore.DeviceKeyStore, string) {
	t.Helper()
	dir := t.TempDir()
	s, e := sqlite.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	open := func() *sql.DB {
		db, e := activationrepo.OpenDedicated(dir)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { db.Close() })
		return db
	}
	repo := activationrepo.New(open())
	store := bridgepersist.New(open())
	keys := secretstore.NewDeviceKeyStore(dir)
	key := ed25519.NewKeyFromSeed(make([]byte, 32))
	ref, e := keys.Put(context.Background(), key)
	if e != nil {
		t.Fatal(e)
	}
	_, d := factoryHarness(t)
	d.PublicKey = base64.RawURLEncoding.EncodeToString(key.Public().(ed25519.PublicKey))
	d.KeyCustodyRef = ref
	r, e := repo.Reserve(context.Background(), d.OwnerID)
	if e != nil {
		t.Fatal(e)
	}
	if e = repo.Complete(context.Background(), d.OwnerID, r.Attempt, d); e != nil {
		t.Fatal(e)
	}
	return repo, store, d, keys, filepath.Join(dir, "secrets", "device-keys")
}

type noPair struct{}

func (noPair) Pair(context.Context, devicebridge.PairRequest) (domain.DeviceBridgeDevice, error) {
	return domain.DeviceBridgeDevice{}, bridgeactivation.ErrBlocked
}

type controlledSocket struct {
	drop   <-chan struct{}
	writes chan []byte
}

func (s controlledSocket) Read(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.drop:
		return nil, io.EOF
	}
}
func (s controlledSocket) Write(ctx context.Context, raw []byte) error {
	if s.writes != nil {
		select {
		case s.writes <- append([]byte(nil), raw...):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
func (s controlledSocket) Close() error { return nil }
func waitPhase(t *testing.T, c *bridgeactivation.Controller, want bridgeactivation.Phase) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s, e := c.Status(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		if s.Phase == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("phase did not become %s", want)
}
func TestStateCallbackDrivesStatusTruthfully(t *testing.T) {
	repo, store, d, keys, dir := realIdentity(t)
	drop := make(chan struct{})
	sleeping := make(chan struct{})
	resume := make(chan struct{})
	var calls atomic.Int32
	f := NewFactory(Dependencies{Fence: repo, Keys: keys, SessionStore: store, KeyDirectory: dir, ReadyProbe: func(context.Context) error { return nil }, Dialer: dialFunc(func(context.Context, string, http.Header) (devicebridge.Socket, error) {
		if calls.Add(1) == 1 {
			return controlledSocket{drop: drop}, nil
		}
		return nil, devicebridge.ErrAuthenticationRejected
	}), Sleep: func(ctx context.Context, _ time.Duration) error {
		close(sleeping)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-resume:
			return nil
		}
	}})
	c, e := bridgeactivation.New(d.OwnerID, "https://example.test", repo, noPair{}, f)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Stop(context.Background())
	waitPhase(t, c, bridgeactivation.Offline)
	if e = c.Start(context.Background()); e != nil {
		t.Fatal(e)
	}
	waitPhase(t, c, bridgeactivation.Online)
	close(drop)
	<-sleeping
	waitPhase(t, c, bridgeactivation.Offline)
	close(resume)
	waitPhase(t, c, bridgeactivation.Unpaired)
	if e = c.Stop(context.Background()); e != nil {
		t.Fatal(e)
	}
	r, e := repo.Load(context.Background(), d.OwnerID)
	if e != nil || r.Device.State != domain.DeviceBridgeStateRevoked {
		t.Fatal("revocation not durable")
	}
}

type memoryRecord struct {
	mu     sync.Mutex
	record bridgeactivation.Record
}

func (m *memoryRecord) Load(context.Context, string) (bridgeactivation.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.record, nil
}
func (m *memoryRecord) Reserve(context.Context, string) (bridgeactivation.Record, error) {
	return bridgeactivation.Record{}, bridgeactivation.ErrBlocked
}
func (m *memoryRecord) Complete(context.Context, string, string, domain.DeviceBridgeDevice) error {
	return bridgeactivation.ErrBlocked
}
func (m *memoryRecord) Block(context.Context, string, string, *devicebridge.PairingRecoveryError) error {
	return bridgeactivation.ErrBlocked
}
func (m *memoryRecord) Revoke(context.Context, string, domain.DeviceBridgeDeviceID) error {
	return bridgeactivation.ErrBlocked
}

type lifecycleFactory struct{ active, max atomic.Int32 }

func (f *lifecycleFactory) Ready(context.Context) error { return nil }
func (f *lifecycleFactory) New(context.Context, string, domain.DeviceBridgeDevice, func(devicebridge.ConnectionState)) (bridgeactivation.Session, error) {
	return lifecycleSession{f}, nil
}

type lifecycleSession struct{ f *lifecycleFactory }

func (s lifecycleSession) Run(ctx context.Context) error {
	n := s.f.active.Add(1)
	defer s.f.active.Add(-1)
	for {
		old := s.f.max.Load()
		if n <= old || s.f.max.CompareAndSwap(old, n) {
			break
		}
	}
	<-ctx.Done()
	return nil
}
func TestStartStopRestartRace(t *testing.T) {
	before := runtime.NumGoroutine()
	_, d := factoryHarness(t)
	repo := &memoryRecord{record: bridgeactivation.Record{OwnerID: d.OwnerID, Device: d}}
	f := &lifecycleFactory{}
	c, e := bridgeactivation.New(d.OwnerID, "https://example.test", repo, noPair{}, f)
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = c.Start(context.Background())
			if e := c.Stop(ctx); e != nil {
				t.Error(e)
			}
			_ = c.Start(context.Background())
		}()
	}
	wg.Wait()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if e = c.Stop(ctx); e != nil {
		t.Fatal(e)
	}
	if f.max.Load() > 1 || f.active.Load() != 0 {
		t.Fatal("double or surviving Run")
	}
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before+1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if runtime.NumGoroutine() > before+1 {
		t.Fatal("goroutine leak")
	}
}
func TestRevokedIdentityNeverReconnects(t *testing.T) {
	repo, store, d, keys, dir := realIdentity(t)
	if e := repo.Revoke(context.Background(), d.OwnerID, d.DeviceID); e != nil {
		t.Fatal(e)
	}
	var calls atomic.Int32
	f := NewFactory(Dependencies{Fence: repo, Keys: keys, SessionStore: store, KeyDirectory: dir, ReadyProbe: func(context.Context) error { return nil }, Dialer: dialFunc(func(context.Context, string, http.Header) (devicebridge.Socket, error) {
		calls.Add(1)
		return nil, errors.New("offline")
	})})
	c, e := bridgeactivation.New(d.OwnerID, "https://example.test", repo, noPair{}, f)
	if e != nil {
		t.Fatal(e)
	}
	if c.Start(context.Background()) == nil || calls.Load() != 0 {
		t.Fatal("revoked reconnect")
	}
}
func TestNoSecretsInRuntimeLogs(t *testing.T) {
	repo, store, d, keys, dir := realIdentity(t)
	var logs bytes.Buffer
	private := "recognisable-private-key-marker"
	code := strings.Repeat("A", 43)
	f := NewFactory(Dependencies{Fence: repo, Keys: keys, SessionStore: store, KeyDirectory: dir, ReadyProbe: func(context.Context) error { return nil }, Logger: slog.New(slog.NewTextHandler(&logs, nil)), Dialer: dialFunc(func(context.Context, string, http.Header) (devicebridge.Socket, error) {
		return nil, errors.Join(devicebridge.ErrAuthenticationRejected, errors.New(private+code))
	})})
	c, e := bridgeactivation.New(d.OwnerID, "https://example.test", repo, noPair{}, f)
	if e != nil {
		t.Fatal(e)
	}
	if e = c.Start(context.Background()); e != nil {
		t.Fatal(e)
	}
	waitPhase(t, c, bridgeactivation.Unpaired)
	if e = c.Stop(context.Background()); e != nil {
		if strings.Contains(e.Error(), private) || strings.Contains(e.Error(), code) {
			t.Fatal("error leaked")
		}
	}
	if strings.Contains(logs.String(), private) || strings.Contains(logs.String(), code) {
		t.Fatal("log leaked")
	}
}

// Diagnostic for the failed B4 status transition: reproduce the transaction
// interaction without retrying the failed lifecycle gate or changing production.
func TestActivationReadWriterLockConflictsWithClearPaired(t *testing.T) {
	repo, store, d, _, keyDir := realIdentity(t)
	db, e := activationrepo.OpenDedicated(filepath.Dir(filepath.Dir(keyDir)))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	conn, e := db.Conn(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	if _, e = conn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); e != nil {
		t.Fatal(e)
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")
	// This is precisely Load's BEGIN IMMEDIATE + read, with no writes.
	var state string
	if e = conn.QueryRowContext(context.Background(), `SELECT phase FROM device_bridge_activations WHERE owner_id=?`, d.OwnerID).Scan(&state); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	e = store.ClearPaired(ctx, devicebridge.Scope{OwnerID: d.OwnerID, DeviceID: string(d.DeviceID)})
	if e == nil {
		t.Fatal("expected read-to-write lock conflict")
	}
	t.Logf("ClearPaired while activation read owns writer lock: %v", e)
	if _, rollbackErr := conn.ExecContext(context.Background(), "ROLLBACK"); rollbackErr != nil {
		t.Fatal(rollbackErr)
	}
	record, loadErr := repo.Load(context.Background(), d.OwnerID)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if record.Device.State != domain.DeviceBridgeStatePaired {
		t.Fatal("failed clear changed identity")
	}
}
