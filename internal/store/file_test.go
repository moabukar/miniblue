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

	fb.Set("auto", "saved")

	// Wait for at least one auto-save tick
	time.Sleep(150 * time.Millisecond)

	// Stop and let any in-flight save complete
	stop()
	time.Sleep(20 * time.Millisecond)

	// File must exist now
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("auto-save did not create state file")
	}

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

func TestFileBackend_ConcurrentWriteDuringSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	fb := NewFileBackend(path)

	for i := 0; i < 100; i++ {
		fb.Set("key-"+string(rune('a'+i%26)), i)
	}

	done := make(chan struct{})
	go func() {
		for j := 0; j < 50; j++ {
			fb.Set("concurrent", j)
			fb.Delete("key-a")
			fb.Set("key-a", j*10)
		}
		close(done)
	}()

	for k := 0; k < 10; k++ {
		if err := fb.Save(); err != nil {
			t.Errorf("Save() error during concurrent writes: %v", err)
		}
	}
	<-done

	if err := fb.Save(); err != nil {
		t.Fatalf("final Save() error: %v", err)
	}

	fb2 := NewFileBackend(path)
	if _, ok := fb2.Get("concurrent"); !ok {
		t.Error("key 'concurrent' missing after concurrent save/write")
	}
}

func TestMemoryBackend_Snapshot(t *testing.T) {
	m := NewMemoryBackend()
	m.Set("a", 1)
	m.Set("b", 2)

	snap := m.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 keys in snapshot, got %d", len(snap))
	}

	m.Set("c", 3)
	m.Delete("a")

	if len(snap) != 2 {
		t.Fatalf("snapshot should be isolated from mutations, got %d keys", len(snap))
	}
	if snap["a"] != 1 {
		t.Errorf("snapshot should still have a=1, got %v", snap["a"])
	}
}

func TestStore_StartAutoSave_NoOp_MemoryBackend(t *testing.T) {
	s := New() // memory backend
	stop := s.StartAutoSave(10 * time.Millisecond)
	// Should not panic and stop should be callable
	time.Sleep(20 * time.Millisecond)
	stop()
}
