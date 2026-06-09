// internal/storage/sync.go
package storage

import (
	"database/sql"
	"errors"
	"time"
)

// SyncJobRow mirrors the sync_jobs table.
type SyncJobRow struct {
	ID               string
	SourceInstanceID string
	TargetInstanceID string
	Selectors        string
	Schedule         string
	Enabled          bool
	CreatedAt        time.Time
}

// InsertSyncJob inserts a new sync job record.
func InsertSyncJob(db *sql.DB, r SyncJobRow) error {
	_, err := db.Exec(
		`INSERT INTO sync_jobs (id, source_instance_id, target_instance_id, selectors, schedule, enabled) VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.SourceInstanceID, r.TargetInstanceID, r.Selectors, nullStr(r.Schedule), boolInt(r.Enabled),
	)
	return err
}

// ListSyncJobs returns all sync jobs ordered by creation time descending.
func ListSyncJobs(db *sql.DB) ([]SyncJobRow, error) {
	rows, err := db.Query(
		`SELECT id, source_instance_id, target_instance_id, selectors, COALESCE(schedule,''), enabled, created_at
         FROM sync_jobs ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SyncJobRow
	for rows.Next() {
		var r SyncJobRow
		var en int
		if err := rows.Scan(&r.ID, &r.SourceInstanceID, &r.TargetInstanceID, &r.Selectors, &r.Schedule, &en, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Enabled = en == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetSyncJob returns the sync job with the given ID, or nil if not found.
func GetSyncJob(db *sql.DB, id string) (*SyncJobRow, error) {
	var r SyncJobRow
	var en int
	err := db.QueryRow(
		`SELECT id, source_instance_id, target_instance_id, selectors, COALESCE(schedule,''), enabled, created_at
         FROM sync_jobs WHERE id = ?`, id,
	).Scan(&r.ID, &r.SourceInstanceID, &r.TargetInstanceID, &r.Selectors, &r.Schedule, &en, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Enabled = en == 1
	return &r, nil
}

// DeleteSyncJob removes a sync job by ID.
func DeleteSyncJob(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM sync_jobs WHERE id = ?`, id)
	return err
}
