package activationrepo

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/bridgepersist"
)

func TestHeldActivationReadNeverBlocksClearPaired(t *testing.T) {
	dir := t.TempDir()
	migrated, err := sqlite.Open(dir)
	must(t, err)
	must(t, migrated.Close())
	reader, err := OpenDedicated(dir)
	must(t, err)
	defer reader.Close()
	writer, err := OpenDedicated(dir)
	must(t, err)
	defer writer.Close()
	for _, db := range []*sql.DB{reader, writer} {
		var mode string
		var timeout int
		must(t, db.QueryRow("PRAGMA journal_mode").Scan(&mode))
		must(t, db.QueryRow("PRAGMA busy_timeout").Scan(&timeout))
		if mode != "wal" || timeout != 5000 {
			t.Fatal("inconsistent reader/device-store pragmas")
		}
	}
	repo := New(reader)
	d := seed(t, repo, "held-read")
	generation, err := repo.ClaimActivation(bg, d.OwnerID, d.DeviceID)
	must(t, err)
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- repo.readOnly(bg, func(c *sql.Conn) error {
			if _, err := read(bg, c, d.OwnerID); err != nil {
				return err
			}
			close(held)
			<-release
			snapshot, err := device(bg, c, string(d.DeviceID))
			if err == nil && snapshot.State != domain.DeviceBridgeStatePaired {
				t.Error("read snapshot changed")
			}
			return err
		})
	}()
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	select {
	case <-held:
	case err := <-done:
		t.Fatalf("reader failed: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("reader did not start")
	}
	ctx, cancel := context.WithTimeout(bg, 2*time.Second)
	defer cancel()
	must(t, bridgepersist.New(writer).ClearPaired(ctx, devicebridge.Scope{OwnerID: d.OwnerID, DeviceID: string(d.DeviceID)}))
	close(release)
	released = true
	must(t, <-done)
	want(t, repo.VerifyClaim(bg, d.OwnerID, d.DeviceID, generation), ErrStaleClaim)
	if load(t, repo, d.OwnerID).Device.State != domain.DeviceBridgeStateRevoked {
		t.Fatal("clear was not durable")
	}
}
