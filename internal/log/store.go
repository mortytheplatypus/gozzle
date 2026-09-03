package log

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

type Store struct {
	logPath string
	nextID  int64
}

func NewStore(walDir string) (*Store, error) {

	s := &Store{
		logPath: filepath.Join(walDir, "wal.log"),
		nextID:  1,
	}

	n, err := countLines(s.logPath)
	if err != nil {
		return nil, err
	}
	s.nextID = n + 1

	return s, nil
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
	if err := json.NewEncoder(f).Encode(entry); err != nil {
		return err
	}

	s.nextID++
	return nil
}

// O(n) for small scale for now
func countLines(path string) (int64, error) {

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var n int64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		n++
	}
	return n, sc.Err()
}
