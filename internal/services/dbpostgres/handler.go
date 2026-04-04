package dbpostgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

// validName matches a simple identifier (letters, digits, underscores, hyphens).
var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type FlexibleServer struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Type       string           `json:"type"`
	Location   string           `json:"location"`
	Properties ServerProperties `json:"properties"`
	SKU        SKU              `json:"sku"`
}

type ServerProperties struct {
	FQDN               string           `json:"fullyQualifiedDomainName"`
	State               string           `json:"state"`
	Version             string           `json:"version"`
	AdministratorLogin  string           `json:"administratorLogin"`
	Storage             StorageProps     `json:"storage"`
	Backup              BackupProps      `json:"backup"`
	Network             NetworkProps     `json:"network"`
	HighAvailability    HAProps          `json:"highAvailability"`
}

type StorageProps struct {
	StorageSizeGB int `json:"storageSizeGB"`
}

type BackupProps struct {
	BackupRetentionDays int `json:"backupRetentionDays"`
}

type NetworkProps struct {
	PublicNetworkAccess string `json:"publicNetworkAccess"`
}

type HAProps struct {
	Mode string `json:"mode"`
}

type SKU struct {
	Name string `json:"name"`
	Tier string `json:"tier"`
}

type Database struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties DatabaseProps  `json:"properties"`
}

type DatabaseProps struct {
	Charset   string `json:"charset"`
	Collation string `json:"collation"`
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	base := "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers"
	r.Route(base, func(r chi.Router) {
		r.Get("/", h.ListServers)
		r.Route("/{serverName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateServer)
			r.Get("/", h.GetServer)
			r.Delete("/", h.DeleteServer)
			r.Route("/databases", func(r chi.Router) {
				r.Get("/", h.ListDatabases)
				r.Route("/{dbName}", func(r chi.Router) {
					r.Put("/", h.CreateOrUpdateDatabase)
					r.Get("/", h.GetDatabase)
					r.Delete("/", h.DeleteDatabase)
				})
			})
		})
	})
}

// ---------------------------------------------------------------------------
// Store keys
// ---------------------------------------------------------------------------

func (h *Handler) serverKey(sub, rg, name string) string {
	return "pgflex:server:" + sub + ":" + rg + ":" + name
}

func (h *Handler) dbKey(sub, rg, server, db string) string {
	return "pgflex:db:" + sub + ":" + rg + ":" + server + ":" + db
}

// ---------------------------------------------------------------------------
// Postgres helpers
// ---------------------------------------------------------------------------

// postgresURL returns the POSTGRES_URL env var (or empty string).
func postgresURL() string {
	return os.Getenv("POSTGRES_URL")
}

// openAdmin opens a connection to the admin Postgres instance.
func openAdmin() (*sql.DB, error) {
	return sql.Open("postgres", postgresURL())
}

// hostPort extracts host and port from POSTGRES_URL for response metadata.
func hostPort() (string, string) {
	raw := postgresURL()
	if raw == "" {
		return "localhost", "5432"
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "localhost", "5432"
	}
	host := u.Hostname()
	port := u.Port()
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	return host, port
}

// execDDL runs a DDL statement against the admin Postgres.
func execDDL(stmt string) error {
	db, err := openAdmin()
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer db.Close()
	_, err = db.Exec(stmt)
	return err
}

// dbExists checks if a database exists on the real Postgres.
func dbExists(name string) (bool, error) {
	db, err := openAdmin()
	if err != nil {
		return false, err
	}
	defer db.Close()
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists)
	return exists, err
}

// ---------------------------------------------------------------------------
// Server CRUD
// ---------------------------------------------------------------------------

func (h *Handler) buildServer(sub, rg, name string) FlexibleServer {
	host, _ := hostPort()
	return FlexibleServer{
		ID:       "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.DBforPostgreSQL/flexibleServers/" + name,
		Name:     name,
		Type:     "Microsoft.DBforPostgreSQL/flexibleServers",
		Location: "eastus",
		Properties: ServerProperties{
			FQDN:              host,
			State:              "Ready",
			Version:            "16",
			AdministratorLogin: "miniblue",
			Storage:            StorageProps{StorageSizeGB: 32},
			Backup:             BackupProps{BackupRetentionDays: 7},
			Network:            NetworkProps{PublicNetworkAccess: "Enabled"},
			HighAvailability:   HAProps{Mode: "Disabled"},
		},
		SKU: SKU{Name: "Standard_B1ms", Tier: "Burstable"},
	}
}

