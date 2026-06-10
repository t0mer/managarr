// internal/api/plex_stats.go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/t0mer/galactica/internal/providers"
)

// PlexStatsHandler handles GET /api/v1/instances/{id}/plex/stats.
type PlexStatsHandler struct{ *Deps }

type plexLibrary struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	Type     string `json:"type"` // "movie" | "show"
	Count    int    `json:"count,omitempty"`    // movie libraries
	Shows    int    `json:"shows,omitempty"`    // show libraries
	Seasons  int    `json:"seasons,omitempty"`  // show libraries
	Episodes int    `json:"episodes,omitempty"` // show libraries
}

type plexStatsResp struct {
	ServerName     string        `json:"server_name"`
	ActiveSessions int           `json:"active_sessions"`
	Libraries      []plexLibrary `json:"libraries"`
}

func (h *PlexStatsHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	inst, err := h.resolve(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}
	if inst.Kind != providers.KindPlex {
		jsonError(w, http.StatusBadRequest, "instance is not a Plex server")
		return
	}

	resp, err := fetchPlexStats(r.Context(), inst)
	if err != nil {
		jsonError(w, http.StatusBadGateway, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, resp)
}

func fetchPlexStats(ctx context.Context, inst providers.Instance) (*plexStatsResp, error) {
	doGet := func(path string, v any) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, inst.BaseURL+path, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Plex-Token", inst.APIKey)
		req.Header.Set("Accept", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("plex %s: HTTP %d", path, resp.StatusCode)
		}
		return json.NewDecoder(resp.Body).Decode(v)
	}

	// Server identity + active sessions in parallel.
	var serverName string
	var activeSessions int
	var sections []struct {
		Key   string `json:"key"`
		Title string `json:"title"`
		Type  string `json:"type"`
	}

	var identErr, sessErr, sectErr error
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		var v struct {
			MediaContainer struct {
				FriendlyName string `json:"friendlyName"`
			} `json:"MediaContainer"`
		}
		if identErr = doGet("/", &v); identErr == nil {
			serverName = v.MediaContainer.FriendlyName
		}
	}()

	go func() {
		defer wg.Done()
		var v struct {
			MediaContainer struct {
				Size int `json:"size"`
			} `json:"MediaContainer"`
		}
		if sessErr = doGet("/status/sessions", &v); sessErr == nil {
			activeSessions = v.MediaContainer.Size
		}
	}()

	go func() {
		defer wg.Done()
		var v struct {
			MediaContainer struct {
				Directory []struct {
					Key   string `json:"key"`
					Title string `json:"title"`
					Type  string `json:"type"`
				} `json:"Directory"`
			} `json:"MediaContainer"`
		}
		if sectErr = doGet("/library/sections", &v); sectErr == nil {
			sections = v.MediaContainer.Directory
		}
	}()

	wg.Wait()

	if identErr != nil && sessErr != nil && sectErr != nil {
		return nil, fmt.Errorf("all Plex calls failed: %v", identErr)
	}

	// For each section fetch counts in parallel.
	libs := make([]plexLibrary, len(sections))
	var libWg sync.WaitGroup

	countSection := func(path string) int {
		var v struct {
			MediaContainer struct {
				TotalSize int `json:"totalSize"`
				Size      int `json:"size"`
			} `json:"MediaContainer"`
		}
		if err := doGet(path+"?X-Plex-Container-Start=0&X-Plex-Container-Size=0", &v); err != nil {
			return 0
		}
		if v.MediaContainer.TotalSize > 0 {
			return v.MediaContainer.TotalSize
		}
		return v.MediaContainer.Size
	}

	for i, sec := range sections {
		i, sec := i, sec
		libWg.Add(1)
		go func() {
			defer libWg.Done()
			lib := plexLibrary{Key: sec.Key, Title: sec.Title, Type: sec.Type}
			base := "/library/sections/" + sec.Key + "/all"
			switch sec.Type {
			case "movie":
				lib.Count = countSection(base)
			case "show":
				var swg sync.WaitGroup
				swg.Add(3)
				go func() { defer swg.Done(); lib.Shows = countSection(base + "?type=2") }()
				go func() { defer swg.Done(); lib.Seasons = countSection(base + "?type=3") }()
				go func() { defer swg.Done(); lib.Episodes = countSection(base + "?type=4") }()
				swg.Wait()
			}
			libs[i] = lib
		}()
	}
	libWg.Wait()

	return &plexStatsResp{
		ServerName:     serverName,
		ActiveSessions: activeSessions,
		Libraries:      libs,
	}, nil
}
