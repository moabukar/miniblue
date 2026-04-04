package sqldb

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers", func(r chi.Router) {
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
	return "sqldb:server:" + sub + ":" + rg + ":" + name
}

func (h *Handler) dbKey(sub, rg, server, db string) string {
	return "sqldb:db:" + sub + ":" + rg + ":" + server + ":" + db
}

func buildServerResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Sql/servers/" + name

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

	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.Sql/servers",
		"location": location,
		"properties": map[string]interface{}{
			"fullyQualifiedDomainName": "localhost",
			"state":                    "Ready",
			"version":                  "12.0",
			"administratorLogin":       adminLogin,
			"provisioningState":        "Succeeded",
		},
	}
}

func buildDatabaseResponse(sub, rg, server, db string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Sql/servers/" + server + "/databases/" + db

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	return map[string]interface{}{
		"id":       id,
		"name":     db,
		"type":     "Microsoft.Sql/servers/databases",
		"location": location,
		"properties": map[string]interface{}{
			"status":                       "Online",
			"collation":                    "SQL_Latin1_General_CP1_CI_AS",
			"maxSizeBytes":                 2147483648,
			"currentServiceObjectiveName":  "Basic",
			"provisioningState":            "Succeeded",
		},
		"sku": map[string]interface{}{
			"name":     "Basic",
			"tier":     "Basic",
			"capacity": 5,
		},
	}
}

func (h *Handler) CreateOrUpdateServer(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "serverName")

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
		azerr.NotFound(w, "Microsoft.Sql/servers", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "serverName")

	if !h.store.Delete(h.serverKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Sql/servers", name)
		return
	}
	// Clean up databases under this server
	h.store.DeleteByPrefix(h.dbKey(sub, rg, name, ""))
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListServers(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("sqldb:server:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) CreateOrUpdateDatabase(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	server := chi.URLParam(r, "serverName")
	dbName := chi.URLParam(r, "dbName")

	if !h.store.Exists(h.serverKey(sub, rg, server)) {
		azerr.NotFound(w, "Microsoft.Sql/servers", server)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	db := buildDatabaseResponse(sub, rg, server, dbName, input)
	k := h.dbKey(sub, rg, server, dbName)
	_, exists := h.store.Get(k)
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
		azerr.NotFound(w, "Microsoft.Sql/servers/databases", dbName)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteDatabase(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	server := chi.URLParam(r, "serverName")
	dbName := chi.URLParam(r, "dbName")

	if !h.store.Delete(h.dbKey(sub, rg, server, dbName)) {
		azerr.NotFound(w, "Microsoft.Sql/servers/databases", dbName)
		return
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
