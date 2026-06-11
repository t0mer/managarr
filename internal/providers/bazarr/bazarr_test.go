// internal/providers/bazarr/bazarr_test.go
package bazarr_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/galactica/internal/providers"
	_ "github.com/t0mer/galactica/internal/providers/bazarr"
)

func TestBazarrRegistered(t *testing.T) {
	p, err := providers.Get(providers.KindBazarr)
	require.NoError(t, err)
	assert.Equal(t, providers.KindBazarr, p.Kind())
}

// newMockBazarr starts a test HTTP server that responds to the Bazarr v1 API paths.
func newMockBazarr(t *testing.T) (*httptest.Server, providers.Instance) {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"bazarr_version": "1.4.0"}) //nolint:errcheck
	})

	mux.HandleFunc("/api/system/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{ //nolint:errcheck
			{"type": "INFO", "timestamp": time.Now().UTC().Format("2006-01-02 15:04:05"), "message": "subtitle downloaded"},
			{"type": "ERROR", "timestamp": time.Now().UTC().Format("2006-01-02 15:04:05"), "message": "provider failed", "exception": "timeout"},
		})
	})

	mux.HandleFunc("/api/badges", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"episodes": 5, "movies": 3, "providers": 12}) //nolint:errcheck
	})

	mux.HandleFunc("/api/history/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"num_downloaded": 100, "num_failed": 2}) //nolint:errcheck
	})

	mux.HandleFunc("/api/system/backups", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			if r.URL.Query().Get("filename") != "" {
				// Download — return fake ZIP bytes.
				w.Header().Set("Content-Type", "application/zip")
				w.Write([]byte("PK\x03\x04fake-zip-content")) //nolint:errcheck
				return
			}
			// List backups.
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]any{ //nolint:errcheck
				{"filename": "bazarr_backup_20260611.zip", "size": 2048},
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv, providers.Instance{
		ID:      "bazarr-1",
		Kind:    providers.KindBazarr,
		Name:    "Bazarr",
		BaseURL: srv.URL,
		APIKey:  "testkey",
	}
}

func TestBazarrTestConnection(t *testing.T) {
	_, inst := newMockBazarr(t)
	p, err := providers.Get(providers.KindBazarr)
	require.NoError(t, err)
	assert.NoError(t, p.TestConnection(context.Background(), inst))
}

func TestBazarrFetchLogs(t *testing.T) {
	_, inst := newMockBazarr(t)
	p, err := providers.Get(providers.KindBazarr)
	require.NoError(t, err)
	ls, ok := p.(providers.LogSource)
	require.True(t, ok, "Bazarr must implement LogSource")

	entries, err := ls.FetchLogs(context.Background(), inst, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "info", entries[0].Level)
	assert.Equal(t, "error", entries[1].Level)
	assert.Contains(t, entries[1].Message, "timeout")
}

func TestBazarrFetchLogsFiltersOldEntries(t *testing.T) {
	_, inst := newMockBazarr(t)
	p, err := providers.Get(providers.KindBazarr)
	require.NoError(t, err)
	ls, ok := p.(providers.LogSource)
	require.True(t, ok)

	// Requesting only entries newer than now+1h filters out all mock entries.
	entries, err := ls.FetchLogs(context.Background(), inst, time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestBazarrStreamLogsUnsupported(t *testing.T) {
	_, inst := newMockBazarr(t)
	p, err := providers.Get(providers.KindBazarr)
	require.NoError(t, err)
	ls, ok := p.(providers.LogSource)
	require.True(t, ok)

	_, streamErr := ls.StreamLogs(context.Background(), inst)
	assert.Error(t, streamErr)
}

func TestBazarrCollect(t *testing.T) {
	_, inst := newMockBazarr(t)
	p, err := providers.Get(providers.KindBazarr)
	require.NoError(t, err)
	ms, ok := p.(providers.MetricSource)
	require.True(t, ok, "Bazarr must implement MetricSource")

	samples, err := ms.Collect(context.Background(), inst)
	require.NoError(t, err)
	require.Len(t, samples, 5)

	byMetric := make(map[string]float64, len(samples))
	for _, s := range samples {
		byMetric[s.Metric] = s.Value
	}
	assert.Equal(t, float64(5), byMetric["bazarr_wanted_episodes"])
	assert.Equal(t, float64(3), byMetric["bazarr_wanted_movies"])
	assert.Equal(t, float64(12), byMetric["bazarr_providers_total"])
	assert.Equal(t, float64(100), byMetric["bazarr_subtitles_downloaded"])
	assert.Equal(t, float64(2), byMetric["bazarr_subtitles_failed"])
}

func TestBazarrExportConfig(t *testing.T) {
	_, inst := newMockBazarr(t)
	p, err := providers.Get(providers.KindBazarr)
	require.NoError(t, err)
	cb, ok := p.(providers.ConfigBackup)
	require.True(t, ok, "Bazarr must implement ConfigBackup")

	blob, err := cb.ExportConfig(context.Background(), inst)
	require.NoError(t, err)
	assert.NotEmpty(t, blob.Data)
	assert.Equal(t, "application/zip", blob.ContentType)
	assert.Equal(t, "bazarr_backup_20260611.zip", blob.Filename)
}
