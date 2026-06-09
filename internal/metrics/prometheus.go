// internal/metrics/prometheus.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// InstanceMetric tracks numeric metrics per instance (e.g. sonarr_series_total).
	InstanceMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "galactica_instance_metric",
		Help: "Current value of a metric collected from a managed app instance.",
	}, []string{"instance_id", "instance_name", "kind", "metric"})

	// LogEntriesTotal counts log entries received per instance and level.
	LogEntriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "galactica_log_entries_total",
		Help: "Total log entries collected from managed app instances.",
	}, []string{"instance_id", "level"})

	// OpenIssues tracks open issue counts per instance and severity.
	OpenIssues = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "galactica_open_issues",
		Help: "Number of open issues per instance and severity.",
	}, []string{"instance_id", "severity"})
)
