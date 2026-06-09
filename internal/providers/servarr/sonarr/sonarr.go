// internal/providers/servarr/sonarr/sonarr.go
package sonarr

import (
	"context"
	"math/rand"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

func init() { providers.Register(&Sonarr{}) }

type Sonarr struct{}

func (s *Sonarr) Kind() providers.Kind { return providers.KindSonarr }

func (s *Sonarr) TestConnection(_ context.Context, _ providers.Instance) error { return nil }

func (s *Sonarr) FetchLogs(_ context.Context, inst providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	rng := rand.New(rand.NewSource(seedFromID(inst.ID)))
	levels := []string{"info", "warn", "error", "debug"}
	messages := []string{
		"Series scan completed", "Indexer search failed",
		"Download grabbed from indexer", "Import failed: no space left on device",
		"Episode upgraded to better quality", "Refresh series triggered",
		"Database cleanup completed", "RSS sync completed",
	}
	entries := make([]providers.LogEntry, 30)
	base := time.Now()
	for i := range entries {
		entries[i] = providers.LogEntry{
			Timestamp: base.Add(-time.Duration(rng.Intn(3600)) * time.Second),
			Level:     levels[rng.Intn(len(levels))],
			Source:    "Sonarr",
			Message:   messages[rng.Intn(len(messages))],
		}
	}
	return entries, nil
}

func (s *Sonarr) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	ch := make(chan providers.LogEntry)
	close(ch)
	return ch, nil
}

func (s *Sonarr) Collect(_ context.Context, _ providers.Instance) ([]providers.Sample, error) {
	now := time.Now()
	return []providers.Sample{
		{Metric: "sonarr_series_total", Value: 312, TS: now},
		{Metric: "sonarr_series_monitored", Value: 289, TS: now},
		{Metric: "sonarr_missing_episodes", Value: 18, TS: now},
		{Metric: "sonarr_queue_total", Value: 4, TS: now},
	}, nil
}

func (s *Sonarr) ExportConfig(_ context.Context, _ providers.Instance) (providers.ConfigBlob, error) {
	return providers.ConfigBlob{
		ContentType: "application/json",
		Data:        []byte(`{"qualityProfiles":[],"namingConvention":{},"rootFolders":[]}`),
	}, nil
}

func (s *Sonarr) ImportConfig(_ context.Context, _ providers.Instance, _ providers.ConfigBlob) error {
	return nil
}

func (s *Sonarr) Snapshot(_ context.Context, _ providers.Instance) (providers.SyncState, error) {
	return providers.SyncState{Data: map[string]any{
		"qualityProfiles": []string{"HD-1080p", "4K"},
		"namingFormat":    "{Series} S{season:00}E{episode:00} - {Episode}",
	}}, nil
}

func (s *Sonarr) Diff(_, _ providers.SyncState) []providers.SyncChange {
	return []providers.SyncChange{
		{Field: "qualityProfiles[0].name", OldValue: "HD-720p", NewValue: "HD-1080p"},
		{Field: "namingFormat", OldValue: "{Series} - S{season:00}E{episode:00}", NewValue: "{Series} S{season:00}E{episode:00} - {Episode}"},
	}
}

func (s *Sonarr) Apply(_ context.Context, _ providers.Instance, _ []providers.SyncChange) error {
	return nil
}

func seedFromID(id string) int64 {
	var h int64 = 17
	for _, c := range id {
		h = h*31 + int64(c)
	}
	return h
}
