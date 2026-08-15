package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// OpenDB opens a SQLite handle with the settings every long-lived connection
// to thoth.db wants: busy_timeout so concurrent writers across the app's
// separate handles block briefly instead of failing with SQLITE_BUSY, and a
// single pooled connection so that wait is deterministic and file
// descriptors stay bounded.
//
// The pragma forces an early connection, which creates the file on a missing
// path — callers that must stay lazy (the settings repo's read-only doctor
// contract) open their own handle instead.
func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}
