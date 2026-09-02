package log

import (
	"fmt"
	"time"
)

type Service struct {
}

func NewLogService() *Service {
	return &Service{}
}

func (s *Service) Append(req AppendLogRequest) error {

	entry := &LogEntry{
		ID:        1,
		ServiceID: req.ServiceID,
		Message:   req.Message,
		Level:     req.Level,
		CreatedAt: time.Now(),
	}

	fmt.Printf("log: id=%d service=%s level=%s message=%q at=%s\n",
		entry.ID,
		entry.ServiceID,
		entry.Level,
		entry.Message,
		entry.CreatedAt.Format(time.RFC3339),
	)

	return nil
}
