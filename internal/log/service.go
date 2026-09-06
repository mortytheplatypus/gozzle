package log

import (
	"sync"
	"time"
)

type appendJob struct {
	entry  *LogEntry
	result chan error
}

type Service struct {
	store  *Store
	ingest chan appendJob
	wg     sync.WaitGroup
}

func NewLogService(store *Store, queueSize int) *Service {

	if queueSize < 1 {
		queueSize = 1
	}

	s := &Service{
		store:  store,
		ingest: make(chan appendJob, queueSize),
	}

	s.wg.Add(1)

	go s.worker()

	return s
}

func (s *Service) worker() {

	defer s.wg.Done()

	for job := range s.ingest {
		job.result <- s.store.Append(job.entry)
	}
}

func (s *Service) Close() error {
	close(s.ingest)
	s.wg.Wait()
	return s.store.Close()
}

func (s *Service) Register(serviceID string) error {
	return s.store.Register(serviceID)
}

func (s *Service) Append(req AppendLogRequest) error {

	if err := ValidateServiceID(req.ServiceID); err != nil {
		return err
	}

	if !s.store.IsRegistered(req.ServiceID) {
		return ErrNotRegistered
	}

	entry := &LogEntry{
		ServiceID: req.ServiceID,
		Message:   req.Message,
		Level:     req.Level,
		CreatedAt: time.Now(),
	}

	job := appendJob{
		entry:  entry,
		result: make(chan error, 1),
	}

	select {
	case s.ingest <- job:
	default:
		return ErrBusy
	}

	return <-job.result
}

func (s *Service) Read(serviceID string, offset int64) ([]LogEntry, error) {
	return s.store.Read(serviceID, offset)
}
