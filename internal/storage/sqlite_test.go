// internal/storage/sqlite_test.go
package storage_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/galactica/internal/storage"
)

func TestOpenSQLiteCreatesSchema(t *testing.T) {
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "test.db") + "?_fk=1"

	store, err := storage.Open("sqlite", dsn)
	require.NoError(t, err)
	defer store.Close()

	tables := []string{
		"instances", "secrets", "log_entries", "issues", "samples",
		"notify_channels", "backup_targets", "backups", "sync_jobs",
		"schedules", "settings", "api_tokens", "admin", "_migrations",
	}
	for _, table := range tables {
		var name string
		err := store.DB().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		assert.NoError(t, err, "table %q should exist", table)
		assert.Equal(t, table, name)
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "test.db") + "?_fk=1"

	s1, err := storage.Open("sqlite", dsn)
	require.NoError(t, err)
	s1.Close()

	s2, err := storage.Open("sqlite", dsn)
	require.NoError(t, err)
	s2.Close()
}

func TestUnsupportedDriverErrors(t *testing.T) {
	_, err := storage.Open("mysql", "dsn")
	assert.Error(t, err)
}
