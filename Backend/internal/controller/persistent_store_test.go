package controller

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStore_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "state.json")

	store := NewFileStore(path)

	state := &PersistentState{
		ObjectModes: map[string]*ObjectModeState{
			"obj-1": {
				ObjectID:    "obj-1",
				CurrentMode: "C",
				LastChange:  time.Now(),
				TTL:         30,
				Reason:      "initial",
			},
			"obj-2": {
				ObjectID:    "obj-2",
				CurrentMode: "A",
				LastChange:  time.Now(),
				TTL:         30,
				Reason:      "high_partition_risk",
			},
		},
		GlobalOverride: "C",
		CriticalKeys:   map[string]bool{"obj-1": true},
		EmergencyMode:  false,
	}

	// Save
	if err := store.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("state file was not created")
	}

	// Load
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded state is nil")
	}

	// Verify contents
	if len(loaded.ObjectModes) != 2 {
		t.Errorf("expected 2 object modes, got %d", len(loaded.ObjectModes))
	}
	if loaded.GlobalOverride != "C" {
		t.Errorf("expected global override 'C', got '%s'", loaded.GlobalOverride)
	}
	if !loaded.CriticalKeys["obj-1"] {
		t.Error("expected obj-1 to be a critical key")
	}
	if loaded.EmergencyMode {
		t.Error("expected emergency mode to be false")
	}
	if loaded.Version != 1 {
		t.Errorf("expected version 1, got %d", loaded.Version)
	}
}

func TestFileStore_LoadReturnsNilWhenNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.json")

	store := NewFileStore(path)

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load should not error for missing file: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil state for missing file")
	}
}

func TestFileStore_VersionIncrements(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "state.json")

	store := NewFileStore(path)

	state := &PersistentState{
		ObjectModes:  make(map[string]*ObjectModeState),
		CriticalKeys: make(map[string]bool),
	}

	store.Save(state)
	store.Save(state)
	store.Save(state)

	loaded, _ := store.Load()
	if loaded.Version != 3 {
		t.Errorf("expected version 3 after 3 saves, got %d", loaded.Version)
	}
}

func TestFileStore_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "state.json")

	store := NewFileStore(path)

	// Save initial state
	state := &PersistentState{
		ObjectModes:    map[string]*ObjectModeState{"obj-1": {CurrentMode: "C"}},
		GlobalOverride: "initial",
		CriticalKeys:   make(map[string]bool),
	}
	store.Save(state)

	// Verify no .tmp file remains
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful save")
	}
}
