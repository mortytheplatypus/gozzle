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
	subs    map[string]map[chan LogEntry]struct{}
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
		subs:      make(map[string]map[chan LogEntry]struct{}),
	}
}

func (s *Service) worker(w *serviceWorker) {
	defer s.wg.Done()
	for job := range w.ingest {
		err := s.store.Append(job.entry)
		if err == nil {
			s.broadcast(job.entry.ServiceID, *job.entry)
		}
		job.result <- err
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

func (s *Service) broadcast(serviceID string, entry LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for ch := range s.subs[serviceID] {
		select {
		case ch <- entry:
		default:
			// drop if subscriber is slow
		}
	}
}

func (s *Service) Subscribe(serviceID string) (<-chan LogEntry, func(), error) {

	if err := ValidateServiceID(serviceID); err != nil {
		return nil, nil, err
	}

	if !s.store.IsRegistered(serviceID) {
		return nil, nil, ErrNotRegistered
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, nil, ErrBusy
	}

	ch := make(chan LogEntry, s.queueSize)
	if s.subs[serviceID] == nil {
		s.subs[serviceID] = make(map[chan LogEntry]struct{})
	}
	s.subs[serviceID][ch] = struct{}{}

	var once sync.Once
	unsub := func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			if set, ok := s.subs[serviceID]; ok {
				if _, ok := set[ch]; ok {
					delete(set, ch)
					close(ch)
				}
				if len(set) == 0 {
					delete(s.subs, serviceID)
				}
			}
		})
	}

	return ch, unsub, nil
}

func (s *Service) closeSubscribersLocked() {
	for id, set := range s.subs {
		for ch := range set {
			close(ch)
		}
		delete(s.subs, id)
	}
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

	s.mu.Lock()
	s.closeSubscribersLocked()
	s.mu.Unlock()

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

func (s *Service) NextID(serviceID string) (int64, error) {
	return s.store.NextID(serviceID)
}
