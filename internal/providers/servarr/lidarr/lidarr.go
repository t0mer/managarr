// internal/providers/servarr/lidarr/lidarr.go
package lidarr

import (
	"context"
	"math/rand"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

func init() { providers.Register(&Lidarr{}) }

type Lidarr struct{}

func (l *Lidarr) Kind() providers.Kind { return providers.KindLidarr }

func (l *Lidarr) TestConnection(_ context.Context, _ providers.Instance) error { return nil }

func (l *Lidarr) FetchLogs(_ context.Context, inst providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	rng := rand.New(rand.NewSource(seedFromID(inst.ID)))
	levels := []string{"info", "warn", "error", "debug"}
	messages := []string{
		"Artist scan completed", "Album grabbed", "Track import failed",
		"Metadata refresh triggered", "Indexer search returned 0 results",
	}
	entries := make([]providers.LogEntry, 25)
	base := time.Now()
	for i := range entries {
		entries[i] = providers.LogEntry{
			Timestamp: base.Add(-time.Duration(rng.Intn(3600)) * time.Second),
			Level:     levels[rng.Intn(len(levels))],
			Source:    "Lidarr",
			Message:   messages[rng.Intn(len(messages))],
		}
	}
	return entries, nil
}

func (l *Lidarr) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	ch := make(chan providers.LogEntry)
	close(ch)
	return ch, nil
}

func (l *Lidarr) Collect(_ context.Context, _ providers.Instance) ([]providers.Sample, error) {
	now := time.Now()
	return []providers.Sample{
		{Metric: "lidarr_artists_total", Value: 156, TS: now},
		{Metric: "lidarr_albums_total", Value: 1203, TS: now},
		{Metric: "lidarr_missing_albums", Value: 47, TS: now},
		{Metric: "lidarr_queue_total", Value: 1, TS: now},
	}, nil
}

func (l *Lidarr) ExportConfig(_ context.Context, _ providers.Instance) (providers.ConfigBlob, error) {
	return providers.ConfigBlob{
		ContentType: "application/json",
		Data:        []byte(`{"qualityProfiles":[],"rootFolders":[]}`),
	}, nil
}

func (l *Lidarr) ImportConfig(_ context.Context, _ providers.Instance, _ providers.ConfigBlob) error {
	return nil
}

func (l *Lidarr) Snapshot(_ context.Context, _ providers.Instance) (providers.SyncState, error) {
	return providers.SyncState{Data: map[string]any{"qualityProfiles": []string{"Lossless"}}}, nil
}

func (l *Lidarr) Diff(_, _ providers.SyncState) []providers.SyncChange {
	return []providers.SyncChange{
		{Field: "qualityProfiles[0].name", OldValue: "MP3-320", NewValue: "Lossless"},
	}
}

func (l *Lidarr) Apply(_ context.Context, _ providers.Instance, _ []providers.SyncChange) error {
	return nil
}

func seedFromID(id string) int64 {
	var h int64 = 17
	for _, c := range id {
		h = h*31 + int64(c)
	}
	return h
}
