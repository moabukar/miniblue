package store

// Backend is the storage interface. Implementations include MemoryBackend and PostgresBackend.
type Backend interface {
	Set(key string, value interface{}) error
	Get(key string) (interface{}, bool)
	Delete(key string) bool
	List() []string
	ListByPrefix(prefix string) []interface{}
	CountByPrefix(prefix string) int
	Exists(key string) bool
	DeleteByPrefix(prefix string) int
}
