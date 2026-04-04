package store

import (
	"log"
	"os"
)

// NewBackend creates the appropriate backend based on DATABASE_URL env var.
// If DATABASE_URL is set, uses PostgreSQL. Otherwise uses in-memory.
func NewBackend() Backend {
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
