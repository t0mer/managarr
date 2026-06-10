// cmd/galactica/main.go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/kardianos/service"
	"github.com/spf13/pflag"

	"github.com/t0mer/galactica/internal/config"
	"github.com/t0mer/galactica/internal/scheduler"
	"github.com/t0mer/galactica/internal/server"
	"github.com/t0mer/galactica/internal/storage"
	"github.com/t0mer/galactica/internal/version"

	// Blank imports register all providers via their init() functions.
	_ "github.com/t0mer/galactica/internal/providers/deluge"
	_ "github.com/t0mer/galactica/internal/providers/emby"
	_ "github.com/t0mer/galactica/internal/providers/jackett"
	_ "github.com/t0mer/galactica/internal/providers/jellyfin"
	_ "github.com/t0mer/galactica/internal/providers/plex"
	_ "github.com/t0mer/galactica/internal/providers/servarr/lidarr"
	_ "github.com/t0mer/galactica/internal/providers/servarr/radarr"
	_ "github.com/t0mer/galactica/internal/providers/servarr/sonarr"
)

func main() {
	flags := pflag.NewFlagSet(version.BinaryName, pflag.ExitOnError)
	flags.String("config", "", "path to YAML config file")
	flags.String("listen", ":8080", "listen address")
	flags.String("log-level", "info", "log level (debug|info|warn|error)")
	flags.String("log-format", "json", "log format (json|text)")
	flags.String("service", "", "service action (install|uninstall|start|stop|restart)")
	flags.Bool("version", false, "print version and exit")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if v, _ := flags.GetBool("version"); v {
		fmt.Printf("%s %s (commit %s, built %s)\n", version.AppName, version.Version, version.Commit, version.Date)
		os.Exit(0)
	}

	cfg, err := config.Load(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	log := newLogger(cfg.Log.Level, cfg.Log.Format)
	slog.SetDefault(log)

	svcCfg := &service.Config{
		Name:        version.BinaryName,
		DisplayName: version.AppName,
		Description: version.AppName + " — media stack mission control",
	}

	prg := &program{cfg: cfg, log: log}
	svc, err := service.New(prg, svcCfg)
	if err != nil {
		log.Error("creating service", "error", err)
		os.Exit(1)
	}

	if action, _ := flags.GetString("service"); action != "" {
		if err := service.Control(svc, action); err != nil {
			log.Error("service control failed", "action", action, "error", err)
			os.Exit(1)
		}
		return
	}

	if err := svc.Run(); err != nil {
		log.Error("service run error", "error", err)
		os.Exit(1)
	}
}

type program struct {
	cfg    *config.Config
	log    *slog.Logger
	cancel context.CancelFunc
}

func (p *program) Start(_ service.Service) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	go p.run(ctx)
	return nil
}

func (p *program) run(ctx context.Context) {
	store, err := storage.Open(p.cfg.Storage.Driver, p.cfg.Storage.DSN)
	if err != nil {
		p.log.Error("opening storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	sched := scheduler.New(store, p.cfg.SecretKey, p.log)
	sched.Start()
	defer sched.Stop()

	srv := server.New(p.cfg, store, p.log)
	if err := srv.Start(ctx, p.cfg.Server.Listen); err != nil {
		p.log.Error("server error", "error", err)
	}
}

func (p *program) Stop(_ service.Service) error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func newLogger(level, format string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}
	if format == "text" {
		return slog.New(slog.NewTextHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, opts))
}
