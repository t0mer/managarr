// internal/providers/servarr/sonarr/sonarr.go
package sonarr

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/t0mer/galactica/internal/providers"
	"github.com/t0mer/galactica/internal/providers/servarr"
)

func init() { providers.Register(&sonarrProvider{}) }

type sonarrProvider struct{}

func (p *sonarrProvider) Kind() providers.Kind { return providers.KindSonarr }

// TestConnection calls /api/v3/system/status.
func (p *sonarrProvider) TestConnection(ctx context.Context, inst providers.Instance) error {
	var status map[string]any
	return servarr.GetJSON(ctx, inst, "/api/v3/system/status", &status)
}

// FetchLogs returns recent log entries from Sonarr filtered by since.
func (p *sonarrProvider) FetchLogs(ctx context.Context, inst providers.Instance, since time.Time) ([]providers.LogEntry, error) {
	var resp struct {
		Records []struct {
			Time    string `json:"time"`
			Level   string `json:"level"`
			Logger  string `json:"logger"`
			Message string `json:"message"`
		} `json:"records"`
	}
	if err := servarr.GetJSON(ctx, inst, "/api/v3/log?pageSize=200&sortKey=time&sortDirection=descending", &resp); err != nil {
		return nil, fmt.Errorf("sonarr fetch logs: %w", err)
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

// StreamLogs is not supported for Sonarr (no realtime log endpoint).
func (p *sonarrProvider) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	return nil, fmt.Errorf("streaming not supported for sonarr")
}

// Collect returns key metrics from Sonarr.
func (p *sonarrProvider) Collect(ctx context.Context, inst providers.Instance) ([]providers.Sample, error) {
	now := time.Now()

	var series []struct{ ID int }
	_ = servarr.GetJSON(ctx, inst, "/api/v3/series", &series)

	var queue struct {
		TotalRecords int `json:"totalRecords"`
	}
	_ = servarr.GetJSON(ctx, inst, "/api/v3/queue?pageSize=1", &queue)

	var missing struct {
		TotalRecords int `json:"totalRecords"`
	}
	_ = servarr.GetJSON(ctx, inst, "/api/v3/wanted/missing?pageSize=1", &missing)

	return []providers.Sample{
		{Metric: "sonarr_series_total", Value: float64(len(series)), TS: now},
		{Metric: "sonarr_queue_total", Value: float64(queue.TotalRecords), TS: now},
		{Metric: "sonarr_missing_episodes", Value: float64(missing.TotalRecords), TS: now},
	}, nil
}

// ExportConfig downloads the most recent backup archive Sonarr has on disk.
func (p *sonarrProvider) ExportConfig(ctx context.Context, inst providers.Instance) (providers.ConfigBlob, error) {
	blob, err := servarr.DownloadLatestBackup(ctx, inst, "/api/v3")
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("sonarr backup: %w", err)
	}
	return blob, nil
}

// ImportConfig restores config from a blob. Full apply deferred to v2.
func (p *sonarrProvider) ImportConfig(_ context.Context, _ providers.Instance, blob providers.ConfigBlob) error {
	var payload map[string]any
	if err := json.Unmarshal(blob.Data, &payload); err != nil {
		return fmt.Errorf("sonarr import: invalid blob: %w", err)
	}
	return nil
}

// Snapshot captures quality profiles and naming for sync comparison.
func (p *sonarrProvider) Snapshot(ctx context.Context, inst providers.Instance) (providers.SyncState, error) {
	var profiles, naming any
	if err := servarr.GetJSON(ctx, inst, "/api/v3/qualityProfile", &profiles); err != nil {
		return providers.SyncState{}, fmt.Errorf("sonarr snapshot qualityProfile: %w", err)
	}
	if err := servarr.GetJSON(ctx, inst, "/api/v3/config/naming", &naming); err != nil {
		return providers.SyncState{}, fmt.Errorf("sonarr snapshot naming: %w", err)
	}
	return providers.SyncState{Data: map[string]any{
		"qualityProfiles": profiles,
		"naming":          naming,
	}}, nil
}

// Diff compares two snapshots and returns field-level changes.
func (p *sonarrProvider) Diff(a, b providers.SyncState) []providers.SyncChange {
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
func (p *sonarrProvider) Apply(_ context.Context, _ providers.Instance, _ []providers.SyncChange) error {
	return nil
}
