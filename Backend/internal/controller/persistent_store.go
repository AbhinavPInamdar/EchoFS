package controller

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// PersistentState represents the controller state that survives restarts.
type PersistentState struct {
	ObjectModes    map[string]*ObjectModeState `json:"object_modes"`
	GlobalOverride string                      `json:"global_override"`
	CriticalKeys   map[string]bool             `json:"critical_keys"`
	EmergencyMode  bool                        `json:"emergency_mode"`
	LastPersisted  time.Time                   `json:"last_persisted"`
	Version        int64                       `json:"version"`
}

// FileStore persists controller state to a JSON file.
// This is simple and sufficient for a single-instance controller.
// For multi-instance, replace with etcd or Raft-backed store.
type FileStore struct {
	path    string
	mu      sync.Mutex
	version int64
}

// NewFileStore creates a new file-based persistent store.
func NewFileStore(path string) *FileStore {
	return &FileStore{
		path:    path,
		version: 0,
	}
}

// Save persists the current controller state to disk.
func (fs *FileStore) Save(state *PersistentState) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.version++
	state.Version = fs.version
	state.LastPersisted = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file first, then rename (atomic on most filesystems)
	tmpPath := fs.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	if err := os.Rename(tmpPath, fs.path); err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// Load reads the persisted state from disk.
// Returns nil state (not an error) if the file doesn't exist yet.
func (fs *FileStore) Load() (*PersistentState, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := os.ReadFile(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No persisted state yet
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state PersistentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	fs.version = state.Version
	return &state, nil
}
