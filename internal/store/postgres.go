package store

import (
	"database/sql"
	"encoding/json"
	"log"
	"strings"

	_ "github.com/lib/pq"
)

// PostgresBackend implements Backend using a PostgreSQL database.
type PostgresBackend struct {
	db *sql.DB
}

// NewPostgresBackend connects to Postgres and auto-creates the storage table.
func NewPostgresBackend(databaseURL string) (*PostgresBackend, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS miniblue_store (
		key TEXT PRIMARY KEY,
		value JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}

	log.Println("Connected to PostgreSQL backend")
	return &PostgresBackend{db: db}, nil
}

func (p *PostgresBackend) Set(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(
		`INSERT INTO miniblue_store (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value = $2`,
		key, data,
	)
	return err
}

func (p *PostgresBackend) Get(key string) (interface{}, bool) {
	var data []byte
	err := p.db.QueryRow(`SELECT value FROM miniblue_store WHERE key = $1`, key).Scan(&data)
	if err != nil {
		return nil, false
	}
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, false
	}
	return value, true
}

func (p *PostgresBackend) Delete(key string) bool {
	result, err := p.db.Exec(`DELETE FROM miniblue_store WHERE key = $1`, key)
	if err != nil {
		return false
	}
	n, _ := result.RowsAffected()
	return n > 0
}

func (p *PostgresBackend) List() []string {
	rows, err := p.db.Query(`SELECT key FROM miniblue_store`)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	keys := make([]string, 0)
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			keys = append(keys, k)
		}
	}
	return keys
}

func (p *PostgresBackend) ListByPrefix(prefix string) []interface{} {
	rows, err := p.db.Query(
		`SELECT value FROM miniblue_store WHERE key LIKE $1`,
		prefix+"%",
	)
	if err != nil {
		return []interface{}{}
	}
	defer rows.Close()

	results := make([]interface{}, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var v interface{}
		if err := json.Unmarshal(data, &v); err == nil {
			results = append(results, v)
		}
	}
	return results
}

func (p *PostgresBackend) CountByPrefix(prefix string) int {
	var count int
	// Escape LIKE wildcards in the prefix itself.
	escaped := strings.NewReplacer("%", "\\%", "_", "\\_").Replace(prefix)
	err := p.db.QueryRow(
		`SELECT COUNT(*) FROM miniblue_store WHERE key LIKE $1`,
		escaped+"%",
	).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

func (p *PostgresBackend) Exists(key string) bool {
	var exists bool
	err := p.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM miniblue_store WHERE key = $1)`, key).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func (p *PostgresBackend) DeleteByPrefix(prefix string) int {
	result, err := p.db.Exec(
		`DELETE FROM miniblue_store WHERE key LIKE $1`,
		prefix+"%",
	)
	if err != nil {
		return 0
	}
	n, _ := result.RowsAffected()
	return int(n)
}

func (p *PostgresBackend) Reset() {
	_, _ = p.db.Exec(`TRUNCATE miniblue_store`)
}
