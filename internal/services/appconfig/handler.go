package appconfig

import (
	"encoding/json"
	"net/http"
	"github.com/moabukar/local-azure/internal/azerr"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type KeyValue struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	Label        string `json:"label,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
	LastModified string `json:"last_modified"`
	Etag         string `json:"etag"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/appconfig/{configStoreName}/kv", func(r chi.Router) {
		r.Get("/", h.ListKeyValues)
		r.Route("/{key}", func(r chi.Router) {
			r.Put("/", h.SetKeyValue)
			r.Get("/", h.GetKeyValue)
			r.Delete("/", h.DeleteKeyValue)
		})
	})
}

func (h *Handler) kvKey(store, key string) string {
	return "appconfig:" + store + ":" + key
}

func (h *Handler) SetKeyValue(w http.ResponseWriter, r *http.Request) {
	storeName := chi.URLParam(r, "configStoreName")
	key := chi.URLParam(r, "key")
	
	var kv KeyValue
	json.NewDecoder(r.Body).Decode(&kv)
	kv.Key = key
	kv.LastModified = time.Now().UTC().Format(time.RFC3339)
	kv.Etag = "etag-" + key
	
	h.store.Set(h.kvKey(storeName, key), kv)
	json.NewEncoder(w).Encode(kv)
}

func (h *Handler) GetKeyValue(w http.ResponseWriter, r *http.Request) {
	storeName := chi.URLParam(r, "configStoreName")
	key := chi.URLParam(r, "key")
	
	v, ok := h.store.Get(h.kvKey(storeName, key))
	if !ok {
		azerr.NotFound(w, "AppConfiguration/keyValues", key)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteKeyValue(w http.ResponseWriter, r *http.Request) {
	storeName := chi.URLParam(r, "configStoreName")
	key := chi.URLParam(r, "key")
	h.store.Delete(h.kvKey(storeName, key))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListKeyValues(w http.ResponseWriter, r *http.Request) {
	storeName := chi.URLParam(r, "configStoreName")
	items := h.store.ListByPrefix("appconfig:" + storeName + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"items": items})
}
