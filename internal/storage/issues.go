// internal/storage/issues.go
package storage

import (
	"database/sql"
	"errors"
	"time"
)

// IssueRow mirrors the issues table.
type IssueRow struct {
	ID          string
	InstanceID  string
	Fingerprint string
	Title       string
	Severity    string
	ImpactScore float64
	Status      string
	FirstSeen   time.Time
	LastSeen    time.Time
	Count       int
}

// UpsertIssue inserts a new issue or increments count + updates last_seen on collision.
func UpsertIssue(db *sql.DB, r IssueRow) error {
	_, err := db.Exec(
		`INSERT INTO issues (id, instance_id, fingerprint, title, severity, impact_score, status, first_seen, last_seen, count)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
         ON CONFLICT(fingerprint) DO UPDATE SET
           last_seen    = excluded.last_seen,
           count        = count + 1,
           impact_score = excluded.impact_score,
           title        = excluded.title`,
		r.ID, r.InstanceID, r.Fingerprint, r.Title, r.Severity, r.ImpactScore,
		r.Status, r.FirstSeen, r.LastSeen, r.Count,
	)
	return err
}

// ListIssues returns issues filtered by instanceID and status.
// status="" returns all. instanceID="" returns all instances.
func ListIssues(db *sql.DB, instanceID, status string) ([]IssueRow, error) {
	var rows *sql.Rows
	var err error
	if instanceID != "" && status != "" {
		rows, err = db.Query(
			`SELECT id, instance_id, fingerprint, title, severity, impact_score, status, first_seen, last_seen, count
             FROM issues WHERE instance_id = ? AND status = ?
             ORDER BY impact_score DESC`,
			instanceID, status,
		)
	} else if instanceID != "" {
		rows, err = db.Query(
			`SELECT id, instance_id, fingerprint, title, severity, impact_score, status, first_seen, last_seen, count
             FROM issues WHERE instance_id = ?
             ORDER BY impact_score DESC`,
			instanceID,
		)
	} else if status != "" {
		rows, err = db.Query(
			`SELECT id, instance_id, fingerprint, title, severity, impact_score, status, first_seen, last_seen, count
             FROM issues WHERE status = ?
             ORDER BY impact_score DESC`,
			status,
		)
	} else {
		rows, err = db.Query(
			`SELECT id, instance_id, fingerprint, title, severity, impact_score, status, first_seen, last_seen, count
             FROM issues ORDER BY impact_score DESC`,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IssueRow
	for rows.Next() {
		var r IssueRow
		if err := rows.Scan(&r.ID, &r.InstanceID, &r.Fingerprint, &r.Title, &r.Severity,
			&r.ImpactScore, &r.Status, &r.FirstSeen, &r.LastSeen, &r.Count); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetIssue returns the issue with the given ID, or nil if not found.
func GetIssue(db *sql.DB, id string) (*IssueRow, error) {
	var r IssueRow
	err := db.QueryRow(
		`SELECT id, instance_id, fingerprint, title, severity, impact_score, status, first_seen, last_seen, count
         FROM issues WHERE id = ?`, id,
	).Scan(&r.ID, &r.InstanceID, &r.Fingerprint, &r.Title, &r.Severity,
		&r.ImpactScore, &r.Status, &r.FirstSeen, &r.LastSeen, &r.Count)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &r, err
}

// UpdateIssueStatus sets the status for an issue.
func UpdateIssueStatus(db *sql.DB, id, status string) error {
	_, err := db.Exec(`UPDATE issues SET status = ? WHERE id = ?`, status, id)
	return err
}
