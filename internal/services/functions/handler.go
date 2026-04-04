package functions

import (
	"encoding/json"
	"github.com/moabukar/local-azure/internal/azerr"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type Function struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Location string            `json:"location"`
	Properties map[string]string `json:"properties"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Web/sites", func(r chi.Router) {
		r.Get("/", h.List)
		r.Route("/{name}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdate)
			r.Get("/", h.Get)
			r.Delete("/", h.Delete)
		})
	})
}

func (h *Handler) key(sub, rg, name string) string {
	return "func:" + sub + ":" + rg + ":" + name
}

func (h *Handler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")
	
	var fn Function
	json.NewDecoder(r.Body).Decode(&fn)
	fn.ID = "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name
	fn.Name = name
	fn.Type = "Microsoft.Web/sites"
	
	h.store.Set(h.key(sub, rg, name), fn)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(fn)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")
	
	v, ok := h.store.Get(h.key(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")
	
	h.store.Delete(h.key(sub, rg, name))
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("func:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
