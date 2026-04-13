package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/chaso/ai-usage-monitor/internal/usage"
)

// Store is a thread-safe cache backed by a local JSON file.
type Store struct {
	mu   sync.RWMutex
	path string
	prev *usage.Snapshot
}

func New(path string) *Store {
	return &Store{path: path}
}

// Write atomically persists the snapshot and updates the in-memory previous value.
func (s *Store) Write(snap usage.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("cache: mkdir: %w", err)
	}

	// Atomic write via temp file + rename.
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("cache: write tmp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("cache: rename: %w", err)
	}

	// Keep a copy for diffing on the next cycle.
	prev := snap
	s.prev = &prev
	return nil
}

// Read returns the last snapshot written to disk, or an error if unavailable.
func (s *Store) Read() (usage.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		return usage.Snapshot{}, fmt.Errorf("cache: read: %w", err)
	}

	var snap usage.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return usage.Snapshot{}, fmt.Errorf("cache: unmarshal: %w", err)
	}
	return snap, nil
}

// Previous returns the snapshot from the previous write cycle (in-memory only).
// Returns nil when no previous snapshot exists yet.
func (s *Store) Previous() *usage.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.prev
}
