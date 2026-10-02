package activationrepo

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"path/filepath"
)

// OpenDedicated opens an already migrated database; it never runs migrations.
func OpenDedicated(dataDir string) (*sql.DB, error) {
	db, e := sql.Open("sqlite", "file:"+filepath.Join(dataDir, "kennel.db")+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)")
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if e = db.Ping(); e != nil {
		db.Close()
		return nil, e
	}
	return db, nil
}
