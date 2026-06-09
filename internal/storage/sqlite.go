// internal/storage/sqlite.go
package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type sqliteStore struct {
	db *sql.DB
}

func openSQLite(dsn string) (*sqliteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}
	// SQLite supports one writer at a time.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging sqlite: %w", err)
	}
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return &sqliteStore{db: db}, nil
}

func (s *sqliteStore) DB() *sql.DB  { return s.db }
func (s *sqliteStore) Close() error { return s.db.Close() }
