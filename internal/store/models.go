package store

import (
	"time"
)

// formatTime formats a timestamp in UTC RFC3339Nano.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// parseTime parses a timestamp formatted in UTC RFC3339Nano.
func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}
