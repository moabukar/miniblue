package store

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileBackend struct {
	mem  *MemoryBackend
	path string
	mu   sync.Mutex
}

func NewFileBackend(path string) *FileBackend {
	fb := &FileBackend{
		mem:  NewMemoryBackend(),
		path: path,
	}
	fb.load()
	return fb
}

func (f *FileBackend) load() {
	data, err := os.ReadFile(f.path)
	if err != nil {
		return // no file yet, start fresh
	}
	var state map[string]json.RawMessage
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("failed to load state from %s: %v", f.path, err)
		return
	}
	for k, v := range state {
		var val interface{}
		json.Unmarshal(v, &val)
		f.mem.Set(k, val)
	}
	log.Printf("loaded %d items from %s", len(state), f.path)
}

// Save atomically persists all in-memory state to disk.
// It writes to a .tmp file first and then renames it to avoid corruption on crash.
func (f *FileBackend) Save() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	state := f.mem.Snapshot()
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	// Write to a temp file first, then atomically rename to avoid corruption.
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}

// StartAutoSave starts a background goroutine that saves state to disk every
// interval. Call the returned stop function to halt it (e.g. on shutdown).
// The stop function is safe to call multiple times.
func (f *FileBackend) StartAutoSave(interval time.Duration) func() {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	var once sync.Once
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := f.Save(); err != nil {
					log.Printf("auto-save failed: %v", err)
				}
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	log.Printf("auto-save enabled: saving every %s to %s", interval, f.path)
	return func() { once.Do(func() { close(done) }) }
}

// Delegate all Backend methods to the in-memory backend
func (f *FileBackend) Set(key string, value interface{}) error { return f.mem.Set(key, value) }
func (f *FileBackend) Get(key string) (interface{}, bool)      { return f.mem.Get(key) }
func (f *FileBackend) Delete(key string) bool                  { return f.mem.Delete(key) }
func (f *FileBackend) List() []string                          { return f.mem.List() }
func (f *FileBackend) ListByPrefix(prefix string) []interface{} {
	return f.mem.ListByPrefix(prefix)
}
func (f *FileBackend) CountByPrefix(prefix string) int  { return f.mem.CountByPrefix(prefix) }
func (f *FileBackend) Exists(key string) bool           { return f.mem.Exists(key) }
func (f *FileBackend) DeleteByPrefix(prefix string) int { return f.mem.DeleteByPrefix(prefix) }
func (f *FileBackend) Reset()                           { f.mem.Reset() }
