package log

import (
	"sync"
	"time"
)

type appendJob struct {
	entry  *LogEntry
	result chan error
}

type serviceWorker struct {
	ingest chan appendJob
}

type Service struct {
	store     *Store
	queueSize int

	mu      sync.Mutex
	closed  bool
	workers map[string]*serviceWorker
	wg      sync.WaitGroup
}

func NewLogService(store *Store, queueSize int) *Service {
	if queueSize < 1 {
		queueSize = 1
	}
	return &Service{
		store:     store,
		queueSize: queueSize,
		workers:   make(map[string]*serviceWorker),
	}
}

func (s *Service) worker(w *serviceWorker) {
	defer s.wg.Done()
	for job := range w.ingest {
		job.result <- s.store.Append(job.entry)
	}
}

func (s *Service) startWorkerLocked(serviceID string) *serviceWorker {

	if w, ok := s.workers[serviceID]; ok {
		return w
	}

	w := &serviceWorker{ingest: make(chan appendJob, s.queueSize)}
	s.workers[serviceID] = w

	s.wg.Add(1)
	go s.worker(w)

	return w
}

func (s *Service) workerFor(serviceID string) (*serviceWorker, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, ErrBusy
	}

	if w, ok := s.workers[serviceID]; ok {
		return w, nil
	}

	if !s.store.IsRegistered(serviceID) {
		return nil, ErrNotRegistered
	}

	return s.startWorkerLocked(serviceID), nil
}

func (s *Service) Close() error {

	s.mu.Lock()

	s.closed = true
	for id, w := range s.workers {
		close(w.ingest)
		delete(s.workers, id)
	}

	s.mu.Unlock()

	s.wg.Wait()
	return s.store.Close()
}

func (s *Service) Register(serviceID string) error {

	if err := s.store.Register(serviceID); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.startWorkerLocked(serviceID)

	return nil
}

func (s *Service) Append(req AppendLogRequest) error {

	if err := ValidateServiceID(req.ServiceID); err != nil {
		return err
	}

	if !s.store.IsRegistered(req.ServiceID) {
		return ErrNotRegistered
	}

	w, err := s.workerFor(req.ServiceID)
	if err != nil {
		return err
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
	case w.ingest <- job:
	default:
		return ErrBusy
	}

	return <-job.result
}

func (s *Service) Read(serviceID string, offset int64) ([]LogEntry, error) {
	return s.store.Read(serviceID, offset)
}
