// internal/providers/emby/emby.go
package emby

import (
	"context"
	"math/rand"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

func init() { providers.Register(&Emby{}) }

type Emby struct{}

func (e *Emby) Kind() providers.Kind { return providers.KindEmby }

func (e *Emby) TestConnection(_ context.Context, _ providers.Instance) error { return nil }

func (e *Emby) FetchLogs(_ context.Context, inst providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	rng := rand.New(rand.NewSource(seedFromID(inst.ID)))
	levels := []string{"info", "warn", "error"}
	messages := []string{
		"Library refresh complete", "Media stream started", "Transcode session started",
		"Plugin updated", "Scheduled task executed", "User activity logged",
	}
	entries := make([]providers.LogEntry, 20)
	base := time.Now()
	for i := range entries {
		entries[i] = providers.LogEntry{
			Timestamp: base.Add(-time.Duration(rng.Intn(3600)) * time.Second),
			Level:     levels[rng.Intn(len(levels))],
			Source:    "Emby",
			Message:   messages[rng.Intn(len(messages))],
		}
	}
	return entries, nil
}

func (e *Emby) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	ch := make(chan providers.LogEntry)
	close(ch)
	return ch, nil
}

func (e *Emby) Collect(_ context.Context, _ providers.Instance) ([]providers.Sample, error) {
	now := time.Now()
	return []providers.Sample{
		{Metric: "emby_movies_total", Value: 983, TS: now},
		{Metric: "emby_shows_total", Value: 64, TS: now},
		{Metric: "emby_sessions_active", Value: 1, TS: now},
		{Metric: "emby_transcodes_active", Value: 0, TS: now},
	}, nil
}

func seedFromID(id string) int64 {
	var h int64 = 17
	for _, c := range id {
		h = h*31 + int64(c)
	}
	return h
}
