package functions

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

type FunctionApp struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Location   string `json:"location"`
	Kind       string `json:"kind"`
	Properties struct {
		ProvisioningState string `json:"provisioningState"`
		DefaultHostName   string `json:"defaultHostName"`
		State             string `json:"state"`
		Enabled           bool   `json:"enabled"`
	} `json:"properties"`
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

	var fn FunctionApp
	json.NewDecoder(r.Body).Decode(&fn)
	fn.ID = "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name
	fn.Name = name
	fn.Type = "Microsoft.Web/sites"
	fn.Kind = "functionapp"
	fn.Properties.ProvisioningState = "Succeeded"
	fn.Properties.DefaultHostName = name + ".azurewebsites.net"
	fn.Properties.State = "Running"
	fn.Properties.Enabled = true

	k := h.key(sub, rg, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, fn)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
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

	if !h.store.Delete(h.key(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("func:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
