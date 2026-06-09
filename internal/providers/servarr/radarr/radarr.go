// internal/providers/servarr/radarr/radarr.go
package radarr

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/t0mer/galactica/internal/providers"
	"github.com/t0mer/galactica/internal/providers/servarr"
)

func init() { providers.Register(&radarrProvider{}) }

type radarrProvider struct{}

func (p *radarrProvider) Kind() providers.Kind { return providers.KindRadarr }

// TestConnection calls /api/v3/system/status.
func (p *radarrProvider) TestConnection(ctx context.Context, inst providers.Instance) error {
	var status map[string]any
	return servarr.GetJSON(ctx, inst, "/api/v3/system/status", &status)
}

// FetchLogs returns recent log entries from Radarr filtered by since.
func (p *radarrProvider) FetchLogs(ctx context.Context, inst providers.Instance, since time.Time) ([]providers.LogEntry, error) {
	var resp struct {
		Records []struct {
			Time    string `json:"time"`
			Level   string `json:"level"`
			Logger  string `json:"logger"`
			Message string `json:"message"`
		} `json:"records"`
	}
	if err := servarr.GetJSON(ctx, inst, "/api/v3/log?pageSize=200&sortKey=time&sortDirection=descending", &resp); err != nil {
		return nil, fmt.Errorf("radarr fetch logs: %w", err)
	}
	var out []providers.LogEntry
	for _, rec := range resp.Records {
		t, _ := time.Parse(time.RFC3339, rec.Time)
		if t.Before(since) {
			continue
		}
		out = append(out, providers.LogEntry{
			Timestamp: t,
			Level:     rec.Level,
			Source:    rec.Logger,
			Message:   rec.Message,
		})
	}
	return out, nil
}

// StreamLogs is not supported for Radarr (no realtime log endpoint).
func (p *radarrProvider) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	return nil, fmt.Errorf("streaming not supported for radarr")
}

// Collect returns key metrics from Radarr.
func (p *radarrProvider) Collect(ctx context.Context, inst providers.Instance) ([]providers.Sample, error) {
	now := time.Now()

	var movies []struct {
		HasFile   bool `json:"hasFile"`
		Monitored bool `json:"monitored"`
	}
	_ = servarr.GetJSON(ctx, inst, "/api/v3/movie", &movies)

	var queue struct {
		TotalRecords int `json:"totalRecords"`
	}
	_ = servarr.GetJSON(ctx, inst, "/api/v3/queue?pageSize=1", &queue)

	missing := 0
	for _, m := range movies {
		if m.Monitored && !m.HasFile {
			missing++
		}
	}

	return []providers.Sample{
		{Metric: "radarr_movies_total", Value: float64(len(movies)), TS: now},
		{Metric: "radarr_queue_total", Value: float64(queue.TotalRecords), TS: now},
		{Metric: "radarr_missing_movies", Value: float64(missing), TS: now},
	}, nil
}

// ExportConfig exports quality profiles and naming config.
func (p *radarrProvider) ExportConfig(ctx context.Context, inst providers.Instance) (providers.ConfigBlob, error) {
	var profiles, naming any
	if err := servarr.GetJSON(ctx, inst, "/api/v3/qualityProfile", &profiles); err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("radarr export qualityProfile: %w", err)
	}
	if err := servarr.GetJSON(ctx, inst, "/api/v3/config/naming", &naming); err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("radarr export naming: %w", err)
	}
	b, err := json.Marshal(map[string]any{"qualityProfiles": profiles, "naming": naming})
	if err != nil {
		return providers.ConfigBlob{}, err
	}
	return providers.ConfigBlob{ContentType: "application/json", Data: b}, nil
}

// ImportConfig restores config from a blob. Full apply deferred to v2.
func (p *radarrProvider) ImportConfig(_ context.Context, _ providers.Instance, blob providers.ConfigBlob) error {
	var payload map[string]any
	if err := json.Unmarshal(blob.Data, &payload); err != nil {
		return fmt.Errorf("radarr import: invalid blob: %w", err)
	}
	return nil
}

// Snapshot captures quality profiles and naming for sync comparison.
func (p *radarrProvider) Snapshot(ctx context.Context, inst providers.Instance) (providers.SyncState, error) {
	var profiles, naming any
	if err := servarr.GetJSON(ctx, inst, "/api/v3/qualityProfile", &profiles); err != nil {
		return providers.SyncState{}, fmt.Errorf("radarr snapshot qualityProfile: %w", err)
	}
	if err := servarr.GetJSON(ctx, inst, "/api/v3/config/naming", &naming); err != nil {
		return providers.SyncState{}, fmt.Errorf("radarr snapshot naming: %w", err)
	}
	return providers.SyncState{Data: map[string]any{
		"qualityProfiles": profiles,
		"naming":          naming,
	}}, nil
}

// Diff compares two snapshots and returns field-level changes.
func (p *radarrProvider) Diff(a, b providers.SyncState) []providers.SyncChange {
	var changes []providers.SyncChange
	for k, va := range a.Data {
		if vb, ok := b.Data[k]; ok {
			ja, _ := json.Marshal(va)
			jb, _ := json.Marshal(vb)
			if string(ja) != string(jb) {
				changes = append(changes, providers.SyncChange{Field: k, OldValue: va, NewValue: vb})
			}
		}
	}
	return changes
}

// Apply pushes sync changes to the target instance. Full apply deferred to v2.
func (p *radarrProvider) Apply(_ context.Context, _ providers.Instance, _ []providers.SyncChange) error {
	return nil
}
