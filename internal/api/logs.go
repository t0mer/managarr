// internal/api/logs.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/t0mer/galactica/internal/storage"
)

// LogsHandler handles /api/v1/logs routes.
type LogsHandler struct{ *Deps }

type logEntryResp struct {
	ID         int64  `json:"id"`
	InstanceID string `json:"instance_id"`
	Timestamp  string `json:"timestamp"`
	Level      string `json:"level"`
	Source     string `json:"source"`
	Message    string `json:"message"`
}

func toLogResp(r storage.LogEntryRow) logEntryResp {
	return logEntryResp{
		ID:         r.ID,
		InstanceID: r.InstanceID,
		Timestamp:  r.TS.Format(time.RFC3339),
		Level:      r.Level,
		Source:     r.SourceType,
		Message:    r.Message,
	}
}

// List handles GET /api/v1/logs
// Query params: instance_id, level, since (duration like "24h"), limit
func (h *LogsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	instanceID := q.Get("instance_id")
	level := q.Get("level")
	sinceStr := q.Get("since")
	limitStr := q.Get("limit")

	since := time.Now().Add(-24 * time.Hour)
	if sinceStr != "" {
		d, err := time.ParseDuration(sinceStr)
		if err == nil {
			since = time.Now().Add(-d)
		}
	}

	limit := 200
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	rows, err := storage.QueryLogs(h.DB, instanceID, level, since, limit)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]logEntryResp, len(rows))
	for i, row := range rows {
		out[i] = toLogResp(row)
	}
	jsonResponse(w, http.StatusOK, out)
}

// Stream handles GET /api/v1/logs/stream via Server-Sent Events.
// Query params: instance_id, level
// Backfills last 50 entries, then polls every 3 seconds for new ones.
func (h *LogsHandler) Stream(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	instanceID := q.Get("instance_id")
	level := q.Get("level")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Backfill last 50 entries.
	var lastID int64
	backfill, err := storage.QueryLogs(h.DB, instanceID, level, time.Now().Add(-1*time.Hour), 50)
	if err == nil {
		// backfill is in DESC order; send oldest first
		for i := len(backfill) - 1; i >= 0; i-- {
			entry := backfill[i]
			if entry.ID > lastID {
				lastID = entry.ID
			}
			sendSSEEvent(w, entry)
		}
		flusher.Flush()
	}

	ctx := r.Context()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			entries, err := storage.QueryLogsSinceID(h.DB, lastID, instanceID, level)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.ID > lastID {
					lastID = entry.ID
				}
				sendSSEEvent(w, entry)
			}
			if len(entries) > 0 {
				flusher.Flush()
			}
		}
	}
}

func sendSSEEvent(w http.ResponseWriter, entry storage.LogEntryRow) {
	data, err := json.Marshal(toLogResp(entry))
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data) //nolint:errcheck
}
