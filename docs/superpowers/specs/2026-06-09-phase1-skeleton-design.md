# Phase 1 Skeleton — Design Spec

**Project:** Galactica  
**Date:** 2026-06-09  
**Approach:** Layered (A) — four sequential layers, each independently buildable

---

## Overview

Stand up the full project skeleton: Go binary with kardianos service integration, viper/pflag config, SQLite storage with complete schema, all provider interfaces and stub providers returning realistic fake data, and a React 18 + Vite + Tailwind + shadcn/ui SPA with 7 placeholder pages embedded in the binary. No real third-party integrations. Must build, run, and pass `make build test lint`.

---

## Layer 1: Go module + binary entrypoint + kardianos service

### Module
- `go.mod` at `github.com/t0mer/galactica`, Go 1.25
- `internal/version/version.go`: constants `AppName = "Galactica"`, `BinaryName = "galactica"`, `EnvPrefix = "GALACTICA"` plus `Version`, `Commit`, `Date` (set via `-ldflags`)

### `cmd/galactica/main.go`
- Defines all CLI flags with `pflag`, binds to viper
- Hands off to `github.com/kardianos/service`; `Start()` launches HTTP server in a goroutine, `Stop()` cancels the root context
- Binary runs identically as foreground process or OS service from day one

**Flags:**
```
--config       path to YAML config file
--listen       listen address (default :8080)
--log-level    debug/info/warn/error (default info)
--log-format   json/text (default json)
--service      install|uninstall|start|stop|restart
--version      print version and exit
```

### Logging
- `log/slog` initialised once at startup from `--log-level` / `--log-format`
- All packages receive a `*slog.Logger` — no global logger, no `fmt.Println`

### Tooling files created in this layer
- `Makefile` with targets: `build`, `dev`, `test`, `lint`, `ui-watch`, `docker`
- `.air.toml` for hot reload
- `.gitignore` additions: `bin/`, `web/dist/`, `web/node_modules/`
- `config/config.example.yaml` skeleton

---

## Layer 2: Config + Storage

### `internal/config/config.go`
- Typed `Config` struct; viper unmarshals into it
- Viper init: env prefix `GALACTICA_`, optional YAML from `--config`, pflag bindings
- Precedence: flags > env > YAML > defaults
- Post-load validation (e.g. `SecretKey` required if secrets feature is enabled)

### `internal/storage/`
```
storage.go          # Store interface + Open() factory
sqlite.go           # modernc.org/sqlite implementation (CGO_ENABLED=0)
crypto.go           # AES-256-GCM encrypt/decrypt keyed from GALACTICA_SECRET_KEY
migrations/
  001_initial.sql   # complete schema, all tables
```

- `Open()` returns a `Store` interface — SQLite now, swappable to Postgres later
- Migrations run at startup via `//go:embed migrations/*.sql` sequential runner (no external `goose` binary in Phase 1)
- `crypto.go` present but not called by Phase 1 code; Phase 2 imports it without ceremony

### `001_initial.sql` — full schema
Tables (from spec):
- `instances` (kind, name, base_url, enabled)
- `secrets` (instance_id FK, encrypted blob)
- `log_entries` (instance_id, ts, level, source_type, message, raw) — indexed on (instance_id, ts, level)
- `issues` (fingerprint, title, severity, impact_score, status, first_seen, last_seen, count)
- `samples` (instance_id, metric, ts, value) — indexed on (instance_id, metric, ts)
- `notify_channels` (type, config_encrypted, enabled, notify_on_success, notify_on_failure)
- `backup_targets`, `backups`, `sync_jobs`, `schedules`
- `settings`, `api_tokens`, `admin`

---

## Layer 3: Provider interfaces + stub providers

### `internal/providers/provider.go`
- `Kind` type string constants for all 8 apps
- `Instance` struct (ID, Kind, Name, BaseURL — secrets resolved separately, never serialised)
- Base `Provider` interface: `Kind() Kind`, `TestConnection(ctx, inst) error`
- Capability interfaces: `LogSource`, `MetricSource`, `ConfigBackup`, `Syncable`
- Registry: `map[Kind]Provider` with `Register()` and `Get()`

