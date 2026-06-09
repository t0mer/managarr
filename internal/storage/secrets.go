// internal/storage/secrets.go
package storage

import (
	"database/sql"
	"errors"
)

// PutSecret upserts an encrypted secret value for (instanceID, key).
func PutSecret(db *sql.DB, instanceID, key string, ciphertext []byte) error {
	_, err := db.Exec(
		`INSERT INTO secrets (instance_id, key, value) VALUES (?, ?, ?)
         ON CONFLICT(instance_id, key) DO UPDATE SET value = excluded.value`,
		instanceID, key, ciphertext,
	)
	return err
}

// GetSecret returns the encrypted blob for (instanceID, key), or nil if missing.
func GetSecret(db *sql.DB, instanceID, key string) ([]byte, error) {
	var val []byte
	err := db.QueryRow(
		`SELECT value FROM secrets WHERE instance_id = ? AND key = ?`,
		instanceID, key,
	).Scan(&val)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return val, err
}

// DeleteSecrets removes all secrets for an instance.
func DeleteSecrets(db *sql.DB, instanceID string) error {
	_, err := db.Exec(`DELETE FROM secrets WHERE instance_id = ?`, instanceID)
	return err
}
