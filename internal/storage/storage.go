// internal/storage/storage.go
package storage

import (
	"database/sql"
	"fmt"
)

// Store is the database abstraction. Phase 1: SQLite only.
// Phase 2+: Open() can return a Postgres implementation without changing callers.
type Store interface {
	DB() *sql.DB
	Close() error
}

// Open returns a Store for the given driver and DSN.
// Supported drivers: "sqlite" (default).
func Open(driver, dsn string) (Store, error) {
	switch driver {
	case "sqlite", "":
		return openSQLite(dsn)
	default:
		return nil, fmt.Errorf("unsupported storage driver %q", driver)
	}
}
