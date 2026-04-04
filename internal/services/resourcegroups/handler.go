package resourcegroups

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
			r.Patch("/", h.Update)
			r.Get("/", h.Get)
			r.Delete("/", h.Delete)
			r.Head("/", h.CheckExistence)
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
	if input.Location == "" {
		azerr.BadRequest(w, "The location property is required for resource group creation.")
		return
	}

	k := h.key(sub, name)
	_, exists := h.store.Get(k)

	rg := ResourceGroup{
		ID:       "/subscriptions/" + sub + "/resourceGroups/" + name,
		Name:     name,
		Type:     "Microsoft.Resources/resourceGroups",
		Location: input.Location,
		Tags:     input.Tags,
	}
	rg.Properties.ProvisioningState = "Succeeded"

	h.store.Set(k, rg)

	if exists {
		// Azure returns 200 for update of existing resource group
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(rg)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")

	k := h.key(sub, name)
	v, ok := h.store.Get(k)
	if !ok {
		azerr.NotFound(w, "Microsoft.Resources/resourceGroups", name)
		return
	}

	rg := v.(ResourceGroup)
	var patch struct {
		Tags map[string]string `json:"tags,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&patch)
	if patch.Tags != nil {
		rg.Tags = patch.Tags
	}

	h.store.Set(k, rg)
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

func (h *Handler) CheckExistence(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")

	if h.store.Exists(h.key(sub, name)) {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")

	k := h.key(sub, name)
	if !h.store.Delete(k) {
		azerr.NotFound(w, "Microsoft.Resources/resourceGroups", name)
		return
	}

	// Also clean up resources in this resource group
	h.store.DeleteByPrefix("vnet:" + sub + ":" + name + ":")
	h.store.DeleteByPrefix("subnet:" + sub + ":" + name + ":")
	h.store.DeleteByPrefix("func:" + sub + ":" + name + ":")
	h.store.DeleteByPrefix("acr:registry:" + sub + ":" + name + ":")
	h.store.DeleteByPrefix("eg:topic:" + sub + ":" + name + ":")
	h.store.DeleteByPrefix("dns:zone:" + sub + ":" + name + ":")

	_ = fmt.Sprint(time.Now()) // used for async operation simulation
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	items := h.store.ListByPrefix("rg:" + sub + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
