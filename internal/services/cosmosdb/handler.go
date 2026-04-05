package cosmosdb

import (
	"encoding/json"
	"github.com/moabukar/miniblue/internal/azerr"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/store"
)

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	// ARM-style paths: used by Azure SDKs to enumerate and manage Cosmos DB accounts
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DocumentDB/databaseAccounts", func(r chi.Router) {
		r.Get("/", h.ListAccounts)
		r.Route("/{accountName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateAccount)
			r.Get("/", h.GetAccount)
			r.Delete("/", h.DeleteAccount)
			r.Route("/sqlDatabases", func(r chi.Router) {
				r.Get("/", h.ListSQLDatabases)
				r.Route("/{dbName}", func(r chi.Router) {
					r.Put("/", h.CreateOrUpdateSQLDatabase)
					r.Get("/", h.GetSQLDatabase)
					r.Delete("/", h.DeleteSQLDatabase)
					r.Route("/containers", func(r chi.Router) {
						r.Get("/", h.ListContainers)
						r.Route("/{containerName}", func(r chi.Router) {
							r.Put("/", h.CreateOrUpdateContainer)
							r.Get("/", h.GetContainer)
							r.Delete("/", h.DeleteContainer)
						})
					})
				})
			})
		})
	})

	// Data-plane paths: used for document operations
	r.Route("/cosmosdb/{accountName}/dbs/{dbName}/colls/{collName}/docs", func(r chi.Router) {
		r.Post("/", h.CreateDocument)
		r.Get("/", h.QueryDocuments)
		r.Route("/{docId}", func(r chi.Router) {
			r.Get("/", h.GetDocument)
			r.Put("/", h.ReplaceDocument)
			r.Delete("/", h.DeleteDocument)
		})
	})
}

func (h *Handler) key(account, db, coll, id string) string {
	return "cosmos:" + account + ":" + db + ":" + coll + ":" + id
}

func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	db := chi.URLParam(r, "dbName")
	coll := chi.URLParam(r, "collName")

	var doc map[string]interface{}
	json.NewDecoder(r.Body).Decode(&doc)

	id, _ := doc["id"].(string)
	if id == "" {
		azerr.BadRequest(w, "Document must contain an id property.")
		return
	}

	h.store.Set(h.key(account, db, coll, id), doc)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	db := chi.URLParam(r, "dbName")
	coll := chi.URLParam(r, "collName")
	docId := chi.URLParam(r, "docId")

	v, ok := h.store.Get(h.key(account, db, coll, docId))
	if !ok {
		azerr.NotFound(w, "Microsoft.DocumentDB/documents", docId)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) ReplaceDocument(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	db := chi.URLParam(r, "dbName")
	coll := chi.URLParam(r, "collName")
	docId := chi.URLParam(r, "docId")

	var doc map[string]interface{}
	json.NewDecoder(r.Body).Decode(&doc)
	doc["id"] = docId

	h.store.Set(h.key(account, db, coll, docId), doc)
	json.NewEncoder(w).Encode(doc)
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	db := chi.URLParam(r, "dbName")
	coll := chi.URLParam(r, "collName")
	docId := chi.URLParam(r, "docId")

	h.store.Delete(h.key(account, db, coll, docId))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) QueryDocuments(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	db := chi.URLParam(r, "dbName")
	coll := chi.URLParam(r, "collName")
	items := h.store.ListByPrefix("cosmos:" + account + ":" + db + ":" + coll + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"Documents": items, "_count": len(items)})
}

// --- ARM account handlers ---

func (h *Handler) accountKey(sub, rg, name string) string {
	return "cosmos:account:" + sub + ":" + rg + ":" + name
}

func (h *Handler) buildAccountResponse(sub, rg, name string) map[string]interface{} {
	return map[string]interface{}{
		"id":       "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.DocumentDB/databaseAccounts/" + name,
		"name":     name,
		"type":     "Microsoft.DocumentDB/databaseAccounts",
		"location": "eastus",
		"kind":     "GlobalDocumentDB",
		"properties": map[string]interface{}{
			"provisioningState":     "Succeeded",
			"documentEndpoint":      "https://" + name + ".documents.azure.com:443/",
			"databaseAccountOfferType": "Standard",
			"consistencyPolicy": map[string]interface{}{
				"defaultConsistencyLevel": "Session",
			},
			"locations": []map[string]interface{}{
				{"locationName": "East US", "failoverPriority": 0, "isZoneRedundant": false},
			},
		},
	}
}

