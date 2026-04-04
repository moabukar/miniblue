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
			r.Get("/replications", h.ListReplications)
		})
	})
	r.Post("/subscriptions/{subscriptionId}/providers/Microsoft.ContainerRegistry/checkNameAvailability", h.CheckNameAvailability)
	r.Route("/acr/{registryName}/v2/{repository}/manifests", func(r chi.Router) {
		r.Get("/", h.ListManifests)
		r.Get("/{reference}", h.GetManifest)
	})
	r.Get("/acr/{registryName}/v2/{repository}/tags/list", h.ListTags)
}

func (h *Handler) registryKey(sub, rg, name string) string {
	return "acr:registry:" + sub + ":" + rg + ":" + name
}

func buildRegistryResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.ContainerRegistry/registries/" + name

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}
	skuName := "Basic"
	if sku, ok := input["sku"].(map[string]interface{}); ok {
		if n, ok := sku["name"].(string); ok {
			skuName = n
		}
	}

	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.ContainerRegistry/registries",
		"location": location,
		"sku": map[string]interface{}{
			"name": skuName,
			"tier": skuName,
		},
		"properties": map[string]interface{}{
			"loginServer":              name + ".azurecr.io",
			"provisioningState":        "Succeeded",
			"adminUserEnabled":         false,
			"creationDate":             "2026-01-01T00:00:00Z",
			"publicNetworkAccess":      "Enabled",
			"zoneRedundancy":           "Disabled",
			"networkRuleBypassOptions": "AzureServices",
			"dataEndpointEnabled":      false,
			"encryption": map[string]interface{}{
				"status": "disabled",
			},
			"networkRuleSet": map[string]interface{}{
				"defaultAction": "Allow",
				"ipRules":       []interface{}{},
			},
			"policies": map[string]interface{}{
				"quarantinePolicy": map[string]interface{}{"status": "disabled"},
				"trustPolicy":     map[string]interface{}{"status": "disabled", "type": "Notary"},
				"retentionPolicy": map[string]interface{}{"status": "disabled", "days": 7},
				"exportPolicy":    map[string]interface{}{"status": "enabled"},
			},
		},
	}
}

func (h *Handler) CreateOrUpdateRegistry(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "registryName")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	reg := buildRegistryResponse(sub, rg, name, input)
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

func (h *Handler) ListReplications(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"value": []interface{}{}})
}

func (h *Handler) CheckNameAvailability(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "InvalidRequestBody",
				"message": "Could not parse request body.",
			},
		})
		return
	}

	// Check if any existing registry already uses this name
	sub := chi.URLParam(r, "subscriptionId")
	nameAvailable := true
	reason := ""
	message := ""

	items := h.store.ListByPrefix("acr:registry:" + sub + ":")
	for _, item := range items {
		if reg, ok := item.(Registry); ok {
			if reg.Name == input.Name {
				nameAvailable = false
				reason = "AlreadyExists"
				message = "The registry " + input.Name + " is already in use."
				break
			}
		}
	}

	resp := map[string]interface{}{
		"nameAvailable": nameAvailable,
	}
	if !nameAvailable {
		resp["reason"] = reason
		resp["message"] = message
	}
	json.NewEncoder(w).Encode(resp)
}
