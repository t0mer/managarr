// internal/providers/deluge/deluge.go
package deluge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/t0mer/galactica/internal/providers"
)

func init() { providers.Register(&Deluge{}) }

type Deluge struct{}

func (d *Deluge) Kind() providers.Kind { return providers.KindDeluge }

type rpcReq struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
	ID     int    `json:"id"`
}

type rpcResp struct {
	Result any `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
	ID int `json:"id"`
}

func newClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: 15 * time.Second, Jar: jar}
}

func rpc(ctx context.Context, cli *http.Client, baseURL, method string, params []any) (any, error) {
	body, err := json.Marshal(rpcReq{Method: method, Params: params, ID: 1})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var r rpcResp
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if r.Error != nil {
		return nil, fmt.Errorf("deluge RPC error: %s", r.Error.Message)
	}
	return r.Result, nil
}

func authenticate(ctx context.Context, cli *http.Client, inst providers.Instance) error {
	result, err := rpc(ctx, cli, inst.BaseURL, "auth.login", []any{inst.APIKey})
	if err != nil {
		return fmt.Errorf("deluge auth: %w", err)
	}
	if ok, _ := result.(bool); !ok {
		return fmt.Errorf("deluge auth failed")
	}
	return nil
}

func (d *Deluge) TestConnection(ctx context.Context, inst providers.Instance) error {
	cli := newClient()
	if err := authenticate(ctx, cli, inst); err != nil {
		return err
	}
	_, err := rpc(ctx, cli, inst.BaseURL, "core.get_session_status", []any{[]string{}})
	return err
}

func (d *Deluge) FetchLogs(_ context.Context, _ providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	return nil, nil
}

func (d *Deluge) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	return nil, fmt.Errorf("streaming not supported for deluge")
}

func (d *Deluge) Collect(ctx context.Context, inst providers.Instance) ([]providers.Sample, error) {
	now := time.Now()
	cli := newClient()
	if err := authenticate(ctx, cli, inst); err != nil {
		return nil, fmt.Errorf("deluge collect: %w", err)
	}
	result, err := rpc(ctx, cli, inst.BaseURL, "core.get_session_status", []any{[]string{"upload_rate", "download_rate", "num_connections"}})
	if err != nil {
		return nil, fmt.Errorf("deluge get_session_status: %w", err)
	}
	status, _ := result.(map[string]any)
	downloadRate, _ := status["download_rate"].(float64)
	uploadRate, _ := status["upload_rate"].(float64)
	numConns, _ := status["num_connections"].(float64)
	return []providers.Sample{
		{Metric: "deluge_download_rate_bytes", Value: downloadRate, TS: now},
		{Metric: "deluge_upload_rate_bytes", Value: uploadRate, TS: now},
		{Metric: "deluge_num_connections", Value: numConns, TS: now},
	}, nil
}

func (d *Deluge) ExportConfig(ctx context.Context, inst providers.Instance) (providers.ConfigBlob, error) {
	cli := newClient()
	if err := authenticate(ctx, cli, inst); err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("deluge export: %w", err)
	}
	result, err := rpc(ctx, cli, inst.BaseURL, "core.get_config", []any{})
	if err != nil {
		return providers.ConfigBlob{}, fmt.Errorf("deluge export config: %w", err)
	}
	b, err := json.Marshal(result)
	if err != nil {
		return providers.ConfigBlob{}, err
	}
	return providers.ConfigBlob{ContentType: "application/json", Data: b}, nil
}

func (d *Deluge) ImportConfig(_ context.Context, _ providers.Instance, _ providers.ConfigBlob) error {
	return nil
}
