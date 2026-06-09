// internal/providers/deluge/deluge.go
package deluge

import (
	"context"
	"math/rand"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

func init() { providers.Register(&Deluge{}) }

type Deluge struct{}

func (d *Deluge) Kind() providers.Kind { return providers.KindDeluge }

func (d *Deluge) TestConnection(_ context.Context, _ providers.Instance) error { return nil }

func (d *Deluge) FetchLogs(_ context.Context, inst providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	rng := rand.New(rand.NewSource(seedFromID(inst.ID)))
	levels := []string{"info", "warn", "error", "debug"}
	messages := []string{
		"Torrent added", "Download complete", "Seeding ratio reached",
		"Disk space low", "Tracker announce success", "Move completed",
	}
	entries := make([]providers.LogEntry, 20)
	base := time.Now()
	for i := range entries {
		entries[i] = providers.LogEntry{
			Timestamp: base.Add(-time.Duration(rng.Intn(3600)) * time.Second),
			Level:     levels[rng.Intn(len(levels))],
			Source:    "Deluge",
			Message:   messages[rng.Intn(len(messages))],
		}
	}
	return entries, nil
}

func (d *Deluge) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	ch := make(chan providers.LogEntry)
	close(ch)
	return ch, nil
}

func (d *Deluge) Collect(_ context.Context, _ providers.Instance) ([]providers.Sample, error) {
	now := time.Now()
	return []providers.Sample{
		{Metric: "deluge_torrents_active", Value: 12, TS: now},
		{Metric: "deluge_torrents_seeding", Value: 9, TS: now},
		{Metric: "deluge_torrents_downloading", Value: 3, TS: now},
		{Metric: "deluge_upload_rate_bytes", Value: 1_250_000, TS: now},
		{Metric: "deluge_download_rate_bytes", Value: 4_500_000, TS: now},
		{Metric: "deluge_ratio_avg", Value: 1.87, TS: now},
	}, nil
}

func (d *Deluge) ExportConfig(_ context.Context, _ providers.Instance) (providers.ConfigBlob, error) {
	return providers.ConfigBlob{
		ContentType: "application/json",
		Data:        []byte(`{"max_connections_global":200,"max_upload_speed":-1}`),
	}, nil
}

func (d *Deluge) ImportConfig(_ context.Context, _ providers.Instance, _ providers.ConfigBlob) error {
	return nil
}

func seedFromID(id string) int64 {
	var h int64 = 17
	for _, c := range id {
		h = h*31 + int64(c)
	}
	return h
}
