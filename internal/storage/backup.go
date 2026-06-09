// internal/storage/backup.go
package storage

import (
	"database/sql"
	"errors"
	"time"
)

// BackupTargetRow mirrors the backup_targets table.
type BackupTargetRow struct {
	ID              string
	Name            string
	Type            string
	ConfigEncrypted []byte
	RetentionDays   int
	Enabled         bool
	CreatedAt       time.Time
}

// BackupRow mirrors the backups table.
type BackupRow struct {
	ID         string
	TargetID   string
	InstanceID string
	TS         time.Time
	SizeBytes  int64
	Status     string
	Location   string
	Error      string
	CreatedAt  time.Time
}

// InsertBackupTarget inserts a new backup target record.
func InsertBackupTarget(db *sql.DB, r BackupTargetRow) error {
	_, err := db.Exec(
		`INSERT INTO backup_targets (id, name, type, config_encrypted, retention_days, enabled) VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.Type, r.ConfigEncrypted, r.RetentionDays, boolInt(r.Enabled),
	)
	return err
}

// ListBackupTargets returns all backup targets ordered by name.
func ListBackupTargets(db *sql.DB) ([]BackupTargetRow, error) {
	rows, err := db.Query(
		`SELECT id, name, type, config_encrypted, retention_days, enabled, created_at FROM backup_targets ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BackupTargetRow
	for rows.Next() {
		var r BackupTargetRow
		var en int
		if err := rows.Scan(&r.ID, &r.Name, &r.Type, &r.ConfigEncrypted, &r.RetentionDays, &en, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Enabled = en == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetBackupTarget returns the target with the given ID, or nil if not found.
func GetBackupTarget(db *sql.DB, id string) (*BackupTargetRow, error) {
	var r BackupTargetRow
	var en int
	err := db.QueryRow(
		`SELECT id, name, type, config_encrypted, retention_days, enabled, created_at FROM backup_targets WHERE id = ?`, id,
	).Scan(&r.ID, &r.Name, &r.Type, &r.ConfigEncrypted, &r.RetentionDays, &en, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Enabled = en == 1
	return &r, nil
}

// DeleteBackupTarget removes a backup target by ID.
func DeleteBackupTarget(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM backup_targets WHERE id = ?`, id)
	return err
}

// InsertBackup inserts a new backup run record.
func InsertBackup(db *sql.DB, r BackupRow) error {
	_, err := db.Exec(
		`INSERT INTO backups (id, target_id, instance_id, ts, size_bytes, status, location, error) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.TargetID, nullStr(r.InstanceID), r.TS, r.SizeBytes, r.Status, nullStr(r.Location), nullStr(r.Error),
	)
	return err
}

// UpdateBackupStatus updates status, location, error message, and size for a backup.
func UpdateBackupStatus(db *sql.DB, id, status, location, errMsg string, sizeBytes int64) error {
	_, err := db.Exec(
		`UPDATE backups SET status = ?, location = ?, error = ?, size_bytes = ? WHERE id = ?`,
		status, nullStr(location), nullStr(errMsg), sizeBytes, id,
	)
	return err
}

// ListBackups returns recent backups for a target ordered by timestamp descending.
func ListBackups(db *sql.DB, targetID string, limit int) ([]BackupRow, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Query(
		`SELECT id, target_id, COALESCE(instance_id,''), ts, size_bytes, status, COALESCE(location,''), COALESCE(error,''), created_at
         FROM backups WHERE target_id = ? ORDER BY ts DESC LIMIT ?`,
		targetID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BackupRow
	for rows.Next() {
		var r BackupRow
		if err := rows.Scan(&r.ID, &r.TargetID, &r.InstanceID, &r.TS, &r.SizeBytes, &r.Status, &r.Location, &r.Error, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
