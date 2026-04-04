package table

import (
	"encoding/json"
	"github.com/moabukar/local-azure/internal/azerr"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type Entity struct {
	PartitionKey string                 `json:"PartitionKey"`
	RowKey       string                 `json:"RowKey"`
	Properties   map[string]interface{} `json:"properties"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/table/{accountName}/{tableName}", func(r chi.Router) {
		r.Post("/", h.CreateTable)
		r.Get("/", h.QueryEntities)
		r.Delete("/", h.DeleteTable)
		r.Route("/{partitionKey}/{rowKey}", func(r chi.Router) {
			r.Put("/", h.UpsertEntity)
			r.Get("/", h.GetEntity)
			r.Delete("/", h.DeleteEntity)
		})
	})
}

func (h *Handler) key(account, table, pk, rk string) string {
	return "table:" + account + ":" + table + ":" + pk + ":" + rk
}

func (h *Handler) CreateTable(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) DeleteTable(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpsertEntity(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	table := chi.URLParam(r, "tableName")
	pk := chi.URLParam(r, "partitionKey")
	rk := chi.URLParam(r, "rowKey")
	
	var entity Entity
	json.NewDecoder(r.Body).Decode(&entity)
	entity.PartitionKey = pk
	entity.RowKey = rk
	
	h.store.Set(h.key(account, table, pk, rk), entity)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetEntity(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	table := chi.URLParam(r, "tableName")
	pk := chi.URLParam(r, "partitionKey")
	rk := chi.URLParam(r, "rowKey")
	
	v, ok := h.store.Get(h.key(account, table, pk, rk))
	if !ok {
		azerr.NotFound(w, "TableStorage/entity", rk)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteEntity(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	table := chi.URLParam(r, "tableName")
	pk := chi.URLParam(r, "partitionKey")
	rk := chi.URLParam(r, "rowKey")
	
	h.store.Delete(h.key(account, table, pk, rk))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) QueryEntities(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	table := chi.URLParam(r, "tableName")
	items := h.store.ListByPrefix("table:" + account + ":" + table + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
