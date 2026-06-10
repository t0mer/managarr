// internal/api/servarr_stats.go
package api

import (
	"context"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/t0mer/galactica/internal/providers"
	"github.com/t0mer/galactica/internal/providers/servarr"
)

// ServarrStatsHandler handles stats endpoints for Sonarr and Radarr.
type ServarrStatsHandler struct{ *Deps }

// ── Sonarr ────────────────────────────────────────────────────────────────────

type sonarrStatsResp struct {
	SeriesTotal    int `json:"series_total"`
	QueueTotal     int `json:"queue_total"`
	MissingEpisodes int `json:"missing_episodes"`
}

// SonarrStats handles GET /api/v1/instances/{id}/sonarr/stats.
func (h *ServarrStatsHandler) SonarrStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	inst, err := h.resolve(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}
	if inst.Kind != providers.KindSonarr {
		jsonError(w, http.StatusBadRequest, "instance is not Sonarr")
		return
	}

	resp, err := fetchSonarrStats(r.Context(), inst)
	if err != nil {
		jsonError(w, http.StatusBadGateway, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, resp)
}

func fetchSonarrStats(ctx context.Context, inst providers.Instance) (*sonarrStatsResp, error) {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		firstErr error
		resp    sonarrStatsResp
	)

	set := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	set(func() {
		var series []struct{}
		if err := servarr.GetJSON(ctx, inst, "/api/v3/series", &series); err != nil {
			mu.Lock(); if firstErr == nil { firstErr = err }; mu.Unlock()
			return
		}
		mu.Lock(); resp.SeriesTotal = len(series); mu.Unlock()
	})

	set(func() {
		var queue struct{ TotalRecords int `json:"totalRecords"` }
		if err := servarr.GetJSON(ctx, inst, "/api/v3/queue?pageSize=1", &queue); err != nil {
			return
		}
		mu.Lock(); resp.QueueTotal = queue.TotalRecords; mu.Unlock()
	})

	set(func() {
		var missing struct{ TotalRecords int `json:"totalRecords"` }
		if err := servarr.GetJSON(ctx, inst, "/api/v3/wanted/missing?pageSize=1", &missing); err != nil {
			return
		}
		mu.Lock(); resp.MissingEpisodes = missing.TotalRecords; mu.Unlock()
	})

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return &resp, nil
}

// ── Radarr ────────────────────────────────────────────────────────────────────

type radarrStatsResp struct {
	MoviesTotal   int `json:"movies_total"`
	MoviesOnDisk  int `json:"movies_on_disk"`
	MissingMovies int `json:"missing_movies"`
	QueueTotal    int `json:"queue_total"`
}

// RadarrStats handles GET /api/v1/instances/{id}/radarr/stats.
func (h *ServarrStatsHandler) RadarrStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	inst, err := h.resolve(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}
	if inst.Kind != providers.KindRadarr {
		jsonError(w, http.StatusBadRequest, "instance is not Radarr")
		return
	}

	resp, err := fetchRadarrStats(r.Context(), inst)
	if err != nil {
		jsonError(w, http.StatusBadGateway, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, resp)
}

// ── Lidarr ────────────────────────────────────────────────────────────────────

type lidarrStatsResp struct {
	ArtistsTotal  int `json:"artists_total"`
	AlbumsTotal   int `json:"albums_total"`
	QueueTotal    int `json:"queue_total"`
	MissingAlbums int `json:"missing_albums"`
}

// LidarrStats handles GET /api/v1/instances/{id}/lidarr/stats.
func (h *ServarrStatsHandler) LidarrStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	inst, err := h.resolve(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}
	if inst.Kind != providers.KindLidarr {
		jsonError(w, http.StatusBadRequest, "instance is not Lidarr")
		return
	}

	resp, err := fetchLidarrStats(r.Context(), inst)
	if err != nil {
		jsonError(w, http.StatusBadGateway, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, resp)
}

func fetchLidarrStats(ctx context.Context, inst providers.Instance) (*lidarrStatsResp, error) {
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		firstErr error
		resp     lidarrStatsResp
	)

	set := func(fn func()) {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}

	set(func() {
		var artists []struct{}
		if err := servarr.GetJSON(ctx, inst, "/api/v1/artist", &artists); err != nil {
			mu.Lock(); if firstErr == nil { firstErr = err }; mu.Unlock()
			return
		}
		mu.Lock(); resp.ArtistsTotal = len(artists); mu.Unlock()
	})

	set(func() {
		var albums struct{ TotalRecords int `json:"totalRecords"` }
		if err := servarr.GetJSON(ctx, inst, "/api/v1/album?pageSize=1", &albums); err != nil {
			return
		}
		mu.Lock(); resp.AlbumsTotal = albums.TotalRecords; mu.Unlock()
	})

	set(func() {
		var queue struct{ TotalRecords int `json:"totalRecords"` }
		if err := servarr.GetJSON(ctx, inst, "/api/v1/queue?pageSize=1", &queue); err != nil {
			return
		}
		mu.Lock(); resp.QueueTotal = queue.TotalRecords; mu.Unlock()
	})

	set(func() {
		var missing struct{ TotalRecords int `json:"totalRecords"` }
		if err := servarr.GetJSON(ctx, inst, "/api/v1/wanted/missing?pageSize=1", &missing); err != nil {
			return
		}
		mu.Lock(); resp.MissingAlbums = missing.TotalRecords; mu.Unlock()
	})

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return &resp, nil
}

func fetchRadarrStats(ctx context.Context, inst providers.Instance) (*radarrStatsResp, error) {
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		firstErr error
		resp     radarrStatsResp
	)

	set := func(fn func()) {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}

	set(func() {
		var movies []struct {
			HasFile   bool `json:"hasFile"`
			Monitored bool `json:"monitored"`
		}
		if err := servarr.GetJSON(ctx, inst, "/api/v3/movie", &movies); err != nil {
			mu.Lock(); if firstErr == nil { firstErr = err }; mu.Unlock()
			return
		}
		onDisk, missing := 0, 0
		for _, m := range movies {
			if m.HasFile {
				onDisk++
			} else if m.Monitored {
				missing++
			}
		}
		mu.Lock()
		resp.MoviesTotal = len(movies)
		resp.MoviesOnDisk = onDisk
		resp.MissingMovies = missing
		mu.Unlock()
	})

	set(func() {
		var queue struct{ TotalRecords int `json:"totalRecords"` }
		if err := servarr.GetJSON(ctx, inst, "/api/v3/queue?pageSize=1", &queue); err != nil {
			return
		}
		mu.Lock(); resp.QueueTotal = queue.TotalRecords; mu.Unlock()
	})

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return &resp, nil
}
