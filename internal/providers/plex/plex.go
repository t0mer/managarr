// internal/providers/plex/plex.go
package plex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

var cli = &http.Client{Timeout: 15 * time.Second}

func init() { providers.Register(&Plex{}) }

type Plex struct{}

func (p *Plex) Kind() providers.Kind { return providers.KindPlex }

func (p *Plex) getJSON(ctx context.Context, inst providers.Instance, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, inst.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Plex-Token", inst.APIKey)
	req.Header.Set("Accept", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("plex API: HTTP %d for %s", resp.StatusCode, path)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (p *Plex) TestConnection(ctx context.Context, inst providers.Instance) error {
	var status map[string]any
	return p.getJSON(ctx, inst, "/", &status)
}

// FetchLogs returns an empty slice; Plex does not expose a structured log API.
func (p *Plex) FetchLogs(_ context.Context, _ providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	return nil, nil
}

// StreamLogs is not supported by Plex.
func (p *Plex) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	return nil, fmt.Errorf("log streaming not supported for plex")
}

// Collect gathers active session count and library section count from the Plex API.
func (p *Plex) Collect(ctx context.Context, inst providers.Instance) ([]providers.Sample, error) {
	now := time.Now()

	var sessions struct {
		MediaContainer struct {
			Size int `json:"size"`
		} `json:"MediaContainer"`
	}
	_ = p.getJSON(ctx, inst, "/status/sessions", &sessions)

	var sections struct {
		MediaContainer struct {
			Size int `json:"size"`
		} `json:"MediaContainer"`
	}
	_ = p.getJSON(ctx, inst, "/library/sections", &sections)

	return []providers.Sample{
		{Metric: "plex_active_sessions", Value: float64(sessions.MediaContainer.Size), TS: now},
		{Metric: "plex_library_sections", Value: float64(sections.MediaContainer.Size), TS: now},
	}, nil
}
