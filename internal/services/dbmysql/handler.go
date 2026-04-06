package dbmysql

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
	_ "github.com/go-sql-driver/mysql"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforMySQL/flexibleServers", func(r chi.Router) {
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

func (h *Handler) serverKey(sub, rg, name string) string {
	return "dbmysql:server:" + sub + ":" + rg + ":" + name
}

func (h *Handler) dbKey(sub, rg, server, db string) string {
	return "dbmysql:db:" + sub + ":" + rg + ":" + server + ":" + db
}

func mysqlURL() string {
	return os.Getenv("MYSQL_URL")
}

func openAdmin() (*sql.DB, error) {
	return sql.Open("mysql", mysqlURL())
}

func hostPort() (string, string) {
	raw := mysqlURL()
	if raw == "" {
		return "localhost", "3306"
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "localhost", "3306"
	}
	host := u.Hostname()
	port := u.Port()
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "3306"
	}
	return host, port
}

func execDDL(stmt string) error {
	db, err := openAdmin()
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer db.Close()
	_, err = db.Exec(stmt)
	return err
}

func dbExists(name string) (bool, error) {
	db, err := openAdmin()
	if err != nil {
		return false, err
	}
	defer db.Close()
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?)", name).Scan(&exists)
	return exists, err
}

func buildServerResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.DBforMySQL/flexibleServers/" + name

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	adminLogin := "miniblue"
	if props, ok := input["properties"].(map[string]interface{}); ok {
		if v, ok := props["administratorLogin"].(string); ok && v != "" {
			adminLogin = v
		}
	}

	host, port := hostPort()
	fqdn := host
	if port != "3306" {
		fqdn = fmt.Sprintf("%s:%s", host, port)
	}

	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.DBforMySQL/flexibleServers",
		"location": location,
		"properties": map[string]interface{}{
			"fullyQualifiedDomainName": fqdn,
			"state":                    "Ready",
			"version":                  "8.0",
			"administratorLogin":       adminLogin,
			"storage": map[string]interface{}{
				"storageSizeGB": 20,
			},
			"provisioningState": "Succeeded",
		},
		"sku": map[string]interface{}{
			"name": "Standard_B1ms",
			"tier": "Burstable",
		},
	}
}

func buildDatabaseResponse(sub, rg, server, db string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.DBforMySQL/flexibleServers/" + server + "/databases/" + db

	charset := "utf8mb4"
	collation := "utf8mb4_0900_ai_ci"
	if props, ok := input["properties"].(map[string]interface{}); ok {
		if v, ok := props["charset"].(string); ok && v != "" {
			charset = v
		}
		if v, ok := props["collation"].(string); ok && v != "" {
			collation = v
		}
	}

	return map[string]interface{}{
		"id":   id,
		"name": db,
		"type": "Microsoft.DBforMySQL/flexibleServers/databases",
		"properties": map[string]interface{}{
			"charset":           charset,
			"collation":         collation,
			"provisioningState": "Succeeded",
		},
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

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	srv := buildServerResponse(sub, rg, name, input)
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
		azerr.NotFound(w, "Microsoft.DBforMySQL/flexibleServers", name)
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
		azerr.NotFound(w, "Microsoft.DBforMySQL/flexibleServers", name)
		return
	}

	if mysqlURL() != "" {
		prefix := h.dbKey(sub, rg, name, "")
		items := h.store.ListByPrefix(prefix)
		for _, item := range items {
			if db, ok := item.(map[string]interface{}); ok {
				if dbName, ok := db["name"].(string); ok {
					if err := execDDL(fmt.Sprintf("DROP DATABASE IF EXISTS %s", quoteIdent(dbName))); err != nil {
						log.Printf("dbmysql: warning: failed to drop database %s: %v", dbName, err)
					}
				}
			}
		}
	}

	h.store.DeleteByPrefix(h.dbKey(sub, rg, name, ""))
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListServers(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("dbmysql:server:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
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

	if !h.store.Exists(h.serverKey(sub, rg, server)) {
		azerr.NotFound(w, "Microsoft.DBforMySQL/flexibleServers", server)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	k := h.dbKey(sub, rg, server, dbName)
	_, exists := h.store.Get(k)

	if mysqlURL() != "" && !exists {
		already, err := dbExists(dbName)
		if err != nil {
			log.Printf("dbmysql: warning: could not check existence of %s: %v", dbName, err)
		}
		if !already {
			if err := execDDL(fmt.Sprintf("CREATE DATABASE %s", quoteIdent(dbName))); err != nil {
				azerr.WriteError(w, http.StatusInternalServerError, "InternalError",
					fmt.Sprintf("Failed to create database: %v", err))
				return
			}
		}
	}

	db := buildDatabaseResponse(sub, rg, server, dbName, input)
	h.store.Set(k, db)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(db)
}

func (h *Handler) GetDatabase(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	server := chi.URLParam(r, "serverName")
	dbName := chi.URLParam(r, "dbName")

	v, ok := h.store.Get(h.dbKey(sub, rg, server, dbName))
	if !ok {
		azerr.NotFound(w, "Microsoft.DBforMySQL/flexibleServers/databases", dbName)
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
		azerr.NotFound(w, "Microsoft.DBforMySQL/flexibleServers/databases", dbName)
		return
	}

	if mysqlURL() != "" {
		if err := execDDL(fmt.Sprintf("DROP DATABASE IF EXISTS %s", quoteIdent(dbName))); err != nil {
			log.Printf("dbmysql: warning: failed to drop database %s: %v", dbName, err)
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

func quoteIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}