### Stub packages
```
internal/providers/
  provider.go
  servarr/
    client.go           # shared HTTP client stub (no real HTTP)
    sonarr/sonarr.go    # LogSource + MetricSource + ConfigBackup + Syncable
    radarr/radarr.go    # LogSource + MetricSource + ConfigBackup + Syncable
    lidarr/lidarr.go    # LogSource + MetricSource + ConfigBackup + Syncable
  jackett/jackett.go    # LogSource + MetricSource + ConfigBackup + Syncable
  deluge/deluge.go      # LogSource + MetricSource + ConfigBackup
  plex/plex.go          # LogSource + MetricSource
  emby/emby.go          # LogSource + MetricSource
  jellyfin/jellyfin.go  # LogSource + MetricSource
```

### Fake data quality
- `FetchLogs`: 20–50 entries with randomised timestamps/levels/messages, seeded from instance ID (deterministic per instance)
- `Collect`: plausible fixed metric values (e.g. Sonarr: 312 series, 18 missing, queue 4)
- `ExportConfig`: small valid JSON blob
- `Snapshot`/`Diff`: canned diff with 2–3 changes
- `TestConnection`: always returns `nil`
- All stubs honour `ctx` cancellation

### Registration
Each provider package uses `func init()` to call `providers.Register(...)`. `main.go` blank-imports all provider packages — no factory wiring in main.

---

## Layer 4: React scaffold + Go embed + API handlers

### `web/` scaffold
- Vite + React 18 + TypeScript + Tailwind CSS + shadcn/ui
- React Router with 7 routes: Dashboard, Logs, Issues, Apps, Backup, Sync, Settings
- Each page: title + "coming soon" card — no data calls
- Nav shell (sidebar + topbar with dark/light system-preference toggle) fully functional
- `web/src/lib/api.ts`: typed fetch wrapper for future use; `web/src/lib/types.ts`: shared TS types mirroring Go structs

### Go embed + SPA fallback
```
internal/server/
  embed.go    //go:embed ../../web/dist
  server.go   chi router, middleware, SPA fallback
```
- SPA fallback: any request not matching `/api/*` or `/metrics` returns `index.html`
- `web/dist/.gitkeep` placeholder lets embed compile before first `npm run build`

### Middleware stack (chi)
- `middleware.RequestID`
- `middleware.RealIP`
- `middleware.Logger` (using slog)
- `middleware.Recoverer`
- `middleware.Compress`

### API handlers (Phase 1 only)

| Method | Path | Response |
|---|---|---|
| `GET` | `/api/v1/health` | `{"status":"ok","version":"...","db":"ok"}` |
| `GET` | `/version` | `{"version":"...","commit":"...","date":"..."}` |
| `GET` | `/readyz` | 200 once DB open, 503 otherwise |
| `GET` | `/metrics` | Prometheus default registry (Go runtime metrics) |
| `GET` | `/api/docs/*` | Swagger UI + skeleton `openapi.yaml` |

No instance/log/backup/sync API endpoints — those come with their feature phases.

### `docker-compose.yml`
Single service, port 8080, `/data` volume for SQLite. `docker compose up` works from day one.

---

## What is explicitly out of scope for Phase 1

- Any real HTTP calls to Sonarr, Radarr, Lidarr, Jackett, Deluge, Plex, Emby, or Jellyfin
- Authentication (`GALACTICA_AUTH_ENABLED` defaults to false; middleware stub is a no-op)
- Notification channels, backup engines, sync jobs, scheduler
- Any `/api/v1/instances`, `/api/v1/logs`, `/api/v1/backup`, `/api/v1/sync`, `/api/v1/notify` endpoints
- Goreleaser config and multi-arch Docker image (`.goreleaser.yaml` stub only)

---

## Definition of done

- `make build` produces `bin/galactica` (embeds the built React SPA)
- `make test` passes (`go test ./...` with at minimum one table-driven test per provider stub)
- `make lint` clean (golangci-lint)
- `./bin/galactica` starts, serves the SPA at `http://localhost:8080`, and returns 200 on `/api/v1/health`
- `docker compose up` works
- `./bin/galactica --service install` does not error on Linux
