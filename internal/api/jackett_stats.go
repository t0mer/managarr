// internal/api/jackett_stats.go
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/t0mer/galactica/internal/providers"
	"github.com/t0mer/galactica/internal/storage"
)

// JackettStatsHandler handles Jackett-specific routes.
type JackettStatsHandler struct{ *Deps }

type jackettIndexerResp struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Configured bool   `json:"configured"`
	Monitored  bool   `json:"monitored"`
	TestStatus string `json:"test_status"` // "ok" | "error" | "skipped"
	TestError  string `json:"test_error,omitempty"`
}

type jackettStatsResp struct {
	Indexers   []jackettIndexerResp `json:"indexers"`
	Total      int                  `json:"total"`
	Configured int                  `json:"configured"`
	OK         int                  `json:"ok"`
	Error      int                  `json:"error"`
}

var jackettCli = &http.Client{Timeout: 10 * time.Second}

// Stats handles GET /api/v1/instances/{id}/jackett/stats.
func (h *JackettStatsHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	inst, err := h.resolve(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}
	if inst.Kind != providers.KindJackett {
		jsonError(w, http.StatusBadRequest, "instance is not Jackett")
		return
	}

	resp, err := fetchJackettStats(r.Context(), h.DB, inst)
	if err != nil {
		jsonError(w, http.StatusBadGateway, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, resp)
}

// SetMonitored handles PATCH /api/v1/instances/{id}/jackett/indexers/{indexer_id}.
func (h *JackettStatsHandler) SetMonitored(w http.ResponseWriter, r *http.Request) {
	instanceID := chi.URLParam(r, "id")
	indexerID := chi.URLParam(r, "indexer_id")

	var body struct {
		Monitored bool `json:"monitored"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := storage.SetIndexerMonitored(h.DB, instanceID, indexerID, body.Monitored); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── XML types for Torznab indexer listing ────────────────────────────────────

type xmlJackettIndexer struct {
	ID         string `xml:"id,attr"`
	Configured string `xml:"configured,attr"`
	Title      string `xml:"title"`
}

type xmlJackettIndexers struct {
	Indexers []xmlJackettIndexer `xml:"indexer"`
}

// ── helpers ───────────────────────────────────────────────────────────────────

func torznabFetch(ctx context.Context, baseURL, apiKey, indexerID, t string) ([]byte, error) {
	base := strings.TrimRight(baseURL, "/")
	path := "/api/v2.0/indexers/all/results/torznab"
	if indexerID != "" {
		path = fmt.Sprintf("/api/v2.0/indexers/%s/results/torznab", indexerID)
	}
	u, err := url.Parse(base + path)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("t", t)
	if apiKey != "" {
		q.Set("apikey", apiKey)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := jackettCli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func listJackettIndexers(ctx context.Context, inst providers.Instance) ([]xmlJackettIndexer, error) {
	data, err := torznabFetch(ctx, inst.BaseURL, inst.APIKey, "", "indexers")
	if err != nil {
		return nil, err
	}
	var result xmlJackettIndexers
	if err := xml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing indexers XML: %w", err)
	}
	return result.Indexers, nil
}

func testJackettIndexer(ctx context.Context, inst providers.Instance, indexerID string) error {
	tctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	_, err := torznabFetch(tctx, inst.BaseURL, inst.APIKey, indexerID, "caps")
	return err
}

func fetchJackettStats(ctx context.Context, db *sql.DB, inst providers.Instance) (*jackettStatsResp, error) {
	// Load per-indexer monitored overrides from DB.
	monitored, err := storage.ListIndexerMonitored(db, inst.ID)
	if err != nil {
		return nil, fmt.Errorf("loading indexer settings: %w", err)
	}

	// Fetch the full indexer list from Jackett.
	xmlIndexers, err := listJackettIndexers(ctx, inst)
	if err != nil {
		return nil, fmt.Errorf("fetching indexers: %w", err)
	}

	// Test each configured + monitored indexer in parallel.
	results := make([]jackettIndexerResp, len(xmlIndexers))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, xi := range xmlIndexers {
		configured := xi.Configured == "true"

		// Monitored defaults to true unless explicitly set to false in DB.
		mon := true
		if v, ok := monitored[xi.ID]; ok {
			mon = v
		}

		results[i] = jackettIndexerResp{
			ID:         xi.ID,
			Name:       xi.Title,
			Configured: configured,
			Monitored:  mon,
			TestStatus: "skipped",
		}

		if !configured || !mon {
			continue
		}

		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()
			err := testJackettIndexer(ctx, inst, id)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results[idx].TestStatus = "error"
				results[idx].TestError = err.Error()
			} else {
				results[idx].TestStatus = "ok"
			}
		}(i, xi.ID)
	}
	wg.Wait()

	resp := &jackettStatsResp{Indexers: results, Total: len(results)}
	for _, r := range results {
		if r.Configured {
			resp.Configured++
		}
		switch r.TestStatus {
		case "ok":
			resp.OK++
		case "error":
			resp.Error++
		}
	}
	return resp, nil
}
