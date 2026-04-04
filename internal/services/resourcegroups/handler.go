package resourcegroups

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/azerr"
	"github.com/moabukar/local-azure/internal/store"
)

type ResourceGroup struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties struct {
		ProvisioningState string `json:"provisioningState"`
	} `json:"properties"`
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

	var input struct {
		Location string            `json:"location"`
		Tags     map[string]string `json:"tags,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		azerr.BadRequest(w, "The request content was invalid: "+err.Error())
		return
	}

	rg := ResourceGroup{
		ID:       "/subscriptions/" + sub + "/resourceGroups/" + name,
		Name:     name,
		Type:     "Microsoft.Resources/resourceGroups",
		Location: input.Location,
		Tags:     input.Tags,
	}
	rg.Properties.ProvisioningState = "Succeeded"

	h.store.Set(h.key(sub, name), rg)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rg)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")

	v, ok := h.store.Get(h.key(sub, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Resources/resourceGroups", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")

	if !h.store.Delete(h.key(sub, name)) {
		azerr.NotFound(w, "Microsoft.Resources/resourceGroups", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	items := h.store.ListByPrefix("rg:" + sub + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
