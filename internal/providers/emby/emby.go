// internal/providers/emby/emby.go
package emby

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

var cli = &http.Client{Timeout: 15 * time.Second}

func init() { providers.Register(&Emby{}) }

type Emby struct{}

func (e *Emby) Kind() providers.Kind { return providers.KindEmby }

func (e *Emby) getJSON(ctx context.Context, inst providers.Instance, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, inst.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Emby-Token", inst.APIKey)
	req.Header.Set("Accept", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("emby API: HTTP %d for %s", resp.StatusCode, path)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (e *Emby) TestConnection(ctx context.Context, inst providers.Instance) error {
	var info map[string]any
	return e.getJSON(ctx, inst, "/System/Info", &info)
}

// FetchLogs returns an empty slice; Emby log retrieval requires a separate log file endpoint
// that is not universally available. Stub preserved for future implementation.
func (e *Emby) FetchLogs(_ context.Context, _ providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	return nil, nil
}

// StreamLogs is not supported by Emby.
func (e *Emby) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	return nil, fmt.Errorf("log streaming not supported for emby")
}

// Collect gathers active session count and library folder count from the Emby API.
func (e *Emby) Collect(ctx context.Context, inst providers.Instance) ([]providers.Sample, error) {
	now := time.Now()

	var sessions []any
	_ = e.getJSON(ctx, inst, "/Sessions", &sessions)

	var libraries struct {
		Items []any `json:"Items"`
	}
	_ = e.getJSON(ctx, inst, "/Library/MediaFolders", &libraries)

	return []providers.Sample{
		{Metric: "emby_active_sessions", Value: float64(len(sessions)), TS: now},
		{Metric: "emby_library_folders", Value: float64(len(libraries.Items)), TS: now},
	}, nil
}
