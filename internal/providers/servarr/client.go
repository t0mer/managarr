// internal/providers/servarr/client.go
package servarr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
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
	ID   int       `json:"id"`
	Name string    `json:"name"`
	Path string    `json:"path"`
	Time time.Time `json:"time"`
	Size int64     `json:"size"`
}

// formsLogin performs a Forms Auth login and returns an authenticated http.Client
// with the session cookie stored. The client uses an unrestricted cookie jar so
// the session cookie is sent to all paths regardless of the cookie's Path attribute.
func formsLogin(ctx context.Context, inst providers.Instance) (*http.Client, error) {
	if inst.Username == "" {
		return nil, fmt.Errorf("no credentials for forms auth")
	}
	jar := &allCookieJar{jar: newSimpleCookieJar()}
	client := &http.Client{
		Timeout: 2 * time.Minute,
		Jar:     jar,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	loginURL := inst.BaseURL + "/login"
	formData := url.Values{
		"username":   {inst.Username},
		"password":   {inst.Password},
		"returnUrl":  {"/"},
		"rememberMe": {"true"},
	}
	loginReq, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("building login request: %w", err)
	}
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResp, err := client.Do(loginReq)
	if err != nil {
		return nil, fmt.Errorf("login POST: %w", err)
	}
	io.Copy(io.Discard, loginResp.Body) //nolint:errcheck
	loginResp.Body.Close()

	location := loginResp.Header.Get("Location")
	slog.Debug("forms auth login response",
		"instance", inst.Name,
		"status", loginResp.StatusCode,
		"location", location,
		"cookies", len(jar.Cookies(loginResp.Request.URL)),
	)
	if strings.Contains(location, "loginFailed=true") {
		return nil, fmt.Errorf("login failed — check username/password for %s (redirected to %s)", inst.Name, location)
	}
	if loginResp.StatusCode >= 400 {
		return nil, fmt.Errorf("login returned HTTP %d", loginResp.StatusCode)
	}
	// Allow redirects for subsequent requests.
	client.CheckRedirect = nil
	return client, nil
}

// formsAuthDownload logs in via the *arr web forms endpoint and downloads
// the resource at the given URL path using the resulting session cookie.
// It tries the given path first, then the flat /backup/<filename> variant
// (Radarr's BackupFileMapper may only serve the flat path, not type-prefixed ones).
func formsAuthDownload(ctx context.Context, inst providers.Instance, path string) ([]byte, error) {
	client, err := formsLogin(ctx, inst)
	if err != nil {
		return nil, err
	}

	// Candidates: supplied path first, then the flat /backup/<filename> variant.
	// Radarr v6 BackupFileMapper may use Path.GetFileName internally, meaning it
	// only finds files when the URL is /backup/<name>.zip (no type subdirectory).
	candidates := []string{path}
	if base := filepath.Base(path); base != path && base != "." && strings.HasPrefix(path, "/backup/") {
		candidates = append(candidates, "/backup/"+base)
	}

	var lastErr error
	for _, p := range candidates {
		dlURL := inst.BaseURL + p
		dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
		if err != nil {
			lastErr = fmt.Errorf("building download request for %s: %w", p, err)
			continue
		}
		dlResp, err := client.Do(dlReq)
		if err != nil {
			lastErr = fmt.Errorf("download GET %s: %w", p, err)
			continue
		}
		body, readErr := io.ReadAll(dlResp.Body)
		dlResp.Body.Close()
		slog.Debug("forms auth download response",
			"instance", inst.Name,
			"url", dlURL,
			"status", dlResp.StatusCode,
			"content_type", dlResp.Header.Get("Content-Type"),
			"size", len(body),
			"body_preview", string(body[:min(len(body), 300)]),
		)
		if readErr != nil {
			lastErr = fmt.Errorf("reading response body from %s: %w", p, readErr)
			continue
		}
		if dlResp.StatusCode >= 400 {
			lastErr = fmt.Errorf("download %s returned HTTP %d: %s", p, dlResp.StatusCode, string(body[:min(len(body), 200)]))
			continue
		}
		if strings.Contains(dlResp.Header.Get("Content-Type"), "text/html") {
			lastErr = fmt.Errorf("got HTML from %s (auth may have been rejected)", p)
			continue
		}
		if len(body) == 0 {
			lastErr = fmt.Errorf("empty response from %s", p)
			continue
		}
		return body, nil
	}
	return nil, lastErr
}

