// internal/providers/jackett/jackett.go
package jackett

import (
	"context"
	"encoding/xml"
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

// TestConnection uses the Torznab caps endpoint — the only Jackett API path
// that accepts ?apikey= without requiring a browser session cookie.
func (j *Jackett) TestConnection(ctx context.Context, inst providers.Instance) error {
	_, err := torznabGet(ctx, inst, "caps")
	return err
}

func (j *Jackett) FetchLogs(_ context.Context, _ providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	return nil, nil
}

func (j *Jackett) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	return nil, fmt.Errorf("streaming not supported for jackett")
}

// xmlIndexer is a single entry from the ?t=indexers response.
type xmlIndexer struct {
	ID         string `xml:"id,attr"`
	Configured string `xml:"configured,attr"`
	Title      string `xml:"title"`
}

type xmlIndexers struct {
	Indexers []xmlIndexer `xml:"indexer"`
}

func (j *Jackett) Collect(ctx context.Context, inst providers.Instance) ([]providers.Sample, error) {
	now := time.Now()
	indexers, err := listIndexers(ctx, inst)
	if err != nil {
		return nil, fmt.Errorf("jackett collect: %w", err)
	}
	configured := 0
	for _, idx := range indexers {
		if idx.Configured == "true" {
			configured++
		}
	}
	return []providers.Sample{
		{Metric: "jackett_indexers_total", Value: float64(len(indexers)), TS: now},
		{Metric: "jackett_indexers_configured", Value: float64(configured), TS: now},
	}, nil
}

func (j *Jackett) ExportConfig(ctx context.Context, inst providers.Instance) (providers.ConfigBlob, error) {
	data, err := torznabGet(ctx, inst, "indexers")
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("jackett export: %w", err)
	}
	return providers.ConfigBlob{ContentType: "application/xml", Data: data}, nil
}

func (j *Jackett) ImportConfig(_ context.Context, _ providers.Instance, _ providers.ConfigBlob) error {
	return nil
}

func (j *Jackett) Snapshot(ctx context.Context, inst providers.Instance) (providers.SyncState, error) {
	indexers, err := listIndexers(ctx, inst)
	if err != nil {
		return providers.SyncState{}, fmt.Errorf("jackett snapshot: %w", err)
	}
	ids := make([]string, 0, len(indexers))
	configured := 0
	for _, idx := range indexers {
		if idx.Configured == "true" {
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

// listIndexers fetches all indexers via the Torznab ?t=indexers endpoint.
func listIndexers(ctx context.Context, inst providers.Instance) ([]xmlIndexer, error) {
	data, err := torznabGet(ctx, inst, "indexers")
	if err != nil {
		return nil, err
	}
	var result xmlIndexers
	if err := xml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing indexers XML: %w", err)
	}
	return result.Indexers, nil
}

// torznabGet calls /api/v2.0/indexers/all/results/torznab with ?t=<t> and the apikey.
// This is the only Jackett API surface that accepts API key auth without a session cookie.
func torznabGet(ctx context.Context, inst providers.Instance, t string) ([]byte, error) {
	base := strings.TrimRight(inst.BaseURL, "/")
	u, err := url.Parse(base + "/api/v2.0/indexers/all/results/torznab")
	if err != nil {
		return nil, fmt.Errorf("building URL: %w", err)
	}
	q := u.Query()
	q.Set("t", t)
	if inst.APIKey != "" {
		q.Set("apikey", inst.APIKey)
	}
	u.RawQuery = q.Encode()

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
