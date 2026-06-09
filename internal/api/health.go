// internal/api/health.go
package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/t0mer/galactica/internal/version"
)

// HealthHandler handles the ops endpoints: /api/v1/health, /version, /readyz.
type HealthHandler struct {
	DB *sql.DB
}

// Health reports service and database status.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if err := h.DB.PingContext(r.Context()); err != nil {
		dbStatus = "error"
	}
	jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": version.Version,
		"db":      dbStatus,
	})
}

// Version returns build metadata.
func (h *HealthHandler) Version(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{
		"version": version.Version,
		"commit":  version.Commit,
		"date":    version.Date,
	})
}

// Ready returns 200 when the database is reachable, 503 otherwise.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.DB.PingContext(r.Context()); err != nil {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	jsonResponse(w, status, map[string]string{"error": msg})
}
