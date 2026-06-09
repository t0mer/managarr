// internal/issues/fingerprint.go
package issues

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/t0mer/galactica/internal/storage"
)

var (
	reNumbers    = regexp.MustCompile(`\b\d+\b`)
	reUUIDs      = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	rePaths      = regexp.MustCompile(`(/[^\s/]+)+`)
	reTimestamps = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?`)
)

// normalizeMessage strips variable parts from a log message to produce a stable fingerprint base.
func normalizeMessage(msg string) string {
	msg = reTimestamps.ReplaceAllString(msg, "<ts>")
	msg = reUUIDs.ReplaceAllString(msg, "<uuid>")
	msg = rePaths.ReplaceAllString(msg, "<path>")
	msg = reNumbers.ReplaceAllString(msg, "<n>")
	return strings.Join(strings.Fields(msg), " ")
}

// Fingerprint produces a stable 16-hex-char fingerprint for (instanceID, level, message).
func Fingerprint(instanceID, level, message string) string {
	normalized := normalizeMessage(message)
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s", instanceID, level, normalized)))
	return hex.EncodeToString(h[:8])
}

// ImpactScore computes a 0–100 impact score for an issue.
// Weights: severity 40%, frequency 25%, error bonus 25%, recency 10%.
func ImpactScore(severity string, count int, lastSeen time.Time) float64 {
	var severityScore float64
	switch strings.ToLower(severity) {
	case "error", "fatal", "critical":
		severityScore = 100
	case "warn", "warning":
		severityScore = 50
	default:
		severityScore = 20
	}

	// Frequency: logarithmic scale capped at 100.
	freqScore := math.Min(100, float64(count)*10)

	// Error bonus: errors get extra weight.
	var errorBonus float64
	if strings.ToLower(severity) == "error" || strings.ToLower(severity) == "fatal" {
		errorBonus = 100
	}

	// Recency: decays over 24 hours.
	hoursSince := time.Since(lastSeen).Hours()
	recencyScore := math.Max(0, 100-hoursSince*4)

	return math.Min(100, severityScore*0.40+freqScore*0.25+errorBonus*0.25+recencyScore*0.10)
}

// FromLogs converts warn/error log entries to IssueRow candidates for upsertion.
func FromLogs(entries []storage.LogEntryRow) []storage.IssueRow {
	var out []storage.IssueRow
	for _, e := range entries {
		level := strings.ToLower(e.Level)
		if level != "error" && level != "warn" && level != "warning" && level != "fatal" {
			continue
		}
		fp := Fingerprint(e.InstanceID, e.Level, e.Message)
		score := ImpactScore(e.Level, 1, e.TS)
		out = append(out, storage.IssueRow{
			ID:          uuid.New().String(),
			InstanceID:  e.InstanceID,
			Fingerprint: fp,
			Title:       truncate(e.Message, 200),
			Severity:    level,
			ImpactScore: score,
			Status:      "open",
			FirstSeen:   e.TS,
			LastSeen:    e.TS,
			Count:       1,
		})
	}
	return out
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
