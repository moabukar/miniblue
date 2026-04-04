package store

import (
	"strings"
	"sync"
)

type Store struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

func New() *Store {
	return &Store{data: make(map[string]interface{})}
}

func (s *Store) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// SetIfNotExists returns true if set, false if key already exists.
func (s *Store) SetIfNotExists(key string, value interface{}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; ok {
		return false
	}
	s.data[key] = value
	return true
}

func (s *Store) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[key]
	return ok
}

func (s *Store) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; !ok {
		return false
	}
	delete(s.data, key)
	return true
}

func (s *Store) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

func (s *Store) ListByPrefix(prefix string) []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []interface{}
	for k, v := range s.data {
		if strings.HasPrefix(k, prefix) {
			results = append(results, v)
		}
	}
	return results
}

func (s *Store) CountByPrefix(prefix string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for k := range s.data {
		if strings.HasPrefix(k, prefix) {
			count++
		}
	}
	return count
}

// DeleteByPrefix removes all keys with the given prefix. Returns count deleted.
func (s *Store) DeleteByPrefix(prefix string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for k := range s.data {
		if strings.HasPrefix(k, prefix) {
			delete(s.data, k)
			count++
		}
	}
	return count
}
