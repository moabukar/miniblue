package cosmosdb

import (
	"encoding/json"
	"github.com/moabukar/local-azure/internal/azerr"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
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