func (h *Handler) CreateOrUpdateAccount(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "accountName")

	k := h.accountKey(sub, rg, name)
	_, exists := h.store.Get(k)

	acct := h.buildAccountResponse(sub, rg, name)
	h.store.Set(k, acct)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(acct)
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "accountName")

	v, ok := h.store.Get(h.accountKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.DocumentDB/databaseAccounts", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "accountName")

	if !h.store.Delete(h.accountKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.DocumentDB/databaseAccounts", name)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("cosmos:account:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// --- ARM SQL Database handlers ---

func (h *Handler) sqlDbKey(sub, rg, acct, dbName string) string {
	return "cosmos:sqldb:" + sub + ":" + rg + ":" + acct + ":" + dbName
}

func (h *Handler) CreateOrUpdateSQLDatabase(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	acct := chi.URLParam(r, "accountName")
	dbName := chi.URLParam(r, "dbName")

	k := h.sqlDbKey(sub, rg, acct, dbName)
	_, exists := h.store.Get(k)

	db := map[string]interface{}{
		"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.DocumentDB/databaseAccounts/" + acct + "/sqlDatabases/" + dbName,
		"name": dbName,
		"type": "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
		"properties": map[string]interface{}{
			"resource": map[string]interface{}{"id": dbName},
		},
	}
	h.store.Set(k, db)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(db)
}

func (h *Handler) GetSQLDatabase(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	acct := chi.URLParam(r, "accountName")
	dbName := chi.URLParam(r, "dbName")

	v, ok := h.store.Get(h.sqlDbKey(sub, rg, acct, dbName))
	if !ok {
		azerr.NotFound(w, "Microsoft.DocumentDB/databaseAccounts/sqlDatabases", dbName)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteSQLDatabase(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	acct := chi.URLParam(r, "accountName")
	dbName := chi.URLParam(r, "dbName")

	if !h.store.Delete(h.sqlDbKey(sub, rg, acct, dbName)) {
		azerr.NotFound(w, "Microsoft.DocumentDB/databaseAccounts/sqlDatabases", dbName)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListSQLDatabases(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	acct := chi.URLParam(r, "accountName")
	items := h.store.ListByPrefix("cosmos:sqldb:" + sub + ":" + rg + ":" + acct + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// --- ARM Container handlers ---

func (h *Handler) containerKey(sub, rg, acct, dbName, name string) string {
	return "cosmos:container:" + sub + ":" + rg + ":" + acct + ":" + dbName + ":" + name
}

func (h *Handler) CreateOrUpdateContainer(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	acct := chi.URLParam(r, "accountName")
	dbName := chi.URLParam(r, "dbName")
	name := chi.URLParam(r, "containerName")

	k := h.containerKey(sub, rg, acct, dbName, name)
	_, exists := h.store.Get(k)

	c := map[string]interface{}{
		"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.DocumentDB/databaseAccounts/" + acct + "/sqlDatabases/" + dbName + "/containers/" + name,
		"name": name,
		"type": "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers",
		"properties": map[string]interface{}{
			"resource": map[string]interface{}{
				"id": name,
				"partitionKey": map[string]interface{}{
					"paths": []string{"/id"},
					"kind":  "Hash",
				},
			},
		},
	}
	h.store.Set(k, c)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(c)
}

func (h *Handler) GetContainer(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	acct := chi.URLParam(r, "accountName")
	dbName := chi.URLParam(r, "dbName")
	name := chi.URLParam(r, "containerName")

	v, ok := h.store.Get(h.containerKey(sub, rg, acct, dbName, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteContainer(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	acct := chi.URLParam(r, "accountName")
	dbName := chi.URLParam(r, "dbName")
	name := chi.URLParam(r, "containerName")

	if !h.store.Delete(h.containerKey(sub, rg, acct, dbName, name)) {
		azerr.NotFound(w, "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers", name)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListContainers(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	acct := chi.URLParam(r, "accountName")
	dbName := chi.URLParam(r, "dbName")
	items := h.store.ListByPrefix("cosmos:container:" + sub + ":" + rg + ":" + acct + ":" + dbName + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
