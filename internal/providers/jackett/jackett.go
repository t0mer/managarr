// internal/providers/jackett/jackett.go
package jackett

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

var cli = &http.Client{Timeout: 15 * time.Second}

func init() { providers.Register(&Jackett{}) }

type Jackett struct{}

func (j *Jackett) Kind() providers.Kind { return providers.KindJackett }

func (j *Jackett) TestConnection(ctx context.Context, inst providers.Instance) error {
	_, err := getJSON(ctx, inst, "/api/v2.0/indexers")
	return err
}

func (j *Jackett) FetchLogs(_ context.Context, _ providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	return nil, nil
}

func (j *Jackett) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	return nil, fmt.Errorf("streaming not supported for jackett")
}

type indexer struct {
	ID         string `json:"ID"`
	Configured bool   `json:"Configured"`
}

func (j *Jackett) Collect(ctx context.Context, inst providers.Instance) ([]providers.Sample, error) {
	now := time.Now()

	data, err := getJSON(ctx, inst, "/api/v2.0/indexers")
	if err != nil {
		return nil, fmt.Errorf("jackett collect: %w", err)
	}
	var all []indexer
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("jackett collect unmarshal: %w", err)
	}

	configured := 0
	for _, idx := range all {
		if idx.Configured {
			configured++
		}
	}

	return []providers.Sample{
		{Metric: "jackett_indexers_total", Value: float64(len(all)), TS: now},
		{Metric: "jackett_indexers_configured", Value: float64(configured), TS: now},
	}, nil
}

func (j *Jackett) ExportConfig(ctx context.Context, inst providers.Instance) (providers.ConfigBlob, error) {
	data, err := getJSON(ctx, inst, "/api/v2.0/indexers")
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("jackett export: %w", err)
	}
	return providers.ConfigBlob{ContentType: "application/json", Data: data}, nil
}

func (j *Jackett) ImportConfig(_ context.Context, _ providers.Instance, _ providers.ConfigBlob) error {
	return nil
}

func (j *Jackett) Snapshot(ctx context.Context, inst providers.Instance) (providers.SyncState, error) {
	data, err := getJSON(ctx, inst, "/api/v2.0/indexers")
	if err != nil {
		return providers.SyncState{}, fmt.Errorf("jackett snapshot: %w", err)
	}
	var indexers []indexer
	if err := json.Unmarshal(data, &indexers); err != nil {
		return providers.SyncState{}, fmt.Errorf("jackett snapshot unmarshal: %w", err)
	}

	configured := 0
	ids := make([]string, 0, len(indexers))
	for _, idx := range indexers {
		if idx.Configured {
			configured++
			ids = append(ids, idx.ID)
		}
	}

	return providers.SyncState{Data: map[string]any{
		"indexer_count": configured,
		"indexer_ids":   ids,
	}}, nil
}

func (j *Jackett) Diff(a, b providers.SyncState) []providers.SyncChange {
	var changes []providers.SyncChange
	aCount, _ := a.Data["indexer_count"]
	bCount, _ := b.Data["indexer_count"]
	if aCount != bCount {
		changes = append(changes, providers.SyncChange{
			Field:    "indexer_count",
			OldValue: aCount,
			NewValue: bCount,
		})
	}
	return changes
}

func (j *Jackett) Apply(_ context.Context, _ providers.Instance, _ []providers.SyncChange) error {
	return nil
}

// getJSON calls the Jackett REST API. Authentication uses the ?apikey= query
// parameter — Jackett does not support X-Api-Key headers.
func getJSON(ctx context.Context, inst providers.Instance, path string) ([]byte, error) {
	base := strings.TrimRight(inst.BaseURL, "/")
	u, err := url.Parse(base + path)
	if err != nil {
		return nil, fmt.Errorf("building URL: %w", err)
	}
	if inst.APIKey != "" {
		q := u.Query()
		q.Set("apikey", inst.APIKey)
		u.RawQuery = q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("jackett API: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
