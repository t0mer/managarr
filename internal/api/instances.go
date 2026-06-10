// internal/api/instances.go
package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/t0mer/galactica/internal/providers"
	"github.com/t0mer/galactica/internal/storage"
)

// storeSecret encrypts value and stores it under key for the instance.
// When no secretKey is configured it stores as plaintext with a warning.
func storeSecret(db *sql.DB, secretKey, instanceID, key, value string, log *slog.Logger) error {
	var toStore []byte
	if secretKey != "" {
		enc, err := storage.Encrypt([]byte(value), secretKey)
		if err != nil {
			return err
		}
		toStore = enc
	} else {
		log.Warn("GALACTICA_SECRET_KEY not set — storing secret as plaintext; set a secret key for encrypted storage",
			"key", key)
		toStore = []byte(value)
	}
	return storage.PutSecret(db, instanceID, key, toStore)
}

// InstancesHandler handles /api/v1/instances routes.
type InstancesHandler struct{ *Deps }

type createInstanceReq struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Username string `json:"username"`
	Password string `json:"password"`
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
	if req.APIKey != "" {
		if err := storeSecret(h.DB, h.SecretKey, id, "api_key", req.APIKey, h.Log); err != nil {
			h.Log.Warn("storing api_key", "instance_id", id, "error", err)
		}
	}
	if req.Username != "" {
		if err := storeSecret(h.DB, h.SecretKey, id, "username", req.Username, h.Log); err != nil {
			h.Log.Warn("storing username", "instance_id", id, "error", err)
		}
	}
	if req.Password != "" {
		if err := storeSecret(h.DB, h.SecretKey, id, "password", req.Password, h.Log); err != nil {
			h.Log.Warn("storing password", "instance_id", id, "error", err)
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
	if req.APIKey != "" {
		if err := storeSecret(h.DB, h.SecretKey, id, "api_key", req.APIKey, h.Log); err != nil {
			h.Log.Warn("storing api_key", "instance_id", id, "error", err)
		}
	}
	if req.Username != "" {
		if err := storeSecret(h.DB, h.SecretKey, id, "username", req.Username, h.Log); err != nil {
			h.Log.Warn("storing username", "instance_id", id, "error", err)
		}
	}
	if req.Password != "" {
		if err := storeSecret(h.DB, h.SecretKey, id, "password", req.Password, h.Log); err != nil {
			h.Log.Warn("storing password", "instance_id", id, "error", err)
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
