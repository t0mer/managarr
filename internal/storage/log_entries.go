// internal/storage/log_entries.go
package storage

import (
	"database/sql"
	"time"
)

// LogEntryRow mirrors the log_entries table.
type LogEntryRow struct {
	ID         int64
	InstanceID string
	TS         time.Time
	Level      string
	SourceType string
	Message    string
	Raw        string
}

// InsertLogEntries bulk-inserts log entries within a single transaction.
func InsertLogEntries(db *sql.DB, entries []LogEntryRow) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	stmt, err := tx.Prepare(
		`INSERT OR IGNORE INTO log_entries (instance_id, ts, level, source_type, message, raw)
         VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, e := range entries {
		if _, err := stmt.Exec(e.InstanceID, e.TS, e.Level, e.SourceType, e.Message, e.Raw); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// QueryLogs returns log entries filtered by instanceID, level, and time window.
// level="" means all levels. Pass limit=0 for default 200.
func QueryLogs(db *sql.DB, instanceID, level string, since time.Time, limit int) ([]LogEntryRow, error) {
	if limit <= 0 {
		limit = 200
	}
	var (
		rows *sql.Rows
		err  error
	)
	if level != "" && instanceID != "" {
		rows, err = db.Query(
			`SELECT id, instance_id, ts, level, source_type, message, raw
             FROM log_entries WHERE instance_id = ? AND level = ? AND ts >= ?
             ORDER BY ts DESC LIMIT ?`,
			instanceID, level, since, limit,
		)
	} else if instanceID != "" {
		rows, err = db.Query(
			`SELECT id, instance_id, ts, level, source_type, message, raw
             FROM log_entries WHERE instance_id = ? AND ts >= ?
             ORDER BY ts DESC LIMIT ?`,
			instanceID, since, limit,
		)
	} else if level != "" {
		rows, err = db.Query(
			`SELECT id, instance_id, ts, level, source_type, message, raw
             FROM log_entries WHERE level = ? AND ts >= ?
             ORDER BY ts DESC LIMIT ?`,
			level, since, limit,
		)
	} else {
		rows, err = db.Query(
			`SELECT id, instance_id, ts, level, source_type, message, raw
             FROM log_entries WHERE ts >= ?
             ORDER BY ts DESC LIMIT ?`,
			since, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LogEntryRow
	for rows.Next() {
		var r LogEntryRow
		if err := rows.Scan(&r.ID, &r.InstanceID, &r.TS, &r.Level, &r.SourceType, &r.Message, &r.Raw); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// QueryLogsSinceID returns log entries with id > afterID ordered by id ASC (for SSE polling).
func QueryLogsSinceID(db *sql.DB, afterID int64, instanceID, level string) ([]LogEntryRow, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if instanceID != "" && level != "" {
		rows, err = db.Query(
			`SELECT id, instance_id, ts, level, source_type, message, raw
             FROM log_entries WHERE id > ? AND instance_id = ? AND level = ?
             ORDER BY id ASC LIMIT 100`,
			afterID, instanceID, level,
		)
	} else if instanceID != "" {
		rows, err = db.Query(
			`SELECT id, instance_id, ts, level, source_type, message, raw
             FROM log_entries WHERE id > ? AND instance_id = ?
             ORDER BY id ASC LIMIT 100`,
			afterID, instanceID,
		)
	} else if level != "" {
		rows, err = db.Query(
			`SELECT id, instance_id, ts, level, source_type, message, raw
             FROM log_entries WHERE id > ? AND level = ?
             ORDER BY id ASC LIMIT 100`,
			afterID, level,
		)
	} else {
		rows, err = db.Query(
			`SELECT id, instance_id, ts, level, source_type, message, raw
             FROM log_entries WHERE id > ?
             ORDER BY id ASC LIMIT 100`,
			afterID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LogEntryRow
	for rows.Next() {
		var r LogEntryRow
		if err := rows.Scan(&r.ID, &r.InstanceID, &r.TS, &r.Level, &r.SourceType, &r.Message, &r.Raw); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
