package store

import (
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// AuditRecord represents a persisted audit event with its hash chain metadata.
type AuditRecord struct {
	Sequence  int64             `json:"sequence"`
	Event     domain.AuditEvent `json:"event"`
	PrevHash  string            `json:"prev_hash"`
	EventHash string            `json:"event_hash"`
}

// formatTime formats a timestamp in UTC RFC3339Nano.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// parseTime parses a timestamp formatted in UTC RFC3339Nano.
func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}
