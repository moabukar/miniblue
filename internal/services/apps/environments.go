package apps

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
)

func (h *Handler) managedEnvKey(sub, rg, name string) string {
	return "managedenv:" + sub + ":" + rg + ":" + name
}

func (h *Handler) RegisterManagedEnvironments(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.App/managedEnvironments", func(r chi.Router) {
		r.Get("/", h.ListManagedEnvironments)
		r.Route("/{name}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateManagedEnvironment)
			r.Get("/", h.GetManagedEnvironment)
			r.Delete("/", h.DeleteManagedEnvironment)
		})
	})
}

func (h *Handler) buildManagedEnvironmentResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.App/managedEnvironments/" + name

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	tags, _ := input["tags"].(map[string]interface{})
	if tags == nil {
		tags = map[string]interface{}{}
	}

	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.App/managedEnvironments",
		"location": location,
		"tags":     tags,
		"properties": map[string]interface{}{
			"provisioningState": "Succeeded",
			"deploymentErrors":  "",
			"externalEndpoints": []interface{}{},
			"customDomainConfiguration": map[string]interface{}{
				"customDomainVerificationId": "",
				"customDomainName":           "",
				"dnsSuffix":                  "",
			},
			"peerAuthentication": map[string]interface{}{
				"mtls": map[string]interface{}{
					"enabled": false,
				},
			},
			"vnetConfiguration":    props["vnetConfiguration"],
			"appLogsConfiguration": props["appLogsConfiguration"],
		},
	}
}

func (h *Handler) CreateOrUpdateManagedEnvironment(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	env := h.buildManagedEnvironmentResponse(sub, rg, name, input)
	k := h.managedEnvKey(sub, rg, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, env)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(env)
}

func (h *Handler) GetManagedEnvironment(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.managedEnvKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/managedEnvironments", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteManagedEnvironment(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	if !h.store.Delete(h.managedEnvKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.App/managedEnvironments", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListManagedEnvironments(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("managedenv:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
