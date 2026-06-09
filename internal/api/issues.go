// internal/api/issues.go
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/t0mer/galactica/internal/storage"
)

// IssuesHandler handles /api/v1/issues routes.
type IssuesHandler struct{ *Deps }

type issueResp struct {
	ID          string  `json:"id"`
	InstanceID  string  `json:"instance_id"`
	Fingerprint string  `json:"fingerprint"`
	Title       string  `json:"title"`
	Severity    string  `json:"severity"`
	ImpactScore float64 `json:"impact_score"`
	Status      string  `json:"status"`
	Count       int     `json:"count"`
	FirstSeen   string  `json:"first_seen"`
	LastSeen    string  `json:"last_seen"`
}

func toIssueResp(r storage.IssueRow) issueResp {
	return issueResp{
		ID:          r.ID,
		InstanceID:  r.InstanceID,
		Fingerprint: r.Fingerprint,
		Title:       r.Title,
		Severity:    r.Severity,
		ImpactScore: r.ImpactScore,
		Status:      r.Status,
		Count:       r.Count,
		FirstSeen:   r.FirstSeen.Format(time.RFC3339),
		LastSeen:    r.LastSeen.Format(time.RFC3339),
	}
}

// List handles GET /api/v1/issues?instance_id=<id>&status=<status>
// status defaults to "open"; use "all" for all statuses.
func (h *IssuesHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	instanceID := q.Get("instance_id")
	status := q.Get("status")
	if status == "all" {
		status = ""
	} else if status == "" {
		status = "open"
	}

	rows, err := storage.ListIssues(h.DB, instanceID, status)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]issueResp, len(rows))
	for i, r := range rows {
		out[i] = toIssueResp(r)
	}
	jsonResponse(w, http.StatusOK, out)
}

// Get handles GET /api/v1/issues/{id}
func (h *IssuesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	row, err := storage.GetIssue(h.DB, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if row == nil {
		jsonError(w, http.StatusNotFound, "issue not found")
		return
	}
	jsonResponse(w, http.StatusOK, toIssueResp(*row))
}

// UpdateStatus handles PATCH /api/v1/issues/{id}/status
func (h *IssuesHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	allowed := map[string]bool{"open": true, "acknowledged": true, "resolved": true}
	if !allowed[body.Status] {
		jsonError(w, http.StatusBadRequest, "status must be open, acknowledged, or resolved")
		return
	}
	row, err := storage.GetIssue(h.DB, id)
	if err != nil || row == nil {
		jsonError(w, http.StatusNotFound, "issue not found")
		return
	}
	if err := storage.UpdateIssueStatus(h.DB, id, body.Status); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
