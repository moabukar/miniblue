package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileBackend_AtomicSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	fb := NewFileBackend(path)
	fb.Set("key1", "value1")
	fb.Set("key2", 42.0)

	if err := fb.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Temp file must not exist after successful save
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file still exists after Save()")
	}

	// State file must exist and contain correct data
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if state["key1"] != "value1" {
		t.Errorf("key1: got %v, want value1", state["key1"])
	}
	if state["key2"] != 42.0 {
		t.Errorf("key2: got %v, want 42", state["key2"])
	}
}

func TestFileBackend_LoadExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Write initial state
	fb := NewFileBackend(path)
	fb.Set("persistent", "yes")
	if err := fb.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load a new backend from the same path
	fb2 := NewFileBackend(path)
	v, ok := fb2.Get("persistent")
	if !ok {
		t.Fatal("key 'persistent' not found after reload")
	}
	if v != "yes" {
		t.Errorf("got %v, want yes", v)
	}
}

func TestFileBackend_AutoSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	fb := NewFileBackend(path)
	stop := fb.StartAutoSave(50 * time.Millisecond)
	defer stop()

	fb.Set("auto", "saved")

	// Wait for at least one auto-save tick
	time.Sleep(150 * time.Millisecond)

	// File must exist now
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("auto-save did not create state file")
	}

	stop()

	// Reload and verify
	fb2 := NewFileBackend(path)
	v, ok := fb2.Get("auto")
	if !ok {
		t.Fatal("key 'auto' not found after auto-save")
	}
	if v != "saved" {
		t.Errorf("got %v, want saved", v)
	}
}

func TestStore_StartAutoSave_NoOp_MemoryBackend(t *testing.T) {
	s := New() // memory backend
	stop := s.StartAutoSave(10 * time.Millisecond)
	// Should not panic and stop should be callable
	time.Sleep(20 * time.Millisecond)
	stop()
}