func (h *Handler) CreateOrUpdateServer(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "serverName")

	if !validName.MatchString(name) {
		azerr.BadRequest(w, "Invalid server name.")
		return
	}

	// Accept and ignore the request body (we use fixed defaults).
	// Drain body so the client doesn't get a connection-reset.
	var body json.RawMessage
	_ = json.NewDecoder(r.Body).Decode(&body)

	srv := h.buildServer(sub, rg, name)
	k := h.serverKey(sub, rg, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, srv)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(srv)
}

func (h *Handler) GetServer(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "serverName")

	v, ok := h.store.Get(h.serverKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.DBforPostgreSQL/flexibleServers", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "serverName")

	k := h.serverKey(sub, rg, name)
	if !h.store.Delete(k) {
		azerr.NotFound(w, "Microsoft.DBforPostgreSQL/flexibleServers", name)
		return
	}

	// If POSTGRES_URL is set, drop all real databases that belong to this server.
	if postgresURL() != "" {
		prefix := h.dbKey(sub, rg, name, "")
		items := h.store.ListByPrefix(prefix)
		for _, item := range items {
			if db, ok := item.(Database); ok {
				if err := execDDL(fmt.Sprintf("DROP DATABASE IF EXISTS %s", quoteIdent(db.Name))); err != nil {
					log.Printf("dbpostgres: warning: failed to drop database %s: %v", db.Name, err)
				}
			}
		}
	}

	// Remove metadata for child databases.
	h.store.DeleteByPrefix(h.dbKey(sub, rg, name, ""))
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListServers(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("pgflex:server:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// ---------------------------------------------------------------------------
// Database CRUD
// ---------------------------------------------------------------------------

func (h *Handler) buildDatabase(sub, rg, server, dbName string) Database {
	return Database{
		ID:   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.DBforPostgreSQL/flexibleServers/" + server + "/databases/" + dbName,
		Name: dbName,
		Type: "Microsoft.DBforPostgreSQL/flexibleServers/databases",
		Properties: DatabaseProps{
			Charset:   "UTF8",
			Collation: "en_US.utf8",
		},
	}
}

func (h *Handler) CreateOrUpdateDatabase(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	server := chi.URLParam(r, "serverName")
	dbName := chi.URLParam(r, "dbName")

	if !validName.MatchString(dbName) {
		azerr.BadRequest(w, "Invalid database name.")
		return
	}

	// Parent server must exist.
	if !h.store.Exists(h.serverKey(sub, rg, server)) {
		azerr.NotFound(w, "Microsoft.DBforPostgreSQL/flexibleServers", server)
		return
	}

	// Drain body.
	var body json.RawMessage
	_ = json.NewDecoder(r.Body).Decode(&body)

	k := h.dbKey(sub, rg, server, dbName)
	_, exists := h.store.Get(k)

	// Real Postgres: create the database if POSTGRES_URL is set.
	if postgresURL() != "" && !exists {
		already, err := dbExists(dbName)
		if err != nil {
			log.Printf("dbpostgres: warning: could not check existence of %s: %v", dbName, err)
		}
		if !already {
			if err := execDDL(fmt.Sprintf("CREATE DATABASE %s", quoteIdent(dbName))); err != nil {
				azerr.WriteError(w, http.StatusInternalServerError, "InternalError",
					fmt.Sprintf("Failed to create database: %v", err))
				return
			}
		}
	}

	dbObj := h.buildDatabase(sub, rg, server, dbName)
	h.store.Set(k, dbObj)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(dbObj)
}

func (h *Handler) GetDatabase(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	server := chi.URLParam(r, "serverName")
	dbName := chi.URLParam(r, "dbName")

	v, ok := h.store.Get(h.dbKey(sub, rg, server, dbName))
	if !ok {
		azerr.NotFound(w, "Microsoft.DBforPostgreSQL/flexibleServers/databases", dbName)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteDatabase(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	server := chi.URLParam(r, "serverName")
	dbName := chi.URLParam(r, "dbName")

	k := h.dbKey(sub, rg, server, dbName)
	if !h.store.Delete(k) {
		azerr.NotFound(w, "Microsoft.DBforPostgreSQL/flexibleServers/databases", dbName)
		return
	}

	// Real Postgres: drop the database.
	if postgresURL() != "" {
		if err := execDDL(fmt.Sprintf("DROP DATABASE IF EXISTS %s", quoteIdent(dbName))); err != nil {
			log.Printf("dbpostgres: warning: failed to drop database %s: %v", dbName, err)
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListDatabases(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	server := chi.URLParam(r, "serverName")
	items := h.store.ListByPrefix(h.dbKey(sub, rg, server, ""))
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// quoteIdent quotes a SQL identifier to prevent injection.
// It doubles any embedded double-quotes and wraps in double-quotes.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
