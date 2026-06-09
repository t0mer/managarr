// internal/providers/servarr/sonarr/sonarr_test.go
package sonarr_test

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
	_ "github.com/t0mer/galactica/internal/providers/servarr/sonarr"
)

func TestSonarrRegistered(t *testing.T) {
	p, err := providers.Get(providers.KindSonarr)
	require.NoError(t, err)
	assert.Equal(t, providers.KindSonarr, p.Kind())
}

// newMockSonarr starts a test HTTP server that responds to the Sonarr v3 API paths.
func newMockSonarr(t *testing.T) (*httptest.Server, providers.Instance) {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/system/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": "4.0.0"})
	})

	mux.HandleFunc("/api/v3/log", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"records": []map[string]string{
				{"time": time.Now().UTC().Format(time.RFC3339), "level": "info", "logger": "test", "message": "hello"},
				{"time": time.Now().UTC().Format(time.RFC3339), "level": "error", "logger": "test", "message": "oops"},
			},
		})
	})

	mux.HandleFunc("/api/v3/series", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "title": "ShowA", "monitored": true, "statistics": map[string]any{"episodeFileCount": 3, "episodeCount": 5}},
		})
	})

	mux.HandleFunc("/api/v3/queue", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"totalRecords": 2})
	})

	mux.HandleFunc("/api/v3/qualityProfile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "name": "HD-1080p"}})
	})

	mux.HandleFunc("/api/v3/config/naming", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"standardEpisodeFormat": "{Series} S{Season}E{Episode}"})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv, providers.Instance{
		ID:      "sonarr-1",
		Kind:    providers.KindSonarr,
		Name:    "Sonarr",
		BaseURL: srv.URL,
		APIKey:  "testkey",
	}
}

func TestSonarrTestConnection(t *testing.T) {
	_, inst := newMockSonarr(t)
	p, err := providers.Get(providers.KindSonarr)
	require.NoError(t, err)
	assert.NoError(t, p.TestConnection(context.Background(), inst))
}

func TestSonarrFetchLogsReturnsDeterministicEntries(t *testing.T) {
	_, inst := newMockSonarr(t)
	p, err := providers.Get(providers.KindSonarr)
	require.NoError(t, err)
	ls, ok := p.(providers.LogSource)
	require.True(t, ok, "Sonarr must implement LogSource")

	a, err := ls.FetchLogs(context.Background(), inst, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotEmpty(t, a)

	b, err := ls.FetchLogs(context.Background(), inst, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, len(a), len(b))
}

func TestSonarrCollectReturnsSamples(t *testing.T) {
	_, inst := newMockSonarr(t)
	p, err := providers.Get(providers.KindSonarr)
	require.NoError(t, err)
	ms, ok := p.(providers.MetricSource)
	require.True(t, ok, "Sonarr must implement MetricSource")

	samples, err := ms.Collect(context.Background(), inst)
	require.NoError(t, err)
	assert.NotEmpty(t, samples)
	for _, s := range samples {
		assert.NotEmpty(t, s.Metric)
	}
}

func TestSonarrExportConfig(t *testing.T) {
	_, inst := newMockSonarr(t)
	p, err := providers.Get(providers.KindSonarr)
	require.NoError(t, err)
	cb, ok := p.(providers.ConfigBackup)
	require.True(t, ok)

	blob, err := cb.ExportConfig(context.Background(), inst)
	require.NoError(t, err)
	assert.NotEmpty(t, blob.Data)
}

func TestSonarrSyncDiff(t *testing.T) {
	_, inst := newMockSonarr(t)
	p, err := providers.Get(providers.KindSonarr)
	require.NoError(t, err)
	sy, ok := p.(providers.Syncable)
	require.True(t, ok)

	a, err := sy.Snapshot(context.Background(), inst)
	require.NoError(t, err)
	b, err := sy.Snapshot(context.Background(), inst)
	require.NoError(t, err)
	// Identical snapshots produce no changes (nil or empty slice are both valid).
	changes := sy.Diff(a, b)
	assert.Empty(t, changes)
}
