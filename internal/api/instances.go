// internal/api/instances.go
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/t0mer/galactica/internal/providers"
	"github.com/t0mer/galactica/internal/storage"
)

// InstancesHandler handles /api/v1/instances routes.
type InstancesHandler struct{ *Deps }

type createInstanceReq struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

type instanceResp struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	BaseURL   string `json:"base_url"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

func toInstanceResp(row storage.InstanceRow) instanceResp {
	return instanceResp{
		ID:        row.ID,
		Kind:      row.Kind,
		Name:      row.Name,
		BaseURL:   row.BaseURL,
		Enabled:   row.Enabled,
		CreatedAt: row.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (h *InstancesHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := storage.ListInstances(h.DB)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]instanceResp, len(rows))
	for i, row := range rows {
		out[i] = toInstanceResp(row)
	}
	jsonResponse(w, http.StatusOK, out)
}

func (h *InstancesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createInstanceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Kind == "" || req.Name == "" || req.BaseURL == "" {
		jsonError(w, http.StatusBadRequest, "kind, name, and base_url are required")
		return
	}
	if _, err := providers.Get(providers.Kind(req.Kind)); err != nil {
		jsonError(w, http.StatusBadRequest, "unsupported kind: "+req.Kind)
		return
	}
	id := uuid.New().String()
	if err := storage.InsertInstance(h.DB, id, req.Kind, req.Name, req.BaseURL); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.APIKey != "" && h.SecretKey != "" {
		enc, err := storage.Encrypt([]byte(req.APIKey), h.SecretKey)
		if err == nil {
			_ = storage.PutSecret(h.DB, id, "api_key", enc)
		}
	}
	row, err2 := storage.GetInstance(h.DB, id)
	if err2 != nil || row == nil {
		jsonError(w, http.StatusInternalServerError, "instance created but could not be retrieved")
		return
	}
	jsonResponse(w, http.StatusCreated, toInstanceResp(*row))
}

func (h *InstancesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	row, err := storage.GetInstance(h.DB, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if row == nil {
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}
	jsonResponse(w, http.StatusOK, toInstanceResp(*row))
}

func (h *InstancesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req createInstanceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	row, err := storage.GetInstance(h.DB, id)
	if err != nil || row == nil {
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}
	if err := storage.UpdateInstance(h.DB, id, req.Name, req.BaseURL, row.Enabled); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.APIKey != "" && h.SecretKey != "" {
		enc, err := storage.Encrypt([]byte(req.APIKey), h.SecretKey)
		if err == nil {
			_ = storage.PutSecret(h.DB, id, "api_key", enc)
		}
	}
	row, err = storage.GetInstance(h.DB, id)
	if err != nil || row == nil {
		jsonError(w, http.StatusInternalServerError, "instance updated but could not be retrieved")
		return
	}
	jsonResponse(w, http.StatusOK, toInstanceResp(*row))
}

func (h *InstancesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	row, err := storage.GetInstance(h.DB, id)
	if err != nil || row == nil {
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}
	if err := storage.DeleteInstance(h.DB, id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *InstancesHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	inst, err := h.resolve(r.Context(), id)
	if errors.Is(err, errNotFound) {
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	p, err := providers.Get(inst.Kind)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "provider not registered")
		return
	}
	if err := p.TestConnection(r.Context(), inst); err != nil {
		jsonResponse(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *InstancesHandler) SetEnabled(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	row, err := storage.GetInstance(h.DB, id)
	if err != nil || row == nil {
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}
	if err := storage.SetInstanceEnabled(h.DB, id, body.Enabled); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
