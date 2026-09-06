package log

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

type Store struct {
	logPath  string
	metaPath string
	nextID   int64
	wal      *os.File
}

type metaFile struct {
	NextID int64 `json:"next_id"`
}

func NewStore(walDir string) (*Store, error) {
	s := &Store{
		logPath:  filepath.Join(walDir, "wal.log"),
		metaPath: filepath.Join(walDir, "meta.json"),
		nextID:   1,
	}
	if err := s.loadMeta(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(walDir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(s.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	s.wal = f
	return s, nil
}

func (s *Store) Close() error {
	if s.wal == nil {
		return nil
	}
	err := s.wal.Close()
	s.wal = nil
	return err
}

func (s *Store) loadMeta() error {

	b, err := os.ReadFile(s.metaPath)
	if os.IsNotExist(err) {
		return s.saveMeta()
	}
	if err != nil {
		return err
	}

	var m metaFile
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	if m.NextID < 1 {
		m.NextID = 1
	}

	s.nextID = m.NextID

	return nil
}

func (s *Store) saveMeta() error {

	if err := os.MkdirAll(filepath.Dir(s.metaPath), 0o755); err != nil {
		return err
	}

	b, err := json.Marshal(metaFile{NextID: s.nextID})
	if err != nil {
		return err
	}

	return os.WriteFile(s.metaPath, append(b, '\n'), 0o644)
}

func (s *Store) Append(entry *LogEntry) error {
	entry.ID = s.nextID

	s.nextID++
	if err := s.saveMeta(); err != nil {
		s.nextID--
		return err
	}

	if err := json.NewEncoder(s.wal).Encode(entry); err != nil {
		return err
	}
	return nil
}

func (s *Store) Read(serviceID string, offset int64) ([]LogEntry, error) {

	f, err := os.Open(s.logPath)
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

		if e.ServiceID != serviceID || e.ID < offset {
			continue
		}

		out = append(out, e)
	}

	return out, sc.Err()
}
