package governedtools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

func TestUncertaintyStorePersistsUnknownTerminationAcrossInstances(t *testing.T) {
	dataDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writer, err := NewUncertaintyStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	fact := ports.GovernedCheckUncertainty{
		SessionID: "session-1", CheckID: "check-1", TerminationUnknown: true,
		TimedOut: true, EnforcedBy: "process-group", ObservedAt: time.Now().UTC(),
	}
	if err := writer.RecordGovernedCheckUncertainty(context.Background(), fact); err != nil {
		t.Fatal(err)
	}
	reader, err := NewUncertaintyStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	got, found, err := reader.GovernedCheckUncertainty(context.Background(), domain.SessionID("session-1"))
	if err != nil || !found {
		t.Fatalf("read = (%+v, %v, %v), want persisted marker", got, found, err)
	}
	if got.CheckID != fact.CheckID || !got.TerminationUnknown || !got.TimedOut {
		t.Fatalf("fact = %+v, want %+v", got, fact)
	}
	entries, err := os.ReadDir(filepath.Join(dataDir, uncertaintyDirectory))
	if err != nil || len(entries) != 1 {
		t.Fatalf("marker entries = %d, %v", len(entries), err)
	}
}

func TestUncertaintyStoreRejectsNonUnknownFact(t *testing.T) {
	store, err := NewUncertaintyStore(filepath.Clean(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	err = store.RecordGovernedCheckUncertainty(context.Background(), ports.GovernedCheckUncertainty{
		SessionID: "session-1", CheckID: "check-1", ObservedAt: time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("non-unknown check result was persisted as uncertainty")
	}
}

func TestUncertaintyStoreClearsMarkerOnlyOnConfirmedTermination(t *testing.T) {
	dataDir := filepath.Clean(t.TempDir())
	store, err := NewUncertaintyStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	fact := ports.GovernedCheckUncertainty{
		SessionID: "session-1", CheckID: "check-1", TerminationUnknown: true,
		EnforcedBy: "pending", ObservedAt: time.Now().UTC(),
	}
	if err := store.RecordGovernedCheckUncertainty(context.Background(), fact); err != nil {
		t.Fatal(err)
	}
	if err := store.ClearGovernedCheckUncertainty(context.Background(), fact.SessionID); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.GovernedCheckUncertainty(context.Background(), fact.SessionID); err != nil || found {
		t.Fatalf("cleared marker = found %v, err %v", found, err)
	}
}
