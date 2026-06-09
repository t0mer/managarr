// internal/api/deps.go
package api

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/t0mer/galactica/internal/providers"
	"github.com/t0mer/galactica/internal/storage"
)

// errNotFound signals a 404-worthy response.
var errNotFound = errors.New("not found")

// Deps holds shared dependencies for all API handlers.
type Deps struct {
	DB        *sql.DB
	SecretKey string
	Log       *slog.Logger
}

// resolve loads an instance from DB and decrypts its API key from secrets.
// Returns errNotFound if the instance does not exist.
func (d *Deps) resolve(ctx context.Context, id string) (providers.Instance, error) {
	row, err := storage.GetInstance(d.DB, id)
	if err != nil {
		return providers.Instance{}, err
	}
	if row == nil {
		return providers.Instance{}, errNotFound
	}
	inst := providers.Instance{
		ID:      row.ID,
		Kind:    providers.Kind(row.Kind),
		Name:    row.Name,
		BaseURL: row.BaseURL,
	}
	if d.SecretKey != "" {
		enc, serr := storage.GetSecret(d.DB, id, "api_key")
		if serr == nil && len(enc) > 0 {
			plain, derr := storage.Decrypt(enc, d.SecretKey)
			if derr == nil {
				inst.APIKey = string(plain)
			}
		}
	}
	return inst, nil
}
