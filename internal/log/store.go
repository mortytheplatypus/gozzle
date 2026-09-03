package log

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Store struct {
	logPath string
}

func NewStore(walDir string) *Store {
	return &Store{logPath: filepath.Join(walDir, "wal.log")}
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

	return json.NewEncoder(f).Encode(entry)
}
