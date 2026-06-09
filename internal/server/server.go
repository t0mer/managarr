// internal/server/server.go
package server

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/t0mer/galactica/internal/api"
	"github.com/t0mer/galactica/internal/config"
	"github.com/t0mer/galactica/internal/storage"
)

// Server holds the dependencies needed to run the HTTP server.
type Server struct {
	cfg   *config.Config
	store storage.Store
	log   *slog.Logger
	deps  *api.Deps
}

// New creates a new Server with the given dependencies.
func New(cfg *config.Config, store storage.Store, log *slog.Logger) *Server {
	deps := &api.Deps{
		DB:        store.DB(),
		SecretKey: cfg.SecretKey,
		Log:       log,
	}
	return &Server{cfg: cfg, store: store, log: log, deps: deps}
}

// Start builds the chi router and listens until ctx is cancelled.
func (s *Server) Start(ctx context.Context, listen string) error {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	h := &api.HealthHandler{DB: s.store.DB()}
	r.Get("/api/v1/health", h.Health)
	r.Get("/version", h.Version)
	r.Get("/readyz", h.Ready)
	r.Handle("/metrics", promhttp.Handler())

	r.Mount("/api/docs", api.DocsHandler())

	// Instances
	inst := &api.InstancesHandler{Deps: s.deps}
	r.Route("/api/v1/instances", func(r chi.Router) {
		r.Get("/", inst.List)
		r.Post("/", inst.Create)
		r.Get("/{id}", inst.Get)
		r.Put("/{id}", inst.Update)
		r.Delete("/{id}", inst.Delete)
		r.Post("/{id}/test", inst.TestConnection)
		r.Patch("/{id}/enabled", inst.SetEnabled)
	})

	// Logs
	logs := &api.LogsHandler{Deps: s.deps}
	r.Get("/api/v1/logs", logs.List)
	r.Get("/api/v1/logs/stream", logs.Stream)

	// Metrics
	met := &api.MetricsHandler{Deps: s.deps}
	r.Get("/api/v1/metrics", met.Metrics)
	r.Get("/api/v1/metrics/series", met.Series)

	r.Handle("/*", spaHandler())

	srv := &http.Server{
		Addr:    listen,
		Handler: r,
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			s.log.Error("server shutdown error", "error", err)
		}
	}()

	s.log.Info("server started", "addr", listen)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

// spaHandler serves the embedded React SPA, falling back to index.html for
// any path that doesn't match a static asset (enabling client-side routing).
func spaHandler() http.Handler {
	sub, err := fs.Sub(webDist, "dist")
	if err != nil {
		panic(fmt.Sprintf("web embed: %v", err))
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			fileServer.ServeHTTP(w, r)
			return
		}
		if _, err := sub.Open(name); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
