// internal/api/notify.go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/t0mer/galactica/internal/notify"
	"github.com/t0mer/galactica/internal/storage"
)

// NotifyHandler handles /api/v1/notify/channels routes.
type NotifyHandler struct{ *Deps }

type channelResp struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Provider        string `json:"provider"`
	Enabled         bool   `json:"enabled"`
	NotifyOnSuccess bool   `json:"notify_on_success"`
	NotifyOnFailure bool   `json:"notify_on_failure"`
	CreatedAt       string `json:"created_at"`
}

func toChannelResp(r storage.NotifyChannelRow) channelResp {
	return channelResp{
		ID:              r.ID,
		Name:            r.Name,
		Provider:        r.Provider,
		Enabled:         r.Enabled,
		NotifyOnSuccess: r.NotifyOnSuccess,
		NotifyOnFailure: r.NotifyOnFailure,
		CreatedAt:       r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type createChannelReq struct {
	Name            string               `json:"name"`
	Provider        string               `json:"provider"`
	Config          notify.ChannelConfig `json:"config"`
	Enabled         bool                 `json:"enabled"`
	NotifyOnSuccess bool                 `json:"notify_on_success"`
	NotifyOnFailure bool                 `json:"notify_on_failure"`
}

type testChannelReq struct {
	Provider string               `json:"provider"`
	Config   notify.ChannelConfig `json:"config"`
}

// List returns all notification channels (credentials never returned).
func (h *NotifyHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := storage.ListNotifyChannels(h.DB)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]channelResp, len(rows))
	for i, row := range rows {
		out[i] = toChannelResp(row)
	}
	jsonResponse(w, http.StatusOK, out)
}

// Create adds a new notification channel.
func (h *NotifyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createChannelReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" || req.Provider == "" {
		jsonError(w, http.StatusBadRequest, "name and provider are required")
		return
	}
	var configEnc []byte
	if h.SecretKey != "" {
		b, err := json.Marshal(req.Config)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		enc, err := storage.Encrypt(b, h.SecretKey)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "encrypt config: "+err.Error())
			return
		}
		configEnc = enc
	}
	id := uuid.New().String()
	row := storage.NotifyChannelRow{
		ID:              id,
		Name:            req.Name,
		Provider:        req.Provider,
		ConfigEncrypted: configEnc,
		Enabled:         req.Enabled,
		NotifyOnSuccess: req.NotifyOnSuccess,
		NotifyOnFailure: req.NotifyOnFailure,
	}
	if err := storage.InsertNotifyChannel(h.DB, row); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	saved, err := storage.GetNotifyChannel(h.DB, id)
	if err != nil || saved == nil {
		jsonError(w, http.StatusInternalServerError, "channel created but could not be retrieved")
		return
	}
	jsonResponse(w, http.StatusCreated, toChannelResp(*saved))
}

// Update replaces the mutable fields of a notification channel.
func (h *NotifyHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := storage.GetNotifyChannel(h.DB, id)
	if err != nil || existing == nil {
		jsonError(w, http.StatusNotFound, "channel not found")
		return
	}
	var req createChannelReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	var configEnc []byte
	if h.SecretKey != "" {
		b, err := json.Marshal(req.Config)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		enc, err := storage.Encrypt(b, h.SecretKey)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "encrypt config: "+err.Error())
			return
		}
		configEnc = enc
	}
	updated := storage.NotifyChannelRow{
		ID:              id,
		Name:            req.Name,
		ConfigEncrypted: configEnc,
		Enabled:         req.Enabled,
		NotifyOnSuccess: req.NotifyOnSuccess,
		NotifyOnFailure: req.NotifyOnFailure,
	}
	if err := storage.UpdateNotifyChannel(h.DB, updated); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	saved, err := storage.GetNotifyChannel(h.DB, id)
	if err != nil || saved == nil {
		jsonError(w, http.StatusInternalServerError, "channel updated but could not be retrieved")
		return
	}
	jsonResponse(w, http.StatusOK, toChannelResp(*saved))
}

// Delete removes a notification channel by ID.
func (h *NotifyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := storage.GetNotifyChannel(h.DB, id)
	if err != nil || existing == nil {
		jsonError(w, http.StatusNotFound, "channel not found")
		return
	}
	if err := storage.DeleteNotifyChannel(h.DB, id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// TestSend fires a real test message without persisting the channel config.
func (h *NotifyHandler) TestSend(w http.ResponseWriter, r *http.Request) {
	var req testChannelReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := notify.Send(r.Context(), req.Provider, req.Config, "Galactica test notification"); err != nil {
		jsonResponse(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"ok": true})
}