// tryBackupDownload attempts to download a backup file using four auth strategies
// in order: Forms Auth (POST /login + cookie) → Basic Auth header → no-auth →
// apiKey query param. Returns the binary body of the first successful strategy.
func tryBackupDownload(ctx context.Context, inst providers.Instance, path string) ([]byte, error) {
	type strategyFn func() ([]byte, error)

	strategies := []struct {
		label string
		run   strategyFn
	}{
		{
			label: "Forms Auth",
			run: func() ([]byte, error) {
				return formsAuthDownload(ctx, inst, path)
			},
		},
		{
			label: "Basic Auth",
			run: func() ([]byte, error) {
				if inst.Username == "" {
					return nil, fmt.Errorf("no credentials")
				}
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, inst.BaseURL+path, nil)
				if err != nil {
					return nil, err
				}
				req.SetBasicAuth(inst.Username, inst.Password)
				resp, err := backupCli.Do(req)
				if err != nil {
					return nil, err
				}
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					return nil, fmt.Errorf("read body: %w", err)
				}
				if resp.StatusCode >= 400 {
					return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
				}
				if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
					return nil, fmt.Errorf("got HTML (status %d) — auth rejected", resp.StatusCode)
				}
				if len(body) == 0 {
					return nil, fmt.Errorf("empty response")
				}
				return body, nil
			},
		},
		{
			label: "no-auth",
			run: func() ([]byte, error) {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, inst.BaseURL+path, nil)
				if err != nil {
					return nil, err
				}
				resp, err := backupCli.Do(req)
				if err != nil {
					return nil, err
				}
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					return nil, fmt.Errorf("read body: %w", err)
				}
				if resp.StatusCode >= 400 {
					return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
				}
				if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
					return nil, fmt.Errorf("got HTML (status %d)", resp.StatusCode)
				}
				if len(body) == 0 {
					return nil, fmt.Errorf("empty response")
				}
				return body, nil
			},
		},
		{
			// apiKey in URL is a last resort: the value appears in server access
			// logs but is the only option for Forms Auth without valid credentials.
			label: "apiKey query param",
			run: func() ([]byte, error) {
				u := inst.BaseURL + path + "?apiKey=" + url.QueryEscape(inst.APIKey)
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
				if err != nil {
					return nil, err
				}
				resp, err := backupCli.Do(req)
				if err != nil {
					return nil, err
				}
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					return nil, fmt.Errorf("read body: %w", err)
				}
				if resp.StatusCode >= 400 {
					return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
				}
				if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
					return nil, fmt.Errorf("got HTML (status %d) — auth rejected", resp.StatusCode)
				}
				if len(body) == 0 {
					return nil, fmt.Errorf("empty response")
				}
				return body, nil
			},
		},
	}

	var errs []string
	for _, s := range strategies {
		data, err := s.run()
		if err != nil {
			errs = append(errs, fmt.Sprintf("[%s: %v]", s.label, err))
			continue
		}
		return data, nil
	}
	return nil, fmt.Errorf("all download strategies failed for %q — %s", path, strings.Join(errs, ", "))
}

// simpleCookieJar is an unrestricted cookie store: it sends ALL stored cookies
// for a given host regardless of path/secure constraints. This is necessary
// because ASP.NET Core (Radarr/Sonarr) may set session cookies with a path
// attribute that Go's standard cookiejar would refuse to send to /backup/...
type simpleCookieJar struct {
	cookies []*http.Cookie
}

func newSimpleCookieJar() *simpleCookieJar { return &simpleCookieJar{} }

func (j *simpleCookieJar) SetCookies(_ *url.URL, cookies []*http.Cookie) {
	j.cookies = append(j.cookies, cookies...)
}

func (j *simpleCookieJar) Cookies(_ *url.URL) []*http.Cookie { return j.cookies }

// allCookieJar wraps simpleCookieJar to satisfy http.CookieJar.
type allCookieJar struct{ jar *simpleCookieJar }

