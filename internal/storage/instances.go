// internal/storage/instances.go
package storage

import (
	"database/sql"
	"errors"
	"time"
)

// InstanceRow mirrors the instances table.
type InstanceRow struct {
	ID        string
	Kind      string
	Name      string
	BaseURL   string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// InsertInstance inserts a new instance record.
func InsertInstance(db *sql.DB, id, kind, name, baseURL string) error {
	_, err := db.Exec(
		`INSERT INTO instances (id, kind, name, base_url) VALUES (?, ?, ?, ?)`,
		id, kind, name, baseURL,
	)
	return err
}

// GetInstance returns the instance with the given ID, or nil if not found.
func GetInstance(db *sql.DB, id string) (*InstanceRow, error) {
	row := db.QueryRow(
		`SELECT id, kind, name, base_url, enabled, created_at, updated_at FROM instances WHERE id = ?`,
		id,
	)
	var r InstanceRow
	var enabled int
	err := row.Scan(&r.ID, &r.Kind, &r.Name, &r.BaseURL, &enabled, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Enabled = enabled == 1
	return &r, nil
}

// ListInstances returns all instances ordered by name.
func ListInstances(db *sql.DB) ([]InstanceRow, error) {
	rows, err := db.Query(
		`SELECT id, kind, name, base_url, enabled, created_at, updated_at FROM instances ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InstanceRow
	for rows.Next() {
		var r InstanceRow
		var enabled int
		if err := rows.Scan(&r.ID, &r.Kind, &r.Name, &r.BaseURL, &enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.Enabled = enabled == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateInstance updates name, base_url, and enabled for the given ID.
func UpdateInstance(db *sql.DB, id, name, baseURL string, enabled bool) error {
	_, err := db.Exec(
		`UPDATE instances SET name = ?, base_url = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		name, baseURL, boolInt(enabled), id,
	)
	return err
}

// DeleteInstance removes an instance and its cascaded children.
func DeleteInstance(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM instances WHERE id = ?`, id)
	return err
}

// SetInstanceEnabled sets the enabled flag on an instance.
func SetInstanceEnabled(db *sql.DB, id string, enabled bool) error {
	_, err := db.Exec(
		`UPDATE instances SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		boolInt(enabled), id,
	)
	return err
}
