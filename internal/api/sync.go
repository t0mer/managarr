// internal/api/sync.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/t0mer/galactica/internal/providers"
	"github.com/t0mer/galactica/internal/storage"
)

// SyncHandler handles /api/v1/sync routes.
type SyncHandler struct{ *Deps }

type syncJobResp struct {
	ID               string   `json:"id"`
	SourceInstanceID string   `json:"source_instance_id"`
	TargetInstanceID string   `json:"target_instance_id"`
	Selectors        []string `json:"selectors"`
	Schedule         string   `json:"schedule,omitempty"`
	Enabled          bool     `json:"enabled"`
	CreatedAt        string   `json:"created_at"`
}

type syncChangeResp struct {
	Field    string `json:"field"`
	OldValue any    `json:"old_value"`
	NewValue any    `json:"new_value"`
}

type syncPreviewResp struct {
	Changes []syncChangeResp `json:"changes"`
	Count   int              `json:"count"`
}

func toSyncJobResp(r storage.SyncJobRow) syncJobResp {
	var sel []string
	if r.Selectors != "" && r.Selectors != "[]" {
		_ = json.Unmarshal([]byte(r.Selectors), &sel)
	}
	if sel == nil {
		sel = []string{}
	}
	return syncJobResp{
		ID:               r.ID,
		SourceInstanceID: r.SourceInstanceID,
		TargetInstanceID: r.TargetInstanceID,
		Selectors:        sel,
		Schedule:         r.Schedule,
		Enabled:          r.Enabled,
		CreatedAt:        r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// List handles GET /api/v1/sync/jobs.
func (h *SyncHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := storage.ListSyncJobs(h.DB)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]syncJobResp, len(rows))
	for i, row := range rows {
		out[i] = toSyncJobResp(row)
	}
	jsonResponse(w, http.StatusOK, out)
}

type createSyncJobReq struct {
	SourceInstanceID string   `json:"source_instance_id"`
	TargetInstanceID string   `json:"target_instance_id"`
	Selectors        []string `json:"selectors"`
	Schedule         string   `json:"schedule"`
	Enabled          bool     `json:"enabled"`
}

// Create handles POST /api/v1/sync/jobs.
func (h *SyncHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createSyncJobReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.SourceInstanceID == "" || req.TargetInstanceID == "" {
		jsonError(w, http.StatusBadRequest, "source_instance_id and target_instance_id are required")
		return
	}

	// Validate both instances exist, are same kind, and provider is Syncable.
	src, err := h.resolve(r.Context(), req.SourceInstanceID)
	if err != nil {
		jsonError(w, http.StatusNotFound, "source instance not found")
		return
	}
	tgt, err := h.resolve(r.Context(), req.TargetInstanceID)
	if err != nil {
		jsonError(w, http.StatusNotFound, "target instance not found")
		return
	}
	if src.Kind != tgt.Kind {
		jsonError(w, http.StatusBadRequest, "source and target must be the same kind")
		return
	}
	p, err := providers.Get(src.Kind)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "provider not registered")
		return
	}
	if _, ok := p.(providers.Syncable); !ok {
		jsonError(w, http.StatusBadRequest, "provider does not support sync")
		return
	}

	selectors := req.Selectors
	if selectors == nil {
		selectors = []string{}
	}
	selJSON, _ := json.Marshal(selectors)

	id := uuid.New().String()
	row := storage.SyncJobRow{
		ID:               id,
		SourceInstanceID: req.SourceInstanceID,
		TargetInstanceID: req.TargetInstanceID,
		Selectors:        string(selJSON),
		Schedule:         req.Schedule,
		Enabled:          req.Enabled,
	}
	if err := storage.InsertSyncJob(h.DB, row); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]string{"id": id})
}

// Delete handles DELETE /api/v1/sync/jobs/{id}.
func (h *SyncHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := storage.GetSyncJob(h.DB, id)
	if err != nil || existing == nil {
		jsonError(w, http.StatusNotFound, "sync job not found")
		return
	}
	if err := storage.DeleteSyncJob(h.DB, id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Preview handles POST /api/v1/sync/jobs/{id}/preview.
func (h *SyncHandler) Preview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := storage.GetSyncJob(h.DB, id)
	if err != nil || job == nil {
		jsonError(w, http.StatusNotFound, "sync job not found")
		return
	}
	src, tgt, syncable, err := h.resolveSyncPair(r, job)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	srcSnap, err := syncable.Snapshot(r.Context(), src)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "snapshot source: "+err.Error())
		return
	}
	tgtSnap, err := syncable.Snapshot(r.Context(), tgt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "snapshot target: "+err.Error())
		return
	}
	changes := syncable.Diff(srcSnap, tgtSnap)
	out := make([]syncChangeResp, len(changes))
	for i, c := range changes {
		out[i] = syncChangeResp{Field: c.Field, OldValue: c.OldValue, NewValue: c.NewValue}
	}
	jsonResponse(w, http.StatusOK, syncPreviewResp{Changes: out, Count: len(changes)})
}

// Apply handles POST /api/v1/sync/jobs/{id}/apply.
// TODO: gate this behind an auth/admin middleware once GALACTICA_AUTH_ENABLED is wired up.
func (h *SyncHandler) Apply(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := storage.GetSyncJob(h.DB, id)
	if err != nil || job == nil {
		jsonError(w, http.StatusNotFound, "sync job not found")
		return
	}
	src, tgt, syncable, err := h.resolveSyncPair(r, job)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	srcSnap, err := syncable.Snapshot(r.Context(), src)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "snapshot source: "+err.Error())
		return
	}
	tgtSnap, err := syncable.Snapshot(r.Context(), tgt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "snapshot target: "+err.Error())
		return
	}
	changes := syncable.Diff(srcSnap, tgtSnap)
	if len(changes) == 0 {
		jsonResponse(w, http.StatusOK, map[string]any{"applied": 0, "message": "already in sync"})
		return
	}
	if err := syncable.Apply(r.Context(), tgt, changes); err != nil {
		jsonError(w, http.StatusInternalServerError, "apply: "+err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"applied": len(changes)})
}

// resolveSyncPair loads source and target instances plus the Syncable provider for a job.
func (h *SyncHandler) resolveSyncPair(r *http.Request, job *storage.SyncJobRow) (providers.Instance, providers.Instance, providers.Syncable, error) {
	src, err := h.resolve(r.Context(), job.SourceInstanceID)
	if err != nil {
		return providers.Instance{}, providers.Instance{}, nil, fmt.Errorf("resolve source: %w", err)
	}
	tgt, err := h.resolve(r.Context(), job.TargetInstanceID)
	if err != nil {
		return providers.Instance{}, providers.Instance{}, nil, fmt.Errorf("resolve target: %w", err)
	}
	p, err := providers.Get(src.Kind)
	if err != nil {
		return providers.Instance{}, providers.Instance{}, nil, fmt.Errorf("get provider: %w", err)
	}
	syncable, ok := p.(providers.Syncable)
	if !ok {
		return providers.Instance{}, providers.Instance{}, nil, fmt.Errorf("provider does not support sync")
	}
	return src, tgt, syncable, nil
}