func (a *allCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	a.jar.SetCookies(u, cookies)
}
func (a *allCookieJar) Cookies(u *url.URL) []*http.Cookie { return a.jar.Cookies(u) }

// formsAuthAPIDownload combines Forms Auth login with API and web-layer download
// attempts. It logs in first, then tries:
//  1. API routes (session cookie sent alongside API key header)
//  2. Web-layer /backup/<path> and /backup/<name> (flat) with session cookie
func formsAuthAPIDownload(ctx context.Context, inst providers.Instance, apiBase string, backup backupEntry) ([]byte, error) {
	client, err := formsLogin(ctx, inst)
	if err != nil {
		return nil, err
	}

	// Build candidate list: API routes (cookie+key), then web routes (cookie only).
	// Radarr v6 API routes accept X-Api-Key but not session cookies; however
	// sending both costs nothing. Web routes accept only the session cookie.
	apiCandidates := []string{
		fmt.Sprintf("%s/backup/download/%d", apiBase, backup.ID),
		fmt.Sprintf("%s/backup/%d/download", apiBase, backup.ID),
	}
	// Web-layer candidates: full path from API listing, then flat /backup/<name>.
	webCandidates := []string{backup.Path}
	if base := filepath.Base(backup.Path); base != backup.Path && base != "." {
		webCandidates = append(webCandidates, "/backup/"+base)
	}

	var lastErr error
	for _, path := range apiCandidates {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, inst.BaseURL+path, nil)
		if err != nil {
			lastErr = fmt.Errorf("%s: build request: %w", path, err)
			continue
		}
		req.Header.Set("X-Api-Key", inst.APIKey)
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%s: %w", path, err)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("%s: read: %w", path, err)
			continue
		}
		slog.Debug("forms+api backup download",
			"instance", inst.Name,
			"path", path,
			"status", resp.StatusCode,
			"content_type", resp.Header.Get("Content-Type"),
			"size", len(body),
		)
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("%s: HTTP %d", path, resp.StatusCode)
			continue
		}
		if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			lastErr = fmt.Errorf("%s: got HTML", path)
			continue
		}
		if len(body) == 0 {
			lastErr = fmt.Errorf("%s: empty response", path)
			continue
		}
		return body, nil
	}

	for _, path := range webCandidates {
		dlURL := inst.BaseURL + path
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
		if err != nil {
			lastErr = fmt.Errorf("%s: build request: %w", path, err)
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%s: %w", path, err)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("%s: read: %w", path, err)
			continue
		}
		slog.Debug("forms+web backup download",
			"instance", inst.Name,
			"path", path,
			"status", resp.StatusCode,
			"content_type", resp.Header.Get("Content-Type"),
			"size", len(body),
			"body_preview", string(body[:min(len(body), 200)]),
		)
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("%s: HTTP %d: %s", path, resp.StatusCode, string(body[:min(len(body), 200)]))
			continue
		}
		if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			lastErr = fmt.Errorf("%s: got HTML (auth rejected)", path)
			continue
		}
		if len(body) == 0 {
			lastErr = fmt.Errorf("%s: empty response", path)
			continue
		}
		return body, nil
	}
	return nil, fmt.Errorf("forms+API download failed: %w", lastErr)
}

