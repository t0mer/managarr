// internal/storage/jackett.go
package storage

import "database/sql"

// ListIndexerMonitored returns a map of indexerID → monitored for all stored
// overrides for a given Jackett instance. Indexers not in the map default to monitored=true.
func ListIndexerMonitored(db *sql.DB, instanceID string) (map[string]bool, error) {
	rows, err := db.Query(
		`SELECT indexer_id, monitored FROM jackett_indexer_settings WHERE instance_id = ?`,
		instanceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		var mon int
		if err := rows.Scan(&id, &mon); err != nil {
			return nil, err
		}
		out[id] = mon != 0
	}
	return out, rows.Err()
}

// SetIndexerMonitored upserts the monitored flag for one indexer.
func SetIndexerMonitored(db *sql.DB, instanceID, indexerID string, monitored bool) error {
	m := 0
	if monitored {
		m = 1
	}
	_, err := db.Exec(
		`INSERT INTO jackett_indexer_settings (instance_id, indexer_id, monitored)
		 VALUES (?, ?, ?)
		 ON CONFLICT(instance_id, indexer_id) DO UPDATE SET monitored = excluded.monitored`,
		instanceID, indexerID, m,
	)
	return err
}
