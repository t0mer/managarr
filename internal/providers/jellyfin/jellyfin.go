// internal/providers/jellyfin/jellyfin.go
package jellyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

var cli = &http.Client{Timeout: 15 * time.Second}

func init() { providers.Register(&Jellyfin{}) }

type Jellyfin struct{}

func (j *Jellyfin) Kind() providers.Kind { return providers.KindJellyfin }

func (j *Jellyfin) getJSON(ctx context.Context, inst providers.Instance, path string, v any) error {
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
		return fmt.Errorf("jellyfin API: HTTP %d for %s", resp.StatusCode, path)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (j *Jellyfin) TestConnection(ctx context.Context, inst providers.Instance) error {
	var info map[string]any
	return j.getJSON(ctx, inst, "/System/Info", &info)
}

// FetchLogs returns an empty slice; Jellyfin log retrieval requires a separate log file endpoint
// that is not universally available. Stub preserved for future implementation.
func (j *Jellyfin) FetchLogs(_ context.Context, _ providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	return nil, nil
}

// StreamLogs is not supported by Jellyfin.
func (j *Jellyfin) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	return nil, fmt.Errorf("log streaming not supported for jellyfin")
}

// Collect gathers active session count and library folder count from the Jellyfin API.
func (j *Jellyfin) Collect(ctx context.Context, inst providers.Instance) ([]providers.Sample, error) {
	now := time.Now()

	var sessions []any
	_ = j.getJSON(ctx, inst, "/Sessions", &sessions)

	var libraries struct {
		Items []any `json:"Items"`
	}
	_ = j.getJSON(ctx, inst, "/Library/MediaFolders", &libraries)

	return []providers.Sample{
		{Metric: "jellyfin_active_sessions", Value: float64(len(sessions)), TS: now},
		{Metric: "jellyfin_library_folders", Value: float64(len(libraries.Items)), TS: now},
	}, nil
}