// downloadViaAPI tries to download a backup via the *arr API using X-Api-Key.
// Radarr v6+ removed direct /backup/ static file serving; backups are
// downloaded through the API using the backup entry's numeric ID.
func downloadViaAPI(ctx context.Context, inst providers.Instance, apiBase string, backup backupEntry) ([]byte, error) {
	// Try every known API route pattern for backup download in Radarr/Sonarr v6+.
	// Routes returning 401 without auth (route exists) are tested with X-Api-Key.
	candidates := []string{
		// ID-based routes
		fmt.Sprintf("%s/backup/download/%d", apiBase, backup.ID),
		fmt.Sprintf("%s/backup/%d/download", apiBase, backup.ID),
		fmt.Sprintf("%s/backup/%d", apiBase, backup.ID),
		fmt.Sprintf("%s/backup/restore/download/%d", apiBase, backup.ID),
		fmt.Sprintf("%s/system/backup/download/%d", apiBase, backup.ID),
		fmt.Sprintf("%s/system/backup/%d", apiBase, backup.ID),
		// Query-param variants
		fmt.Sprintf("%s/backup/download?id=%d", apiBase, backup.ID),
		fmt.Sprintf("%s/backup/download?name=%s", apiBase, url.PathEscape(backup.Name)),
		// Name-based routes
		fmt.Sprintf("%s/backup/file/%s", apiBase, url.PathEscape(backup.Name)),
		fmt.Sprintf("%s/backup/content/%s", apiBase, url.PathEscape(backup.Name)),
		fmt.Sprintf("/api/download/backup/%s", url.PathEscape(backup.Name)),
		// Backup type + name (path from API already includes /backup/manual|scheduled/name)
		backup.Path,
	}
	var lastErr error
	for _, path := range candidates {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, inst.BaseURL+path, nil)
		if err != nil {
			lastErr = fmt.Errorf("%s: build request: %w", path, err)
			continue
		}
		req.Header.Set("X-Api-Key", inst.APIKey)
		resp, err := backupCli.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%s: %w", path, err)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("%s: read: %w", path, err)
			continue
		}
		slog.Debug("api backup download",
			"instance", inst.Name,
			"path", path,
			"status", resp.StatusCode,
			"content_type", resp.Header.Get("Content-Type"),
			"size", len(body),
		)
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("%s: HTTP %d", path, resp.StatusCode)
			continue
		}
		if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			lastErr = fmt.Errorf("%s: got HTML", path)
			continue
		}
		if len(body) == 0 {
			lastErr = fmt.Errorf("%s: empty response", path)
			continue
		}
		return body, nil
	}
	return nil, fmt.Errorf("API download failed: %w", lastErr)
}

// triggerBackupCommand posts a Backup command to the *arr instance and waits
// (up to 2 minutes) for it to reach "completed" status.
func triggerBackupCommand(ctx context.Context, inst providers.Instance, apiBase string) error {
	type commandReq struct {
		Name string `json:"name"`
	}
	type commandResp struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
		Message string `json:"message"`
	}

	b, err := json.Marshal(commandReq{Name: "Backup"})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, inst.BaseURL+apiBase+"/command", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("building backup command request: %w", err)
	}
	req.Header.Set("X-Api-Key", inst.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("posting backup command: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("backup command returned HTTP %d", resp.StatusCode)
	}
	var cmd commandResp
	if err := json.NewDecoder(resp.Body).Decode(&cmd); err != nil {
		return fmt.Errorf("decoding backup command response: %w", err)
	}

	// Poll until completed or failed (max 2 minutes).
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		var poll commandResp
		if err := GetJSON(ctx, inst, fmt.Sprintf("%s/command/%d", apiBase, cmd.ID), &poll); err != nil {
			return fmt.Errorf("polling backup command %d: %w", cmd.ID, err)
		}
		switch poll.Status {
		case "completed":
			return nil
		case "failed":
			return fmt.Errorf("backup command failed: %s", poll.Message)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
	return fmt.Errorf("timed out waiting for backup command to complete")
}

// exportConfigViaAPI exports the app's configuration by fetching key API
// endpoints and packaging them into a single JSON document. This is the
// fallback when the backup ZIP cannot be downloaded (e.g. the user has only
// an API key, not Forms Auth credentials).
func exportConfigViaAPI(ctx context.Context, inst providers.Instance, apiBase string) (providers.ConfigBlob, error) {
	// Common endpoints across all *arr apps.
	endpoints := []string{
		apiBase + "/qualityprofile",
		apiBase + "/indexer",
		apiBase + "/downloadclient",
		apiBase + "/customformat",
		apiBase + "/tag",
		apiBase + "/rootfolder",
		apiBase + "/config/naming",
		apiBase + "/config/mediamanagement",
		apiBase + "/system/status",
		// Media-type specific; 404s are silently skipped.
		apiBase + "/movie",   // Radarr
		apiBase + "/series",  // Sonarr
		apiBase + "/artist",  // Lidarr
	}

	collected := make(map[string]json.RawMessage, len(endpoints))
	for _, ep := range endpoints {
		var raw json.RawMessage
		if err := GetJSON(ctx, inst, ep, &raw); err != nil {
			slog.Debug("api export: skipping endpoint", "instance", inst.Name, "endpoint", ep, "error", err)
			continue
		}
		key := strings.TrimPrefix(ep, apiBase+"/")
		collected[key] = raw
	}
	if len(collected) == 0 {
		return providers.ConfigBlob{}, fmt.Errorf("API config export: all endpoints failed for %s", inst.Name)
	}

	type envelope struct {
		App        string                      `json:"app"`
		ExportedAt string                      `json:"exported_at"`
		Note       string                      `json:"note"`
		Endpoints  map[string]json.RawMessage  `json:"endpoints"`
	}
	env := envelope{
		App:        inst.Name,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Note:       "API config export (backup ZIP download unavailable — add username/password to instance for full backup)",
		Endpoints:  collected,
	}
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("encoding API export: %w", err)
	}

	slug := strings.ToLower(strings.NewReplacer(" ", "_", "/", "_").Replace(inst.Name))
	filename := fmt.Sprintf("%s_config_%s.json", slug, time.Now().UTC().Format("2006-01-02T15-04-05Z"))

	return providers.ConfigBlob{
		ContentType: "application/json",
		Filename:    filename,
		Data:        data,
	}, nil
}

