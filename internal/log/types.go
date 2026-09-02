package log

import "time"

type LogEntry struct {
	ID        int64
	ServiceID string
	Level     string
	Message   string
	CreatedAt time.Time
}
