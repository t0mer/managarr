// internal/providers/jellyfin/jellyfin.go
package jellyfin

import (
	"context"
	"math/rand"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

func init() { providers.Register(&Jellyfin{}) }

type Jellyfin struct{}

func (j *Jellyfin) Kind() providers.Kind { return providers.KindJellyfin }

func (j *Jellyfin) TestConnection(_ context.Context, _ providers.Instance) error { return nil }

func (j *Jellyfin) FetchLogs(_ context.Context, inst providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	rng := rand.New(rand.NewSource(seedFromID(inst.ID)))
	levels := []string{"info", "warn", "error"}
	messages := []string{
		"Library scan completed", "Playback session started", "Metadata updated",
		"Plugin installed", "Database maintenance run", "User session ended",
	}
	entries := make([]providers.LogEntry, 20)
	base := time.Now()
	for i := range entries {
		entries[i] = providers.LogEntry{
			Timestamp: base.Add(-time.Duration(rng.Intn(3600)) * time.Second),
			Level:     levels[rng.Intn(len(levels))],
			Source:    "Jellyfin",
			Message:   messages[rng.Intn(len(messages))],
		}
	}
	return entries, nil
}

func (j *Jellyfin) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	ch := make(chan providers.LogEntry)
	close(ch)
	return ch, nil
}

func (j *Jellyfin) Collect(_ context.Context, _ providers.Instance) ([]providers.Sample, error) {
	now := time.Now()
	return []providers.Sample{
		{Metric: "jellyfin_movies_total", Value: 1102, TS: now},
		{Metric: "jellyfin_shows_total", Value: 71, TS: now},
		{Metric: "jellyfin_sessions_active", Value: 3, TS: now},
		{Metric: "jellyfin_transcodes_active", Value: 2, TS: now},
	}, nil
}

func seedFromID(id string) int64 {
	var h int64 = 17
	for _, c := range id {
		h = h*31 + int64(c)
	}
	return h
}
