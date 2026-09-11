package sqlite

import (
	"database/sql"
	_ "embed"
	"fmt"
	"regexp"
)

//go:embed schema/outcome_trash.sql
var outcomeTrashDDL string

// These startup seams follow the existing recovery support for databases whose
// upstream migration numbers collided. Install a guard only when its target
// table exists; deletion itself still requires the complete Outcome schema.
func installOutcomeDeletionSchema(db *sql.DB) error {
	statement := regexp.MustCompile(`(?s)DROP TRIGGER IF EXISTS \w+;\s*CREATE TRIGGER .*?END;`)
	target := regexp.MustCompile(`(?i)\bON\s+(\w+)`)
	for _, ddl := range []string{outcomeDeletionGuardsDDL, outcomeTrashDDL} {
		for _, stmt := range statement.FindAllString(ddl, -1) {
			match := target.FindStringSubmatch(stmt)
			if len(match) != 2 {
				return fmt.Errorf("invalid Outcome deletion trigger")
			}
			var exists int
			if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", match[1]).Scan(&exists); err != nil {
				return err
			}
			if exists == 0 {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				return err
			}
		}
	}
	return nil
}
