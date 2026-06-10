// internal/providers/provider.go
package providers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Kind identifies the type of managed application.
type Kind string

const (
	KindSonarr   Kind = "sonarr"
	KindRadarr   Kind = "radarr"
	KindLidarr   Kind = "lidarr"
	KindJackett  Kind = "jackett"
	KindDeluge   Kind = "deluge"
	KindPlex     Kind = "plex"
	KindEmby     Kind = "emby"
	KindJellyfin Kind = "jellyfin"
)

// Instance represents one running installation of a managed app.
// Secrets are resolved from storage at call time and never serialised in API responses.
type Instance struct {
	ID      string
	Kind    Kind
	Name    string
	BaseURL string
	APIKey  string // resolved from secrets table, never in API responses
}

// LogEntry is a normalised log line from any provider.
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Source    string
	Message   string
	Raw       string
}

// Sample is a single time-series metric observation.
type Sample struct {
	Metric string
	Value  float64
	TS     time.Time
}

// ConfigBlob is an opaque config export from a provider.
type ConfigBlob struct {
	ContentType string
	Filename    string // original filename, e.g. "sonarr_backup_2024_01_01.zip"
	Data        []byte
}

// SyncChange describes a single field difference between two SyncState snapshots.
type SyncChange struct {
	Field    string
	OldValue any
	NewValue any
}

// SyncState is a snapshot of a provider instance's syncable configuration.
type SyncState struct {
	Data map[string]any
}

// Provider is the base interface every provider must implement.
type Provider interface {
	Kind() Kind
	TestConnection(ctx context.Context, inst Instance) error
}

// LogSource is implemented by providers that expose log entries.
type LogSource interface {
	FetchLogs(ctx context.Context, inst Instance, since time.Time) ([]LogEntry, error)
	StreamLogs(ctx context.Context, inst Instance) (<-chan LogEntry, error)
}

// MetricSource is implemented by providers that expose metrics.
type MetricSource interface {
	Collect(ctx context.Context, inst Instance) ([]Sample, error)
}

// ConfigBackup is implemented by providers that support config export/import.
type ConfigBackup interface {
	ExportConfig(ctx context.Context, inst Instance) (ConfigBlob, error)
	ImportConfig(ctx context.Context, inst Instance, b ConfigBlob) error
}

// Syncable is implemented by providers that support cross-instance config sync.
// Only same-Kind syncs are permitted.
type Syncable interface {
	Snapshot(ctx context.Context, inst Instance) (SyncState, error)
	Diff(a, b SyncState) []SyncChange
	Apply(ctx context.Context, inst Instance, changes []SyncChange) error
}

var (
	mu       sync.RWMutex
	registry = map[Kind]Provider{}
)

// Register adds p to the global provider registry.
// Called from init() in each provider package.
func Register(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	registry[p.Kind()] = p
}

// Get returns the registered provider for kind k.
func Get(k Kind) (Provider, error) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[k]
	if !ok {
		return nil, fmt.Errorf("no provider registered for kind %q", k)
	}
	return p, nil
}

// All returns all registered providers.
func All() []Provider {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Provider, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	return out
}
