package store

import (
	"strings"
	"sync"
)

// MemoryBackend is an in-memory implementation of the Backend interface.
type MemoryBackend struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewMemoryBackend creates a new in-memory backend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{data: make(map[string]interface{})}
}

func (m *MemoryBackend) Set(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *MemoryBackend) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	return v, ok
}

func (m *MemoryBackend) Delete(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; !ok {
		return false
	}
	delete(m.data, key)
	return true
}

func (m *MemoryBackend) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}

func (m *MemoryBackend) ListByPrefix(prefix string) []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var results []interface{}
	for k, v := range m.data {
		if strings.HasPrefix(k, prefix) {
			results = append(results, v)
		}
	}
	return results
}

func (m *MemoryBackend) CountByPrefix(prefix string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for k := range m.data {
		if strings.HasPrefix(k, prefix) {
			count++
		}
	}
	return count
}

func (m *MemoryBackend) Exists(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[key]
	return ok
}

func (m *MemoryBackend) DeleteByPrefix(prefix string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for k := range m.data {
		if strings.HasPrefix(k, prefix) {
			delete(m.data, k)
			count++
		}
	}
	return count
}

// Store wraps a Backend and provides the public API used by all handlers.
// It preserves backward compatibility -- handlers continue to use *store.Store.
type Store struct {
	backend Backend
}

// New creates a Store backed by an in-memory backend.
func New() *Store {
	return &Store{backend: NewMemoryBackend()}
}

// NewWithBackend creates a Store backed by the given Backend.
func NewWithBackend(b Backend) *Store {
	return &Store{backend: b}
}

func (s *Store) Set(key string, value interface{}) {
	// Errors from the backend are silently dropped to preserve the existing
	// fire-and-forget API that all handlers rely on.
	_ = s.backend.Set(key, value)
}

// SetIfNotExists returns true if set, false if key already exists.
func (s *Store) SetIfNotExists(key string, value interface{}) bool {
	if s.backend.Exists(key) {
		return false
	}
	_ = s.backend.Set(key, value)
	return true
}

func (s *Store) Exists(key string) bool {
	return s.backend.Exists(key)
}

func (s *Store) Get(key string) (interface{}, bool) {
	return s.backend.Get(key)
}

func (s *Store) Delete(key string) bool {
	return s.backend.Delete(key)
}

func (s *Store) List() []string {
	return s.backend.List()
}

func (s *Store) ListByPrefix(prefix string) []interface{} {
	return s.backend.ListByPrefix(prefix)
}

func (s *Store) CountByPrefix(prefix string) int {
	return s.backend.CountByPrefix(prefix)
}

func (s *Store) DeleteByPrefix(prefix string) int {
	return s.backend.DeleteByPrefix(prefix)
}
