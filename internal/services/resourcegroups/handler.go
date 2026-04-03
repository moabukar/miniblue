package resourcegroups

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type ResourceGroup struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Properties map[string]string `json:"properties"`
	Tags       map[string]string `json:"tags,omitempty"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourcegroups", func(r chi.Router) {
		r.Get("/", h.List)
		r.Route("/{resourceGroupName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdate)
			r.Get("/", h.Get)
			r.Delete("/", h.Delete)
		})
	})
}

func (h *Handler) key(sub, name string) string {
	return "rg:" + sub + ":" + name
}

func (h *Handler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")
	
	var rg ResourceGroup
	if err := json.NewDecoder(r.Body).Decode(&rg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	rg.ID = "/subscriptions/" + sub + "/resourceGroups/" + name
	rg.Name = name
	rg.Type = "Microsoft.Resources/resourceGroups"
	if rg.Properties == nil {
		rg.Properties = map[string]string{"provisioningState": "Succeeded"}
	}
	
	h.store.Set(h.key(sub, name), rg)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rg)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")
	
	v, ok := h.store.Get(h.key(sub, name))
	if !ok {
		http.Error(w, "ResourceGroupNotFound", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")
	
	if !h.store.Delete(h.key(sub, name)) {
		http.Error(w, "ResourceGroupNotFound", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	items := h.store.ListByPrefix("rg:" + sub + ":")
	result := map[string]interface{}{"value": items}
	json.NewEncoder(w).Encode(result)
}
