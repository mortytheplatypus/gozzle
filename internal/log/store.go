package log

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Store struct {
	logPath  string
	metaPath string
	nextID   int64
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
	return s, nil
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

	if err := os.MkdirAll(filepath.Dir(s.logPath), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(s.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	entry.ID = s.nextID

	s.nextID++
	if err := s.saveMeta(); err != nil {
		s.nextID--
		return err
	}

	if err := json.NewEncoder(f).Encode(entry); err != nil {
		return err
	}
	return nil
}
