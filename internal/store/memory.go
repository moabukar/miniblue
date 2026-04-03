package store

import "sync"

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
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			results = append(results, v)
		}
	}
	return results
}
