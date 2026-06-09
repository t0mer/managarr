// internal/api/metrics.go
package api

import (
	"net/http"
	"time"

	"github.com/t0mer/galactica/internal/storage"
)

// MetricsHandler handles /api/v1/metrics routes.
type MetricsHandler struct{ *Deps }

type dataPoint struct {
	TS    string  `json:"ts"`
	Value float64 `json:"value"`
}

type metricSeriesResp struct {
	InstanceID string      `json:"instance_id"`
	Metric     string      `json:"metric"`
	Points     []dataPoint `json:"points"`
}

type metricNamesResp struct {
	InstanceID string   `json:"instance_id"`
	Metrics    []string `json:"metrics"`
}

// Metrics handles GET /api/v1/metrics?instance_id=<id>
// Returns distinct metric names available for the instance.
func (h *MetricsHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	instanceID := r.URL.Query().Get("instance_id")
	if instanceID == "" {
		jsonError(w, http.StatusBadRequest, "instance_id is required")
		return
	}
	names, err := storage.ListMetricNames(h.DB, instanceID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if names == nil {
		names = []string{}
	}
	jsonResponse(w, http.StatusOK, metricNamesResp{InstanceID: instanceID, Metrics: names})
}

// Series handles GET /api/v1/metrics/series?instance_id=<id>&metric=<name>&since=<duration>
// Returns time-series data points for the given metric.
func (h *MetricsHandler) Series(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	instanceID := q.Get("instance_id")
	metric := q.Get("metric")
	sinceStr := q.Get("since")

	if instanceID == "" || metric == "" {
		jsonError(w, http.StatusBadRequest, "instance_id and metric are required")
		return
	}

	since := time.Now().Add(-24 * time.Hour)
	if sinceStr != "" {
		d, err := time.ParseDuration(sinceStr)
		if err == nil {
			since = time.Now().Add(-d)
		}
	}

	rows, err := storage.QuerySeries(h.DB, instanceID, metric, since)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	points := make([]dataPoint, len(rows))
	for i, row := range rows {
		points[i] = dataPoint{TS: row.TS.Format(time.RFC3339), Value: row.Value}
	}
	jsonResponse(w, http.StatusOK, metricSeriesResp{
		InstanceID: instanceID,
		Metric:     metric,
		Points:     points,
	})
}
