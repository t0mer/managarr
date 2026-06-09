// internal/providers/servarr/lidarr/lidarr.go
package lidarr

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/t0mer/galactica/internal/providers"
	"github.com/t0mer/galactica/internal/providers/servarr"
)

func init() { providers.Register(&lidarrProvider{}) }

type lidarrProvider struct{}

func (p *lidarrProvider) Kind() providers.Kind { return providers.KindLidarr }

// TestConnection calls /api/v1/system/status.
func (p *lidarrProvider) TestConnection(ctx context.Context, inst providers.Instance) error {
	var status map[string]any
	return servarr.GetJSON(ctx, inst, "/api/v1/system/status", &status)
}

// FetchLogs returns recent log entries from Lidarr filtered by since.
func (p *lidarrProvider) FetchLogs(ctx context.Context, inst providers.Instance, since time.Time) ([]providers.LogEntry, error) {
	var resp struct {
		Records []struct {
			Time    string `json:"time"`
			Level   string `json:"level"`
			Logger  string `json:"logger"`
			Message string `json:"message"`
		} `json:"records"`
	}
	if err := servarr.GetJSON(ctx, inst, "/api/v1/log?pageSize=200&sortKey=time&sortDirection=descending", &resp); err != nil {
		return nil, fmt.Errorf("lidarr fetch logs: %w", err)
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

// StreamLogs is not supported for Lidarr (no realtime log endpoint).
func (p *lidarrProvider) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	return nil, fmt.Errorf("streaming not supported for lidarr")
}

// Collect returns key metrics from Lidarr.
func (p *lidarrProvider) Collect(ctx context.Context, inst providers.Instance) ([]providers.Sample, error) {
	now := time.Now()

	var artists []struct{ ID int }
	_ = servarr.GetJSON(ctx, inst, "/api/v1/artist", &artists)

	var queue struct {
		TotalRecords int `json:"totalRecords"`
	}
	_ = servarr.GetJSON(ctx, inst, "/api/v1/queue?pageSize=1", &queue)

	return []providers.Sample{
		{Metric: "lidarr_artists_total", Value: float64(len(artists)), TS: now},
		{Metric: "lidarr_queue_total", Value: float64(queue.TotalRecords), TS: now},
	}, nil
}

// ExportConfig exports quality profiles.
func (p *lidarrProvider) ExportConfig(ctx context.Context, inst providers.Instance) (providers.ConfigBlob, error) {
	var profiles any
	if err := servarr.GetJSON(ctx, inst, "/api/v1/qualityProfile", &profiles); err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("lidarr export qualityProfile: %w", err)
	}
	b, err := json.Marshal(map[string]any{"qualityProfiles": profiles})
	if err != nil {
		return providers.ConfigBlob{}, err
	}
	return providers.ConfigBlob{ContentType: "application/json", Data: b}, nil
}

// ImportConfig restores config from a blob. Full apply deferred to v2.
func (p *lidarrProvider) ImportConfig(_ context.Context, _ providers.Instance, blob providers.ConfigBlob) error {
	var payload map[string]any
	if err := json.Unmarshal(blob.Data, &payload); err != nil {
		return fmt.Errorf("lidarr import: invalid blob: %w", err)
	}
	return nil
}

// Snapshot captures quality profiles for sync comparison.
func (p *lidarrProvider) Snapshot(ctx context.Context, inst providers.Instance) (providers.SyncState, error) {
	var profiles any
	if err := servarr.GetJSON(ctx, inst, "/api/v1/qualityProfile", &profiles); err != nil {
		return providers.SyncState{}, fmt.Errorf("lidarr snapshot qualityProfile: %w", err)
	}
	return providers.SyncState{Data: map[string]any{
		"qualityProfiles": profiles,
	}}, nil
}

// Diff compares two snapshots and returns field-level changes.
func (p *lidarrProvider) Diff(a, b providers.SyncState) []providers.SyncChange {
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
func (p *lidarrProvider) Apply(_ context.Context, _ providers.Instance, _ []providers.SyncChange) error {
	return nil
}
