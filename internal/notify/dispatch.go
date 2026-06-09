// internal/notify/dispatch.go
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/containrrr/shoutrrr"
)

// ChannelConfig holds provider-specific fields for a notification channel.
type ChannelConfig struct {
	// shoutrrr
	URL string `json:"url,omitempty"`
	// greenapi
	InstanceID string `json:"instance_id,omitempty"`
	Token      string `json:"token,omitempty"`
	Phone      string `json:"phone,omitempty"`
	APIURL     string `json:"api_url,omitempty"`
	// whatsapp_web
	BaseURL  string `json:"base_url,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

var httpCli = &http.Client{Timeout: 15 * time.Second}

// Send dispatches a message via the specified provider.
func Send(ctx context.Context, provider string, cfg ChannelConfig, msg string) error {
	switch provider {
	case "shoutrrr":
		return sendShoutrrr(cfg.URL, msg)
	case "greenapi":
		return sendGreenAPI(ctx, cfg, msg)
	case "whatsapp_web":
		return sendWhatsAppWeb(ctx, cfg, msg)
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}
}

func sendShoutrrr(url, msg string) error {
	return shoutrrr.Send(url, msg)
}

func sendGreenAPI(ctx context.Context, cfg ChannelConfig, msg string) error {
	apiURL := strings.TrimRight(cfg.APIURL, "/")
	if apiURL == "" {
		apiURL = "https://api.green-api.com"
	}
	phone := strings.TrimSpace(cfg.Phone)
	if !strings.Contains(phone, "@") {
		phone += "@c.us"
	}
	url := fmt.Sprintf("%s/waInstance%s/sendMessage/%s",
		apiURL,
		strings.TrimSpace(cfg.InstanceID),
		strings.TrimSpace(cfg.Token),
	)
	body, err := json.Marshal(map[string]string{"chatId": phone, "message": msg})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpCli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("greenapi: HTTP %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func sendWhatsAppWeb(ctx context.Context, cfg ChannelConfig, msg string) error {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	phone := strings.TrimSpace(cfg.Phone)
	if !strings.Contains(phone, "@") {
		phone += "@c.us"
	}
	body, err := json.Marshal(map[string]string{"chatId": phone, "message": msg})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/send-message", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Username != "" || cfg.Password != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}
	resp, err := httpCli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("whatsapp_web: HTTP %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
