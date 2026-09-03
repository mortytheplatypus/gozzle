package log

import (
	"time"
)

type Service struct {
	store *Store
}

func NewLogService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) Append(req AppendLogRequest) error {

	entry := &LogEntry{
		ID:        1,
		ServiceID: req.ServiceID,
		Message:   req.Message,
		Level:     req.Level,
		CreatedAt: time.Now(),
	}

	return s.store.Append(entry)
}
