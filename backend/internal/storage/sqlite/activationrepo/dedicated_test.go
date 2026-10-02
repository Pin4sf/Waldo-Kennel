package activationrepo

import (
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"testing"
)

func TestOpenDedicatedUsesMigratedDatabaseAndPragmas(t *testing.T) {
	dir := t.TempDir()
	s, e := sqlite.Open(dir)
	must(t, e)
	must(t, s.Close())
	db, e := OpenDedicated(dir)
	must(t, e)
	defer db.Close()
	var foreignKeys, timeout int
	must(t, db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys))
	must(t, db.QueryRow("PRAGMA busy_timeout").Scan(&timeout))
	if foreignKeys != 1 || timeout != 5000 || db.Stats().MaxOpenConnections != 1 {
		t.Fatal("dedicated connection configuration")
	}
	var n int
	must(t, db.QueryRow("SELECT count(*) FROM device_bridge_activations").Scan(&n))
}
