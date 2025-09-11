package utils

import (
	"context"
	"time"
)

// GetCurrentTimestamp returns current Unix timestamp in nanoseconds
func GetCurrentTimestamp() int64 {
	return time.Now().UnixNano()
}

// InitContext creates a context with timeout
func InitContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// FormatDateTime formats time to string in UTC
func FormatDateTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// ParseDateTime parses datetime string to time.Time
func ParseDateTime(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05.000Z", dateStr)
}
