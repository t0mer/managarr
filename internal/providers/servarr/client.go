// internal/providers/servarr/client.go
package servarr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

var cli = &http.Client{Timeout: 15 * time.Second}

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
