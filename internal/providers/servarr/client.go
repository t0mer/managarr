// internal/providers/servarr/client.go
// Package servarr contains the shared HTTP client for *arr v3 API apps.
// Phase 1: stub only. Phase 2 will implement real HTTP calls.
package servarr

// Client is a placeholder for the shared *arr v3 HTTP client.
// Sonarr, Radarr, and Lidarr embed this once Phase 2 implements real calls.
type Client struct {
	BaseURL string
	APIKey  string
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{BaseURL: baseURL, APIKey: apiKey}
}
