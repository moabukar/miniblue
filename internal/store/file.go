package store

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
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

func (f *FileBackend) Save() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	keys := f.mem.List()
	state := make(map[string]interface{}, len(keys))
	for _, k := range keys {
		if v, ok := f.mem.Get(k); ok {
			state[k] = v
		}
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	dir := filepath.Dir(f.path)
	os.MkdirAll(dir, 0700)
	return os.WriteFile(f.path, data, 0644)
}

// Delegate all Backend methods to the in-memory backend
func (f *FileBackend) Set(key string, value interface{}) error { return f.mem.Set(key, value) }
func (f *FileBackend) Get(key string) (interface{}, bool)      { return f.mem.Get(key) }
func (f *FileBackend) Delete(key string) bool                  { return f.mem.Delete(key) }
func (f *FileBackend) List() []string                          { return f.mem.List() }
func (f *FileBackend) ListByPrefix(prefix string) []interface{} { return f.mem.ListByPrefix(prefix) }
func (f *FileBackend) CountByPrefix(prefix string) int         { return f.mem.CountByPrefix(prefix) }
func (f *FileBackend) Exists(key string) bool                  { return f.mem.Exists(key) }
func (f *FileBackend) DeleteByPrefix(prefix string) int        { return f.mem.DeleteByPrefix(prefix) }
func (f *FileBackend) Reset()                                  { f.mem.Reset() }
