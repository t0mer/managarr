// internal/api/deluge_stats.go
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/t0mer/galactica/internal/providers"
)

// DelugeStatsHandler handles GET /api/v1/instances/{id}/deluge/stats.
type DelugeStatsHandler struct{ *Deps }

type delugeStatsResp struct {
	DownloadRate  float64 `json:"download_rate"`
	UploadRate    float64 `json:"upload_rate"`
	NumConnections int    `json:"num_connections"`
	Torrents      struct {
		Total       int `json:"total"`
		Downloading int `json:"downloading"`
		Seeding     int `json:"seeding"`
		Paused      int `json:"paused"`
		Error       int `json:"error"`
	} `json:"torrents"`
}

func (h *DelugeStatsHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	inst, err := h.resolve(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}
	if inst.Kind != providers.KindDeluge {
		jsonError(w, http.StatusBadRequest, "instance is not Deluge")
		return
	}

	resp, err := fetchDelugeStats(r.Context(), inst)
	if err != nil {
		jsonError(w, http.StatusBadGateway, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, resp)
}

func fetchDelugeStats(ctx context.Context, inst providers.Instance) (*delugeStatsResp, error) {
	jar, _ := cookiejar.New(nil)
	cli := &http.Client{Timeout: 15 * time.Second, Jar: jar}

	doRPC := func(method string, params []any) (any, error) {
		body, err := json.Marshal(map[string]any{
			"method": method,
			"params": params,
			"id":     1,
		})
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, inst.BaseURL+"/json", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := cli.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var r struct {
			Result any `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(b, &r); err != nil {
			return nil, err
		}
		if r.Error != nil {
			return nil, fmt.Errorf("deluge RPC: %s", r.Error.Message)
		}
		return r.Result, nil
	}

	// Authenticate
	authResult, err := doRPC("auth.login", []any{inst.APIKey})
	if err != nil {
		return nil, fmt.Errorf("deluge auth: %w", err)
	}
	if ok, _ := authResult.(bool); !ok {
		return nil, fmt.Errorf("deluge auth failed")
	}

	// Session status
	sessResult, err := doRPC("core.get_session_status", []any{[]string{"upload_rate", "download_rate", "num_connections"}})
	if err != nil {
		return nil, fmt.Errorf("deluge session status: %w", err)
	}
	status, _ := sessResult.(map[string]any)

	// Torrent states
	torrentResult, err := doRPC("core.get_torrents_status", []any{nil, []string{"state"}})
	if err != nil {
		return nil, fmt.Errorf("deluge torrents: %w", err)
	}

	var resp delugeStatsResp
	resp.DownloadRate, _ = status["download_rate"].(float64)
	resp.UploadRate, _ = status["upload_rate"].(float64)
	connF, _ := status["num_connections"].(float64)
	resp.NumConnections = int(connF)

	if torrents, ok := torrentResult.(map[string]any); ok {
		resp.Torrents.Total = len(torrents)
		for _, v := range torrents {
			info, _ := v.(map[string]any)
			state, _ := info["state"].(string)
			switch state {
			case "Downloading":
				resp.Torrents.Downloading++
			case "Seeding":
				resp.Torrents.Seeding++
			case "Paused":
				resp.Torrents.Paused++
			case "Error":
				resp.Torrents.Error++
			}
		}
	}

	return &resp, nil
}
