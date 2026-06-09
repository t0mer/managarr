// internal/storage/notify.go
package storage

import (
	"database/sql"
	"errors"
	"time"
)

// NotifyChannelRow mirrors the notify_channels table.
type NotifyChannelRow struct {
	ID              string
	Name            string
	Provider        string
	ConfigEncrypted []byte
	Enabled         bool
	NotifyOnSuccess bool
	NotifyOnFailure bool
	CreatedAt       time.Time
}

// InsertNotifyChannel inserts a new notification channel record.
func InsertNotifyChannel(db *sql.DB, r NotifyChannelRow) error {
	_, err := db.Exec(
		`INSERT INTO notify_channels (id, name, provider, config_encrypted, enabled, notify_on_success, notify_on_failure)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.Provider, r.ConfigEncrypted,
		boolInt(r.Enabled), boolInt(r.NotifyOnSuccess), boolInt(r.NotifyOnFailure),
	)
	return err
}

// ListNotifyChannels returns all notification channels ordered by name.
func ListNotifyChannels(db *sql.DB) ([]NotifyChannelRow, error) {
	rows, err := db.Query(
		`SELECT id, name, provider, config_encrypted, enabled, notify_on_success, notify_on_failure, created_at
         FROM notify_channels ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotifyChannels(rows)
}

// GetNotifyChannel returns the channel with the given ID, or nil if not found.
func GetNotifyChannel(db *sql.DB, id string) (*NotifyChannelRow, error) {
	row := db.QueryRow(
		`SELECT id, name, provider, config_encrypted, enabled, notify_on_success, notify_on_failure, created_at
         FROM notify_channels WHERE id = ?`, id,
	)
	var r NotifyChannelRow
	var en, ns, nf int
	err := row.Scan(&r.ID, &r.Name, &r.Provider, &r.ConfigEncrypted, &en, &ns, &nf, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Enabled, r.NotifyOnSuccess, r.NotifyOnFailure = en == 1, ns == 1, nf == 1
	return &r, nil
}

// UpdateNotifyChannel updates the mutable fields of a notification channel.
func UpdateNotifyChannel(db *sql.DB, r NotifyChannelRow) error {
	_, err := db.Exec(
		`UPDATE notify_channels SET name = ?, config_encrypted = ?, enabled = ?, notify_on_success = ?, notify_on_failure = ?
         WHERE id = ?`,
		r.Name, r.ConfigEncrypted, boolInt(r.Enabled), boolInt(r.NotifyOnSuccess), boolInt(r.NotifyOnFailure), r.ID,
	)
	return err
}

// DeleteNotifyChannel removes a notification channel by ID.
func DeleteNotifyChannel(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM notify_channels WHERE id = ?`, id)
	return err
}

func scanNotifyChannels(rows *sql.Rows) ([]NotifyChannelRow, error) {
	var out []NotifyChannelRow
	for rows.Next() {
		var r NotifyChannelRow
		var en, ns, nf int
		if err := rows.Scan(&r.ID, &r.Name, &r.Provider, &r.ConfigEncrypted, &en, &ns, &nf, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Enabled, r.NotifyOnSuccess, r.NotifyOnFailure = en == 1, ns == 1, nf == 1
		out = append(out, r)
	}
	return out, rows.Err()
}
