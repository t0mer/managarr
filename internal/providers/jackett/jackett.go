// internal/providers/jackett/jackett.go
package jackett

import (
	"context"
	"math/rand"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

func init() { providers.Register(&Jackett{}) }

type Jackett struct{}

func (j *Jackett) Kind() providers.Kind { return providers.KindJackett }

func (j *Jackett) TestConnection(_ context.Context, _ providers.Instance) error { return nil }

func (j *Jackett) FetchLogs(_ context.Context, inst providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	rng := rand.New(rand.NewSource(seedFromID(inst.ID)))
	levels := []string{"info", "warn", "error"}
	messages := []string{
		"Indexer search completed", "Indexer timeout", "Captcha required",
		"Category mapping updated", "Torznab feed parsed successfully",
	}
	entries := make([]providers.LogEntry, 20)
	base := time.Now()
	for i := range entries {
		entries[i] = providers.LogEntry{
			Timestamp: base.Add(-time.Duration(rng.Intn(3600)) * time.Second),
			Level:     levels[rng.Intn(len(levels))],
			Source:    "Jackett",
			Message:   messages[rng.Intn(len(messages))],
		}
	}
	return entries, nil
}

func (j *Jackett) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	ch := make(chan providers.LogEntry)
	close(ch)
	return ch, nil
}

func (j *Jackett) Collect(_ context.Context, _ providers.Instance) ([]providers.Sample, error) {
	now := time.Now()
	return []providers.Sample{
		{Metric: "jackett_indexers_total", Value: 42, TS: now},
		{Metric: "jackett_indexers_errors", Value: 3, TS: now},
	}, nil
}

func (j *Jackett) ExportConfig(_ context.Context, _ providers.Instance) (providers.ConfigBlob, error) {
	return providers.ConfigBlob{
		ContentType: "application/json",
		Data:        []byte(`{"indexers":[],"server":{}}`),
	}, nil
}

func (j *Jackett) ImportConfig(_ context.Context, _ providers.Instance, _ providers.ConfigBlob) error {
	return nil
}

func (j *Jackett) Snapshot(_ context.Context, _ providers.Instance) (providers.SyncState, error) {
	return providers.SyncState{Data: map[string]any{"indexerCount": 42}}, nil
}

func (j *Jackett) Diff(_, _ providers.SyncState) []providers.SyncChange {
	return []providers.SyncChange{
		{Field: "indexers[5].enabled", OldValue: false, NewValue: true},
	}
}

func (j *Jackett) Apply(_ context.Context, _ providers.Instance, _ []providers.SyncChange) error {
	return nil
}

func seedFromID(id string) int64 {
	var h int64 = 17
	for _, c := range id {
		h = h*31 + int64(c)
	}
	return h
}
