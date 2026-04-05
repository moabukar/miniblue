package store

import (
	"log"
	"os"
	"path/filepath"
)

// NewBackend creates the appropriate backend based on environment variables.
// Priority: PERSISTENCE=1 -> FileBackend, DATABASE_URL -> PostgreSQL, otherwise in-memory.
func NewBackend() Backend {
	if os.Getenv("PERSISTENCE") == "1" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("Failed to get home dir: %v, falling back to in-memory", err)
			return NewMemoryBackend()
		}
		path := filepath.Join(home, ".miniblue", "state.json")
		log.Printf("Using file persistence: %s", path)
		return NewFileBackend(path)
	}
	if url := os.Getenv("DATABASE_URL"); url != "" {
		pg, err := NewPostgresBackend(url)
		if err != nil {
			log.Printf("Failed to connect to Postgres: %v, falling back to in-memory", err)
			return NewMemoryBackend()
		}
		return pg
	}
	return NewMemoryBackend()
}
