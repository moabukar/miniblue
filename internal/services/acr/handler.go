package acr

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/azerr"
	"github.com/moabukar/local-azure/internal/store"
)

type Registry struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Type       string       `json:"type"`
	Location   string       `json:"location"`
	SKU        RegistrySKU  `json:"sku"`
	Properties RegistryProps `json:"properties"`
}

type RegistrySKU struct {
	Name string `json:"name"`
	Tier string `json:"tier"`
}

type RegistryProps struct {
	LoginServer       string `json:"loginServer"`
	ProvisioningState string `json:"provisioningState"`
	AdminUserEnabled  bool   `json:"adminUserEnabled"`
	CreationDate      string `json:"creationDate,omitempty"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerRegistry/registries", func(r chi.Router) {
		r.Get("/", h.ListRegistries)
		r.Route("/{registryName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateRegistry)
			r.Get("/", h.GetRegistry)
			r.Delete("/", h.DeleteRegistry)
		})
	})
	r.Route("/acr/{registryName}/v2/{repository}/manifests", func(r chi.Router) {
		r.Get("/", h.ListManifests)
		r.Get("/{reference}", h.GetManifest)
	})
	r.Get("/acr/{registryName}/v2/{repository}/tags/list", h.ListTags)
}

func (h *Handler) registryKey(sub, rg, name string) string {
	return "acr:registry:" + sub + ":" + rg + ":" + name
}

func (h *Handler) CreateOrUpdateRegistry(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "registryName")

	var input struct {
		Location   string `json:"location"`
		SKU        *RegistrySKU `json:"sku,omitempty"`
		Properties *struct {
			AdminUserEnabled bool `json:"adminUserEnabled"`
		} `json:"properties,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	reg := Registry{
		ID:       "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.ContainerRegistry/registries/" + name,
		Name:     name,
		Type:     "Microsoft.ContainerRegistry/registries",
		Location: input.Location,
		SKU:      RegistrySKU{Name: "Basic", Tier: "Basic"},
	}
	if input.SKU != nil {
		reg.SKU = *input.SKU
	}
	reg.Properties.LoginServer = name + ".azurecr.io"
	reg.Properties.ProvisioningState = "Succeeded"
	if input.Properties != nil {
		reg.Properties.AdminUserEnabled = input.Properties.AdminUserEnabled
	}

	k := h.registryKey(sub, rg, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, reg)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(reg)
}

func (h *Handler) GetRegistry(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "registryName")

	v, ok := h.store.Get(h.registryKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.ContainerRegistry/registries", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteRegistry(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "registryName")
	if !h.store.Delete(h.registryKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.ContainerRegistry/registries", name)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListRegistries(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("acr:registry:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) ListManifests(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"manifests": []interface{}{}})
}

func (h *Handler) GetManifest(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tag":    chi.URLParam(r, "reference"),
		"digest": "sha256:abc123",
	})
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name": chi.URLParam(r, "repository"),
		"tags": []string{},
	})
}
