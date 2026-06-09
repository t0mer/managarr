// internal/providers/plex/plex.go
package plex

import (
	"context"
	"math/rand"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

func init() { providers.Register(&Plex{}) }

type Plex struct{}

func (p *Plex) Kind() providers.Kind { return providers.KindPlex }

func (p *Plex) TestConnection(_ context.Context, _ providers.Instance) error { return nil }

func (p *Plex) FetchLogs(_ context.Context, inst providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	rng := rand.New(rand.NewSource(seedFromID(inst.ID)))
	levels := []string{"info", "warn", "error", "debug"}
	messages := []string{
		"Library scan complete", "Transcoder started", "Playback started",
		"Playback stopped", "Media analysis queued", "Metadata agent updated",
	}
	entries := make([]providers.LogEntry, 20)
	base := time.Now()
	for i := range entries {
		entries[i] = providers.LogEntry{
			Timestamp: base.Add(-time.Duration(rng.Intn(3600)) * time.Second),
			Level:     levels[rng.Intn(len(levels))],
			Source:    "Plex",
			Message:   messages[rng.Intn(len(messages))],
		}
	}
	return entries, nil
}

func (p *Plex) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	ch := make(chan providers.LogEntry)
	close(ch)
	return ch, nil
}

func (p *Plex) Collect(_ context.Context, _ providers.Instance) ([]providers.Sample, error) {
	now := time.Now()
	return []providers.Sample{
		{Metric: "plex_movies_total", Value: 1240, TS: now},
		{Metric: "plex_shows_total", Value: 87, TS: now},
		{Metric: "plex_music_artists_total", Value: 312, TS: now},
		{Metric: "plex_sessions_active", Value: 2, TS: now},
		{Metric: "plex_transcodes_active", Value: 1, TS: now},
	}, nil
}

func seedFromID(id string) int64 {
	var h int64 = 17
	for _, c := range id {
		h = h*31 + int64(c)
	}
	return h
}
