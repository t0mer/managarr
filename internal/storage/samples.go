// internal/storage/samples.go
package storage

import (
	"database/sql"
	"time"
)

// SampleRow mirrors the samples table.
type SampleRow struct {
	ID         int64
	InstanceID string
	Metric     string
	TS         time.Time
	Value      float64
}

// InsertSamples bulk-inserts metric samples within a single transaction.
func InsertSamples(db *sql.DB, samples []SampleRow) error {
	if len(samples) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	stmt, err := tx.Prepare(
		`INSERT INTO samples (instance_id, metric, ts, value) VALUES (?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, s := range samples {
		if _, err := stmt.Exec(s.InstanceID, s.Metric, s.TS, s.Value); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// QuerySeries returns time-series samples for (instanceID, metric) since a given time.
func QuerySeries(db *sql.DB, instanceID, metric string, since time.Time) ([]SampleRow, error) {
	rows, err := db.Query(
		`SELECT id, instance_id, metric, ts, value
         FROM samples WHERE instance_id = ? AND metric = ? AND ts >= ?
         ORDER BY ts ASC`,
		instanceID, metric, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SampleRow
	for rows.Next() {
		var r SampleRow
		if err := rows.Scan(&r.ID, &r.InstanceID, &r.Metric, &r.TS, &r.Value); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListMetricNames returns distinct metric names for an instance.
func ListMetricNames(db *sql.DB, instanceID string) ([]string, error) {
	rows, err := db.Query(
		`SELECT DISTINCT metric FROM samples WHERE instance_id = ? ORDER BY metric`,
		instanceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
