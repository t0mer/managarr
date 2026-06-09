// internal/providers/servarr/radarr/radarr.go
package radarr

import (
	"context"
	"math/rand"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

func init() { providers.Register(&Radarr{}) }

type Radarr struct{}

func (r *Radarr) Kind() providers.Kind { return providers.KindRadarr }

func (r *Radarr) TestConnection(_ context.Context, _ providers.Instance) error { return nil }

func (r *Radarr) FetchLogs(_ context.Context, inst providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	rng := rand.New(rand.NewSource(seedFromID(inst.ID)))
	levels := []string{"info", "warn", "error", "debug"}
	messages := []string{
		"Movie scan completed", "Download grabbed", "Import failed: unknown movie",
		"Upgrade found for movie", "Refresh movie triggered", "Indexer returned no results",
	}
	entries := make([]providers.LogEntry, 30)
	base := time.Now()
	for i := range entries {
		entries[i] = providers.LogEntry{
			Timestamp: base.Add(-time.Duration(rng.Intn(3600)) * time.Second),
			Level:     levels[rng.Intn(len(levels))],
			Source:    "Radarr",
			Message:   messages[rng.Intn(len(messages))],
		}
	}
	return entries, nil
}

func (r *Radarr) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	ch := make(chan providers.LogEntry)
	close(ch)
	return ch, nil
}

func (r *Radarr) Collect(_ context.Context, _ providers.Instance) ([]providers.Sample, error) {
	now := time.Now()
	return []providers.Sample{
		{Metric: "radarr_movies_total", Value: 847, TS: now},
		{Metric: "radarr_movies_monitored", Value: 721, TS: now},
		{Metric: "radarr_missing_movies", Value: 33, TS: now},
		{Metric: "radarr_queue_total", Value: 2, TS: now},
	}, nil
}

func (r *Radarr) ExportConfig(_ context.Context, _ providers.Instance) (providers.ConfigBlob, error) {
	return providers.ConfigBlob{
		ContentType: "application/json",
		Data:        []byte(`{"qualityProfiles":[],"namingConvention":{},"rootFolders":[]}`),
	}, nil
}

func (r *Radarr) ImportConfig(_ context.Context, _ providers.Instance, _ providers.ConfigBlob) error {
	return nil
}

func (r *Radarr) Snapshot(_ context.Context, _ providers.Instance) (providers.SyncState, error) {
	return providers.SyncState{Data: map[string]any{
		"qualityProfiles": []string{"HD-1080p", "4K-HDR"},
	}}, nil
}

func (r *Radarr) Diff(_, _ providers.SyncState) []providers.SyncChange {
	return []providers.SyncChange{
		{Field: "qualityProfiles[1].name", OldValue: "4K", NewValue: "4K-HDR"},
	}
}

func (r *Radarr) Apply(_ context.Context, _ providers.Instance, _ []providers.SyncChange) error {
	return nil
}

func seedFromID(id string) int64 {
	var h int64 = 17
	for _, c := range id {
		h = h*31 + int64(c)
	}
	return h
}
