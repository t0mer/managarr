// internal/api/health_test.go
package api_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/t0mer/galactica/internal/api"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "test.db")+"?_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestHealth(t *testing.T) {
	h := &api.HealthHandler{DB: openTestDB(t)}
	w := httptest.NewRecorder()
	h.Health(w, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
	assert.NotEmpty(t, body["version"])
	assert.Equal(t, "ok", body["db"])
}

func TestReady(t *testing.T) {
	h := &api.HealthHandler{DB: openTestDB(t)}
	w := httptest.NewRecorder()
	h.Ready(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVersion(t *testing.T) {
	h := &api.HealthHandler{DB: openTestDB(t)}
	w := httptest.NewRecorder()
	h.Version(w, httptest.NewRequest(http.MethodGet, "/version", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Contains(t, body, "version")
	assert.Contains(t, body, "commit")
	assert.Contains(t, body, "date")
}
