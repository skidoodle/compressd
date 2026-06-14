package cache

import (
	"errors"

	"github.com/cockroachdb/pebble"
	"github.com/skidoodle/compressd/logger"
)

type pebbleLogger struct{}

func (s *pebbleLogger) Infof(format string, args ...any) {
	logger.LogInfof(format, args...)
}
func (s *pebbleLogger) Fatalf(format string, args ...any) {
	logger.LogDBFatal(format, args...)
}

// Store is a simple wrapper around our Pebble database.
type Store struct {
	db *pebble.DB
}

// Open initializes the database. If path is empty, it returns an error.
func Open(dir string, verbose bool) (*Store, error) {
	if dir == "" {
		return nil, errors.New("cache directory path cannot be empty")
	}
	opts := &pebble.Options{
		Logger: &pebbleLogger{},
	}
	db, err := pebble.Open(dir, opts)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Get looks up metadata by file path.
func (s *Store) Get(key []byte) ([]byte, error) {
	val, closer, err := s.db.Get(key)
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	buf := make([]byte, len(val))
	copy(buf, val)
	return buf, nil
}

func (s *Store) Set(key, value []byte) error {
	return s.db.Set(key, value, pebble.Sync)
}

func (s *Store) NewBatch() *pebble.Batch {
	return s.db.NewBatch()
}