// DownloadLatestBackup triggers a fresh backup on the *arr instance via the
// command API, waits for it to complete, then downloads the resulting archive.
// If the backup ZIP cannot be downloaded (e.g. only an API key is configured,
// not Forms Auth credentials), it falls back to an API-based config export.
// apiBase is "/api/v3" for Sonarr/Radarr, "/api/v1" for Lidarr.
func DownloadLatestBackup(ctx context.Context, inst providers.Instance, apiBase string) (providers.ConfigBlob, error) {
	// Always trigger a fresh backup so we get a file that actually exists on disk.
	if err := triggerBackupCommand(ctx, inst, apiBase); err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("triggering backup: %w", err)
	}

	var backups []backupEntry
	if err := GetJSON(ctx, inst, apiBase+"/system/backup", &backups); err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("listing backups: %w", err)
	}
	if len(backups) == 0 {
		return providers.ConfigBlob{}, fmt.Errorf("no backups found after triggering backup on %s", inst.Name)
	}

	latest := backups[0]
	for _, b := range backups[1:] {
		if b.Time.After(latest.Time) {
			latest = b
		}
	}

	// Attempt to download the backup ZIP. Try in order:
	// 1. Forms Auth session cookie (requires username+password in instance config)
	// 2. Direct API download routes (Radarr v6+ has no such endpoint; kept for future)
	// 3. Web-layer with Basic Auth / no-auth / ?apiKey (Sonarr/older versions)
	var downloadErr error
	if data, err := formsAuthAPIDownload(ctx, inst, apiBase, latest); err == nil {
		return providers.ConfigBlob{
			ContentType: "application/zip",
			Filename:    latest.Name,
			Data:        data,
		}, nil
	} else {
		slog.Debug("forms+API backup download failed", "instance", inst.Name, "error", err)
		downloadErr = err
	}

	if data, err := downloadViaAPI(ctx, inst, apiBase, latest); err == nil {
		return providers.ConfigBlob{
			ContentType: "application/zip",
			Filename:    latest.Name,
			Data:        data,
		}, nil
	} else {
		slog.Debug("API backup download failed", "instance", inst.Name, "error", err)
		downloadErr = fmt.Errorf("%v; %v", downloadErr, err)
	}

	if data, err := tryBackupDownload(ctx, inst, latest.Path); err == nil {
		return providers.ConfigBlob{
			ContentType: "application/zip",
			Filename:    latest.Name,
			Data:        data,
		}, nil
	} else {
		slog.Debug("web backup download failed", "instance", inst.Name, "error", err)
		downloadErr = fmt.Errorf("%v; %v", downloadErr, err)
	}

	// All download strategies exhausted — fall back to an API config export.
	// This happens when Radarr/Sonarr requires Forms Auth but no username/password
	// is configured for the instance. The export captures configuration but not
	// the full database state.
	slog.Warn("backup ZIP download failed, falling back to API config export",
		"instance", inst.Name, "error", downloadErr)
	return exportConfigViaAPI(ctx, inst, apiBase)
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
