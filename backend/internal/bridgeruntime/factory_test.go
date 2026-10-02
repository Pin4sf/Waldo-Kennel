package bridgeruntime

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/bridgeactivation"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/activationrepo"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeFence struct {
	claimErr, verifyErr error
	claims              int
	events              *[]string
}

func (f *fakeFence) ClaimActivation(context.Context, string, domain.DeviceBridgeDeviceID) (int64, error) {
	f.claims++
	if f.events != nil {
		*f.events = append(*f.events, "claim")
	}
	return int64(f.claims), f.claimErr
}
func (f *fakeFence) VerifyClaim(context.Context, string, domain.DeviceBridgeDeviceID, int64) error {
	return f.verifyErr
}

type fakeKeys struct {
	key    ed25519.PrivateKey
	err    error
	events *[]string
}

func (k fakeKeys) Get(context.Context, string) (ed25519.PrivateKey, error) {
	if k.events != nil {
		*k.events = append(*k.events, "key")
	}
	return k.key, k.err
}

type fakeStore struct {
	cleared  int
	clearErr error
}

func (s *fakeStore) Pending(context.Context, devicebridge.Scope) ([]devicebridge.PendingResult, error) {
	return nil, nil
}
func (s *fakeStore) Depth(context.Context, devicebridge.Scope) (int64, error) { return 0, nil }
func (s *fakeStore) AcceptedUnresulted(context.Context, devicebridge.Scope) ([][]byte, error) {
	return nil, nil
}
func (s *fakeStore) ResultForCommand(context.Context, devicebridge.Scope, []byte) ([]byte, bool, error) {
	return nil, false, nil
}
func (s *fakeStore) Admit(context.Context, devicebridge.Scope, []byte, time.Time) (devicebridge.Admission, error) {
	return devicebridge.Admission{State: devicebridge.AckAccepted, New: true}, nil
}
func (s *fakeStore) CommitResult(context.Context, devicebridge.Scope, []byte, []byte) error {
	return nil
}
func (s *fakeStore) ApplyReceipt(context.Context, devicebridge.Scope, []byte, time.Time) error {
	return nil
}
func (s *fakeStore) ClearPaired(context.Context, devicebridge.Scope) error {
	s.cleared++
	return s.clearErr
}
func factoryHarness(t *testing.T) (*Factory, domain.DeviceBridgeDevice) {
	t.Helper()
	key := ed25519.NewKeyFromSeed(make([]byte, 32))
	now := time.Now().UTC()
	d := domain.DeviceBridgeDevice{DeviceID: "device", OwnerID: "owner", Label: "Mac", PublicKey: base64.RawURLEncoding.EncodeToString(key.Public().(ed25519.PublicKey)), KeyCustodyRef: "opaque", ContractVersion: devicebridge.ContractVersion, CapabilityClasses: []string{"machine_state_query", "notify_local"}, State: domain.DeviceBridgeStatePaired, PairedAt: &now, CreatedAt: now, UpdatedAt: now}
	return NewFactory(Dependencies{Fence: &fakeFence{}, Keys: fakeKeys{key: key}, SessionStore: &fakeStore{}, ReadyProbe: func(context.Context) error { return nil }, KeyDirectory: t.TempDir()}), d
}
func TestFactoryReadyRequiresAllDependencies(t *testing.T) {
	for _, kind := range []string{"fence", "keys", "store", "probe", "unmigrated", "unreadable", "present"} {
		t.Run(kind, func(t *testing.T) {
			f, _ := factoryHarness(t)
			switch kind {
			case "fence":
				f.deps.Fence = nil
			case "keys":
				f.deps.Keys = nil
			case "store":
				f.deps.SessionStore = nil
			case "probe":
				f.deps.ReadyProbe = nil
			case "unmigrated":
				f.deps.ReadyProbe = func(context.Context) error { return errors.New("no table at " + f.deps.KeyDirectory) }
			case "unreadable":
				f.deps.KeyDirectory += "/missing"
			}
			e := f.Ready(context.Background())
			if kind == "present" {
				if e != nil {
					t.Fatal(e)
				}
			} else if !errors.Is(e, bridgeactivation.ErrNotReady) || strings.Contains(e.Error(), f.deps.KeyDirectory) {
				t.Fatal("unsafe readiness error")
			}
		})
	}
}
func TestFactoryNewClaimsFenceFirst(t *testing.T) {
	for _, e := range []error{nil, activationrepo.ErrRevoked, activationrepo.ErrNotPaired, activationrepo.ErrStaleClaim} {
		f, d := factoryHarness(t)
		events := []string{}
		f.deps.Fence = &fakeFence{claimErr: e, events: &events}
		keys := f.deps.Keys.(fakeKeys)
		keys.events = &events
		f.deps.Keys = keys
		_, got := f.New(context.Background(), "https://example.test", d, nil)
		if e == nil {
			if got != nil || !reflect.DeepEqual(events, []string{"claim", "key"}) {
				t.Fatal(events, got)
			}
		} else if !errors.Is(got, e) || !reflect.DeepEqual(events, []string{"claim"}) {
			t.Fatal(events, got)
		}
	}
}
func TestFactoryNewRejectsKeyThatDoesNotMatchStoredPublicKey(t *testing.T) {
	f, d := factoryHarness(t)
	d.PublicKey = "different-public"
	s, e := f.New(context.Background(), "https://example.test", d, nil)
	if s != nil || !errors.Is(e, bridgeactivation.ErrNotReady) || strings.Contains(e.Error(), d.PublicKey) {
		t.Fatal("key mismatch exposed")
	}
}
func TestFactoryNeverWidensCapabilities(t *testing.T) {
	for _, caps := range [][]string{{"notify_local"}, {"machine_state_query", "notify_local"}} {
		f, d := factoryHarness(t)
		d.CapabilityClasses = append([]string(nil), caps...)
		raw, e := f.New(context.Background(), "https://example.test", d, nil)
		if e != nil {
			t.Fatal(e)
		}
		s := raw.(*Session)
		d.CapabilityClasses[0] = "mutated"
		if !reflect.DeepEqual(s.client.Capabilities, caps) {
			t.Fatal("capabilities widened or aliased")
		}
	}
}
