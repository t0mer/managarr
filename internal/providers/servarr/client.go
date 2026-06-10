// internal/providers/servarr/client.go
package servarr

import (
	"bytes"
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

// backupCli uses a longer timeout for downloading (potentially large) backup archives.
var backupCli = &http.Client{Timeout: 2 * time.Minute}

// GetJSON performs an authenticated GET to the servarr API and decodes the JSON response.
func GetJSON(ctx context.Context, inst providers.Instance, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, inst.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("X-Api-Key", inst.APIKey)
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

// backupEntry is a single record from the *arr system/backup list endpoint.
type backupEntry struct {
	Name string    `json:"name"`
	Path string    `json:"path"`
	Time time.Time `json:"time"`
	Size int64     `json:"size"`
}

// DownloadLatestBackup lists the instance's existing backup archives via
// {apiBase}/system/backup, picks the most recent by time, downloads it, and
// returns it as a ConfigBlob. apiBase is "/api/v3" for Sonarr/Radarr, "/api/v1"
// for Lidarr.
func DownloadLatestBackup(ctx context.Context, inst providers.Instance, apiBase string) (providers.ConfigBlob, error) {
	var backups []backupEntry
	if err := GetJSON(ctx, inst, apiBase+"/system/backup", &backups); err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("listing backups: %w", err)
	}
	if len(backups) == 0 {
		return providers.ConfigBlob{}, fmt.Errorf("no backups available — trigger one inside %s first", inst.Name)
	}

	latest := backups[0]
	for _, b := range backups[1:] {
		if b.Time.After(latest.Time) {
			latest = b
		}
	}

	// The /backup/ path is served by the *arr web layer, not the API layer.
	// It accepts Basic Auth (username:password) or an apiKey query parameter.
	// Prefer Basic Auth when credentials are provided; fall back to apiKey param.
	downloadURL := inst.BaseURL + latest.Path
	if inst.Username == "" {
		downloadURL += "?apiKey=" + inst.APIKey
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("building download request: %w", err)
	}
	if inst.Username != "" {
		req.SetBasicAuth(inst.Username, inst.Password)
	}

	resp, err := backupCli.Do(req)
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("downloading backup: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return providers.ConfigBlob{}, fmt.Errorf("download %s: HTTP %d", latest.Path, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("reading backup data: %w", err)
	}

	if len(data) == 0 {
		return providers.ConfigBlob{}, fmt.Errorf("download returned empty body")
	}

	// Reject obvious non-binary responses (HTML error/login pages from
	// unauthenticated or misconfigured requests).
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "text/html") {
		return providers.ConfigBlob{}, fmt.Errorf("download returned HTML (content-type: %s) — check API key and base URL", ct)
	}

	return providers.ConfigBlob{
		ContentType: "application/zip",
		Filename:    latest.Name,
		Data:        data,
	}, nil
}

// PutJSON performs an authenticated PUT with a JSON body.
func PutJSON(ctx context.Context, inst providers.Instance, path string, body, v any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshalling body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, inst.BaseURL+path, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("X-Api-Key", inst.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("PUT %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("PUT %s: HTTP %d", path, resp.StatusCode)
	}
	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}
	return nil
}
