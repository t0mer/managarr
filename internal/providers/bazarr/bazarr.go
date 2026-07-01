// internal/providers/bazarr/bazarr.go
package bazarr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

var cli = &http.Client{Timeout: 15 * time.Second}

func init() { providers.Register(&Bazarr{}) }

type Bazarr struct{}

func (b *Bazarr) Kind() providers.Kind { return providers.KindBazarr }

func (b *Bazarr) getJSON(ctx context.Context, inst providers.Instance, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL(inst)+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-KEY", inst.APIKey)
	req.Header.Set("Accept", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("GET %s: HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (b *Bazarr) TestConnection(ctx context.Context, inst providers.Instance) error {
	var status map[string]any
	return b.getJSON(ctx, inst, "/system/status", &status)
}

// FetchLogs returns log entries from Bazarr since the given time.
func (b *Bazarr) FetchLogs(ctx context.Context, inst providers.Instance, since time.Time) ([]providers.LogEntry, error) {
	var raw []struct {
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
		Message   string `json:"message"`
		Exception string `json:"exception"`
	}
	if err := b.getJSON(ctx, inst, "/system/logs", &raw); err != nil {
		return nil, fmt.Errorf("bazarr logs: %w", err)
	}

	var out []providers.LogEntry
	for _, r := range raw {
		ts, err := time.Parse("2006-01-02 15:04:05", r.Timestamp)
		if err != nil {
			ts = time.Time{}
		}
		if !ts.IsZero() && ts.Before(since) {
			continue
		}
		msg := r.Message
		if r.Exception != "" {
			msg += "\n" + r.Exception
		}
		out = append(out, providers.LogEntry{
			Timestamp: ts,
			Level:     normaliseLevel(r.Type),
			Source:    "bazarr",
			Message:   msg,
		})
	}
	return out, nil
}

func (b *Bazarr) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	return nil, fmt.Errorf("log streaming not supported for bazarr")
}

// Collect gathers subtitle wanted counts and download history stats.
func (b *Bazarr) Collect(ctx context.Context, inst providers.Instance) ([]providers.Sample, error) {
	now := time.Now()

	// /api/badges returns wanted episode/movie subtitle counts.
	var badges struct {
		Episodes  int `json:"episodes"`
		Movies    int `json:"movies"`
		Providers int `json:"providers"`
	}
	_ = b.getJSON(ctx, inst, "/badges", &badges)

	// /api/history/stats returns lifetime download counts.
	var stats struct {
		NumDownloaded int `json:"num_downloaded"`
		NumFailed     int `json:"num_failed"`
	}
	_ = b.getJSON(ctx, inst, "/history/stats", &stats)

	return []providers.Sample{
		{Metric: "bazarr_wanted_episodes", Value: float64(badges.Episodes), TS: now},
		{Metric: "bazarr_wanted_movies", Value: float64(badges.Movies), TS: now},
		{Metric: "bazarr_providers_total", Value: float64(badges.Providers), TS: now},
		{Metric: "bazarr_subtitles_downloaded", Value: float64(stats.NumDownloaded), TS: now},
		{Metric: "bazarr_subtitles_failed", Value: float64(stats.NumFailed), TS: now},
	}, nil
}

// ExportConfig triggers a Bazarr backup and downloads the resulting ZIP.
func (b *Bazarr) ExportConfig(ctx context.Context, inst providers.Instance) (providers.ConfigBlob, error) {
	// Trigger a new backup.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL(inst)+"/system/backups", nil)
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("building backup request: %w", err)
	}
	req.Header.Set("X-API-KEY", inst.APIKey)
	resp, err := cli.Do(req)
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("triggering bazarr backup: %w", err)
	}
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return providers.ConfigBlob{}, fmt.Errorf("bazarr backup trigger: HTTP %d", resp.StatusCode)
	}

	// List backups and pick the most recent.
	var backups []struct {
		Name string `json:"filename"`
		Size int    `json:"size"`
	}
	if err := b.getJSON(ctx, inst, "/system/backups", &backups); err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("listing bazarr backups: %w", err)
	}
	if len(backups) == 0 {
		return providers.ConfigBlob{}, fmt.Errorf("no backups found after triggering backup on %s", inst.Name)
	}
	latest := backups[0]

	// Download the backup via PATCH /api/system/backups?filename=<name>
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		apiURL(inst)+"/system/backups?filename="+latest.Name, nil)
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("building download request: %w", err)
	}
	dlReq.Header.Set("X-API-KEY", inst.APIKey)
	dlResp, err := cli.Do(dlReq)
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("downloading bazarr backup: %w", err)
	}
	defer dlResp.Body.Close()
	if dlResp.StatusCode >= 400 {
		return providers.ConfigBlob{}, fmt.Errorf("bazarr backup download: HTTP %d", dlResp.StatusCode)
	}
	data, err := io.ReadAll(dlResp.Body)
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("reading bazarr backup: %w", err)
	}
	if strings.Contains(dlResp.Header.Get("Content-Type"), "text/html") || len(data) == 0 {
		return providers.ConfigBlob{}, fmt.Errorf("bazarr backup download returned no data")
	}

	return providers.ConfigBlob{
		ContentType: "application/zip",
		Filename:    latest.Name,
		Data:        data,
	}, nil
}

func (b *Bazarr) ImportConfig(_ context.Context, _ providers.Instance, _ providers.ConfigBlob) error {
	return nil
}

// apiURL returns the Bazarr API base: <BaseURL>/api (trailing slash stripped).
func apiURL(inst providers.Instance) string {
	return strings.TrimRight(inst.BaseURL, "/") + "/api"
}

func normaliseLevel(t string) string {
	switch strings.ToUpper(t) {
	case "ERROR", "CRITICAL":
		return "error"
	case "WARNING", "WARN":
		return "warn"
	case "DEBUG":
		return "debug"
	default:
		return "info"
	}
}
