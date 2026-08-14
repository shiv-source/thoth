package store

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// migrate applies the embedded SQL migrations in migrations/ in filename
// order: migration i is version i+1. Gated on PRAGMA user_version: idempotent
// across opens, and a fresh database moves 0→N with zero rows touched. Each
// migration runs in one transaction together with its version bump, so a
// failed migration rolls back atomically. The index connection on the same
// file (index.Open) never reads these tables, so no cross-connection
// interference.
func migrate(db *sql.DB) error {
	names, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(names)

	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}
	for i, name := range names {
		version := i + 1
		if v >= version {
			continue
		}
		raw, err := fs.ReadFile(migrationFS, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if err := applyMigration(db, version, splitStatements(string(raw))); err != nil {
			return err
		}
		v = version
	}
	return nil
}

// splitStatements splits a migration file into statements on ";".
// Statements must not contain semicolons inside string literals or comments.
func splitStatements(raw string) []string {
	var out []string
	for s := range strings.SplitSeq(raw, ";") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func applyMigration(db *sql.DB, version int, stmts []string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration %d: %w", version, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after Commit
	for _, s := range stmts {
		if _, err := tx.Exec(s); err != nil {
			return fmt.Errorf("migration %d: %w", version, err)
		}
	}
	// PRAGMA does not accept bound parameters; version comes from code.
	if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, version)); err != nil {
		return fmt.Errorf("set user_version %d: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %d: %w", version, err)
	}
	return nil
}
