package log

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

type Store struct {
	walDir       string
	registryPath string
	services     map[string]*serviceState
}

type serviceState struct {
	nextID int64
	wal    *os.File
}

type registryFile struct {
	Services map[string]registryEntry `json:"services"`
}

type registryEntry struct {
	NextID int64 `json:"next_id"`
}

func NewStore(walDir string) (*Store, error) {

	s := &Store{
		walDir:       walDir,
		registryPath: filepath.Join(walDir, "services.json"),
		services:     make(map[string]*serviceState),
	}

	if err := os.MkdirAll(walDir, 0o755); err != nil {
		return nil, err
	}

	if err := s.loadRegistry(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {

	var firstErr error
	for id, st := range s.services {
		if st.wal == nil {
			continue
		}

		if err := st.wal.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		st.wal = nil
		s.services[id] = st
	}
	return firstErr
}

func (s *Store) loadRegistry() error {

	b, err := os.ReadFile(s.registryPath)
	if os.IsNotExist(err) {
		return s.saveRegistry()
	}
	if err != nil {
		return err
	}

	var reg registryFile
	if err := json.Unmarshal(b, &reg); err != nil {
		return err
	}
	if reg.Services == nil {
		reg.Services = map[string]registryEntry{}
	}

	for id, e := range reg.Services {
		nextID := e.NextID
		if nextID < 1 {
			nextID = 1
		}
		s.services[id] = &serviceState{nextID: nextID}
	}
	return nil
}

func (s *Store) saveRegistry() error {

	reg := registryFile{Services: make(map[string]registryEntry, len(s.services))}
	for id, st := range s.services {
		reg.Services[id] = registryEntry{NextID: st.nextID}
	}

	b, err := json.Marshal(reg)
	if err != nil {
		return err
	}

	return os.WriteFile(s.registryPath, append(b, '\n'), 0o644)
}

func (s *Store) logPath(serviceID string) string {
	return filepath.Join(s.walDir, serviceID+".log")
}

func (s *Store) Register(serviceID string) error {
	if err := ValidateServiceID(serviceID); err != nil {
		return err
	}

	if _, ok := s.services[serviceID]; ok {
		return nil
	}

	s.services[serviceID] = &serviceState{nextID: 1}
	if err := s.saveRegistry(); err != nil {
		delete(s.services, serviceID)
		return err
	}

	f, err := os.OpenFile(s.logPath(serviceID), os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	return f.Close()
}

func (s *Store) openWAL(st *serviceState, serviceID string) error {
	if st.wal != nil {
		return nil
	}
	f, err := os.OpenFile(s.logPath(serviceID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	st.wal = f
	return nil
}

func (s *Store) Append(entry *LogEntry) error {
	if err := ValidateServiceID(entry.ServiceID); err != nil {
		return err
	}

	st, ok := s.services[entry.ServiceID]
	if !ok {
		return ErrNotRegistered
	}

	if err := s.openWAL(st, entry.ServiceID); err != nil {
		return err
	}

	entry.ID = st.nextID
	st.nextID++
	if err := s.saveRegistry(); err != nil {
		st.nextID--
		return err
	}

	if err := json.NewEncoder(st.wal).Encode(entry); err != nil {
		return err
	}
	return nil
}

func (s *Store) Read(serviceID string, offset int64) ([]LogEntry, error) {

	if err := ValidateServiceID(serviceID); err != nil {
		return nil, err
	}

	if _, ok := s.services[serviceID]; !ok {
		return nil, ErrNotRegistered
	}

	f, err := os.Open(s.logPath(serviceID))
	if os.IsNotExist(err) {
		return []LogEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := []LogEntry{}
	sc := bufio.NewScanner(f)

	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}

		var e LogEntry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, err
		}
		if e.ID < offset {
			continue
		}

		out = append(out, e)
	}

	return out, sc.Err()
}
