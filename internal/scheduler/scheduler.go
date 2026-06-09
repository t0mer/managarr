// internal/scheduler/scheduler.go
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/t0mer/galactica/internal/metrics"
	"github.com/t0mer/galactica/internal/providers"
	"github.com/t0mer/galactica/internal/storage"
)

// Scheduler runs periodic log collection and metric sampling.
type Scheduler struct {
	c         *cron.Cron
	store     storage.Store
	secretKey string
	log       *slog.Logger
}

// New creates and configures the scheduler but does not start it.
func New(store storage.Store, secretKey string, log *slog.Logger) *Scheduler {
	s := &Scheduler{
		c:         cron.New(),
		store:     store,
		secretKey: secretKey,
		log:       log,
	}
	s.c.AddFunc("@every 5m", s.collectLogs)    //nolint:errcheck
	s.c.AddFunc("@every 1m", s.collectMetrics) //nolint:errcheck
	return s
}

// Start begins the cron scheduler.
func (s *Scheduler) Start() {
	s.c.Start()
	s.log.Info("scheduler started")
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	s.c.Stop()
}

func (s *Scheduler) collectLogs() {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	db := s.store.DB()
	instances, err := storage.ListInstances(db)
	if err != nil {
		s.log.Error("scheduler: list instances for logs", "error", err)
		return
	}

	for _, inst := range instances {
		if !inst.Enabled {
			continue
		}
		p, err := providers.Get(providers.Kind(inst.Kind))
		if err != nil {
			continue
		}
		ls, ok := p.(providers.LogSource)
		if !ok {
			continue
		}
		resolved := s.resolveInstance(inst)
		since := time.Now().Add(-6 * time.Minute)
		entries, err := ls.FetchLogs(ctx, resolved, since)
		if err != nil {
			s.log.Warn("scheduler: fetch logs", "instance", inst.Name, "error", err)
			continue
		}
		rows := make([]storage.LogEntryRow, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, storage.LogEntryRow{
				InstanceID: inst.ID,
				TS:         e.Timestamp,
				Level:      e.Level,
				SourceType: e.Source,
				Message:    e.Message,
				Raw:        e.Raw,
			})
			metrics.LogEntriesTotal.WithLabelValues(inst.ID, e.Level).Inc()
		}
		if err := storage.InsertLogEntries(db, rows); err != nil {
			s.log.Warn("scheduler: insert log entries", "instance", inst.Name, "error", err)
		}
	}
}

func (s *Scheduler) collectMetrics() {
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()

	db := s.store.DB()
	instances, err := storage.ListInstances(db)
	if err != nil {
		s.log.Error("scheduler: list instances for metrics", "error", err)
		return
	}

	for _, inst := range instances {
		if !inst.Enabled {
			continue
		}
		p, err := providers.Get(providers.Kind(inst.Kind))
		if err != nil {
			continue
		}
		ms, ok := p.(providers.MetricSource)
		if !ok {
			continue
		}
		resolved := s.resolveInstance(inst)
		samples, err := ms.Collect(ctx, resolved)
		if err != nil {
			s.log.Warn("scheduler: collect metrics", "instance", inst.Name, "error", err)
			continue
		}
		rows := make([]storage.SampleRow, 0, len(samples))
		for _, smpl := range samples {
			rows = append(rows, storage.SampleRow{
				InstanceID: inst.ID,
				Metric:     smpl.Metric,
				TS:         smpl.TS,
				Value:      smpl.Value,
			})
			metrics.InstanceMetric.WithLabelValues(inst.ID, inst.Name, inst.Kind, smpl.Metric).Set(smpl.Value)
		}
		if err := storage.InsertSamples(db, rows); err != nil {
			s.log.Warn("scheduler: insert samples", "instance", inst.Name, "error", err)
		}
	}
}

func (s *Scheduler) resolveInstance(row storage.InstanceRow) providers.Instance {
	inst := providers.Instance{
		ID:      row.ID,
		Kind:    providers.Kind(row.Kind),
		Name:    row.Name,
		BaseURL: row.BaseURL,
	}
	if s.secretKey != "" {
		enc, err := storage.GetSecret(s.store.DB(), row.ID, "api_key")
		if err == nil && len(enc) > 0 {
			plain, err := storage.Decrypt(enc, s.secretKey)
			if err == nil {
				inst.APIKey = string(plain)
			}
		}
	}
	return inst
}
