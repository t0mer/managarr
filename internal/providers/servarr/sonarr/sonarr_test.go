// internal/providers/servarr/sonarr/sonarr_test.go
package sonarr_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/galactica/internal/providers"
	_ "github.com/t0mer/galactica/internal/providers/servarr/sonarr"
)

func inst() providers.Instance {
	return providers.Instance{ID: "sonarr-1", Kind: providers.KindSonarr, Name: "Sonarr", BaseURL: "http://localhost:8989"}
}

func TestSonarrRegistered(t *testing.T) {
	p, err := providers.Get(providers.KindSonarr)
	require.NoError(t, err)
	assert.Equal(t, providers.KindSonarr, p.Kind())
}

func TestSonarrTestConnection(t *testing.T) {
	p, _ := providers.Get(providers.KindSonarr)
	assert.NoError(t, p.TestConnection(context.Background(), inst()))
}

func TestSonarrFetchLogsReturnsDeterministicEntries(t *testing.T) {
	p, _ := providers.Get(providers.KindSonarr)
	ls, ok := p.(providers.LogSource)
	require.True(t, ok, "Sonarr must implement LogSource")

	a, err := ls.FetchLogs(context.Background(), inst(), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotEmpty(t, a)

	b, _ := ls.FetchLogs(context.Background(), inst(), time.Now().Add(-time.Hour))
	assert.Equal(t, len(a), len(b))
}

func TestSonarrCollectReturnsSamples(t *testing.T) {
	p, _ := providers.Get(providers.KindSonarr)
	ms, ok := p.(providers.MetricSource)
	require.True(t, ok, "Sonarr must implement MetricSource")

	samples, err := ms.Collect(context.Background(), inst())
	require.NoError(t, err)
	assert.NotEmpty(t, samples)
	for _, s := range samples {
		assert.NotEmpty(t, s.Metric)
	}
}

func TestSonarrExportConfig(t *testing.T) {
	p, _ := providers.Get(providers.KindSonarr)
	cb, ok := p.(providers.ConfigBackup)
	require.True(t, ok)

	blob, err := cb.ExportConfig(context.Background(), inst())
	require.NoError(t, err)
	assert.NotEmpty(t, blob.Data)
}

func TestSonarrSyncDiff(t *testing.T) {
	p, _ := providers.Get(providers.KindSonarr)
	sy, ok := p.(providers.Syncable)
	require.True(t, ok)

	a, _ := sy.Snapshot(context.Background(), inst())
	b, _ := sy.Snapshot(context.Background(), inst())
	changes := sy.Diff(a, b)
	assert.NotNil(t, changes)
}
