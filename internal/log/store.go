package log

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const dayLayout = "20060102"

type Store struct {
	walDir        string
	registryPath  string
	maxBytes      int64
	retentionDays int
	mu            sync.Mutex
	services      map[string]*serviceState
}

type serviceState struct {
	nextID int64
	wal    *os.File
	size   int64
	day    string
}

type registryFile struct {
	Services map[string]registryEntry `json:"services"`
}

type registryEntry struct {
	NextID int64 `json:"next_id"`
}

type logSegment struct {
	path string
	day  string
	seq  int // -1 active; 0 = .YYYYMMDD; 1+ = .YYYYMMDD.N
}

func NewStore(walDir, servicesFile string, maxBytes int, retentionDays int) (*Store, error) {

	s := &Store{
		walDir:        walDir,
		registryPath:  filepath.Join(walDir, filepath.Base(servicesFile)),
		maxBytes:      int64(maxBytes),
		retentionDays: retentionDays,
		services:      make(map[string]*serviceState),
	}

	if err := os.MkdirAll(walDir, 0o755); err != nil {
		return nil, err
	}

	if err := s.loadRegistry(); err != nil {
		return nil, err
	}

	if err := s.applyRetention(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

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

func (s *Store) IsRegistered(serviceID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.services[serviceID]
	return ok
}

func (s *Store) Register(serviceID string) error {
	if err := ValidateServiceID(serviceID); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

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
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	st.wal = f
	st.size = info.Size()
	if st.day == "" {
		st.day = info.ModTime().Format(dayLayout)
	}
	return nil
}

func (s *Store) nextRotatedPath(serviceID, day string) string {

	base := filepath.Join(s.walDir, serviceID+".log."+day)
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return base
	}

	for n := 1; ; n++ {
		p := fmt.Sprintf("%s.%d", base, n)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p
		}
	}
}

func (s *Store) rotate(st *serviceState, serviceID, day string) error {

	if st.wal == nil || st.size == 0 {
		st.day = time.Now().Format(dayLayout)
		return nil
	}

	if err := st.wal.Close(); err != nil {
		return err
	}
	st.wal = nil

	if err := os.Rename(s.logPath(serviceID), s.nextRotatedPath(serviceID, day)); err != nil {
		return err
	}

	if err := s.openWAL(st, serviceID); err != nil {
		return err
	}
	st.day = time.Now().Format(dayLayout)
	st.size = 0

	return s.applyRetention()
}

func (s *Store) maybeRotate(st *serviceState, serviceID string) error {

	today := time.Now().Format(dayLayout)
	if st.day != today {
		if err := s.rotate(st, serviceID, st.day); err != nil {
			return err
		}
	}

	if st.size >= s.maxBytes {
		return s.rotate(st, serviceID, st.day)
	}

	return nil
}

func (s *Store) applyRetention() error {

	cutoffDay := time.Now().AddDate(0, 0, -s.retentionDays).Format(dayLayout)

	entries, err := os.ReadDir(s.walDir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		seg, ok := parseRotatedName(e.Name())
		if !ok || seg.day >= cutoffDay {
			continue
		}
		if err := os.Remove(filepath.Join(s.walDir, e.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// parseRotatedName matches foo.log.YYYYMMDD or foo.log.YYYYMMDD.N (not active foo.log).
func parseRotatedName(name string) (logSegment, bool) {

	parts := strings.Split(name, ".")
	if len(parts) < 3 || len(parts) > 4 || parts[1] != "log" {
		return logSegment{}, false
	}

	day := parts[2]
	if _, err := time.Parse(dayLayout, day); err != nil {
		return logSegment{}, false
	}

	seg := logSegment{path: name, day: day, seq: 0}
	if len(parts) == 4 {
		n, err := strconv.Atoi(parts[3])
		if err != nil || n < 1 {
			return logSegment{}, false
		}
		seg.seq = n
	}

	return seg, true
}

func (s *Store) listLogSegments(serviceID string) ([]logSegment, error) {
	prefix := serviceID + ".log"
	entries, err := os.ReadDir(s.walDir)
	if err != nil {
		return nil, err
	}

	var segs []logSegment
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name != prefix && !strings.HasPrefix(name, prefix+".") {
			continue
		}
		if name == prefix {
			segs = append(segs, logSegment{path: filepath.Join(s.walDir, name), seq: -1})
			continue
		}
		seg, ok := parseRotatedName(name)
		if !ok {
			continue
		}
		seg.path = filepath.Join(s.walDir, name)
		segs = append(segs, seg)
	}

	sort.Slice(segs, func(i, j int) bool { return lessLogSegment(segs[i], segs[j]) })
	return segs, nil
}

func lessLogSegment(a, b logSegment) bool {

	if a.seq < 0 {
		return false
	}

	if b.seq < 0 {
		return true
	}

	if a.day != b.day {
		return a.day < b.day
	}

	return a.seq < b.seq
}

func (s *Store) Append(entry *LogEntry) error {

	if err := ValidateServiceID(entry.ServiceID); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	st, ok := s.services[entry.ServiceID]
	if !ok {
		return ErrNotRegistered
	}

	if err := s.openWAL(st, entry.ServiceID); err != nil {
		return err
	}

	if err := s.maybeRotate(st, entry.ServiceID); err != nil {
		return err
	}

	entry.ID = st.nextID
	st.nextID++
	if err := s.saveRegistry(); err != nil {
		st.nextID--
		return err
	}

	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	b = append(b, '\n')

	if _, err := st.wal.Write(b); err != nil {
		return err
	}
	st.size += int64(len(b))
	return nil
}

func (s *Store) Read(serviceID string, offset int64) ([]LogEntry, error) {

	if err := ValidateServiceID(serviceID); err != nil {
		return nil, err
	}

	s.mu.Lock()
	_, ok := s.services[serviceID]
	s.mu.Unlock()
	if !ok {
		return nil, ErrNotRegistered
	}

	segs, err := s.listLogSegments(serviceID)
	if err != nil {
		return nil, err
	}

	out := []LogEntry{}
	for _, seg := range segs {
		part, err := readLogFile(seg.path, offset)
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	return out, nil
}

func readLogFile(path string, offset int64) ([]LogEntry, error) {

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
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
