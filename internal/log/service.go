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
		ServiceID: req.ServiceID,
		Message:   req.Message,
		Level:     req.Level,
		CreatedAt: time.Now(),
	}

	return s.store.Append(entry)
}

func (s *Service) Read(serviceID string, offset int64) ([]LogEntry, error) {
	return s.store.Read(serviceID, offset)
}
