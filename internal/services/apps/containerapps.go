package apps

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
)

func (h *Handler) containerAppKey(sub, rg, name string) string {
	return "containerapp:" + sub + ":" + rg + ":" + name
}

func (h *Handler) RegisterContainerApps(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.App/containerApps", func(r chi.Router) {
		r.Get("/", h.ListContainerApps)
		r.Route("/{name}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateContainerApp)
			r.Get("/", h.GetContainerApp)
			r.Delete("/", h.DeleteContainerApp)

			r.Route("/revisions", func(r chi.Router) {
				r.Get("/", h.ListRevisions)
				r.Route("/{revisionName}", func(r chi.Router) {
					r.Get("/", h.GetRevision)
				})
			})

			r.Post("/start", h.StartContainerApp)
			r.Post("/stop", h.StopContainerApp)
		})
	})
}

func (h *Handler) buildContainerAppResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.App/containerApps/" + name

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

	environmentID := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.App/managedEnvironments/default"
	if envID, ok := props["environmentId"].(string); ok && envID != "" {
		environmentID = envID
	}

	provisioningState := "Succeeded"
	if state, ok := props["provisioningState"].(string); ok && state != "" {
		provisioningState = state
	}

	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.App/containerApps",
		"location": location,
		"tags":     tags,
		"properties": map[string]interface{}{
			"provisioningState":          provisioningState,
			"environmentId":              environmentID,
			"environmentName":            "default",
			"latestRevisionName":         name + "-v1",
			"latestRevisionFqdn":         name + ".hashed.eastus.containerApps.k4apps.io",
			"fqdn":                       name + ".hashed.eastus.containerApps.k4apps.io",
			"exposedPort":                props["exposedPort"],
			"customDomainVerificationId": "miniblue",
			"configuration":              props["configuration"],
			"template":                   props["template"],
			"managedBy":                  props["managedBy"],
			"workloadProfileName":        props["workloadProfileName"],
		},
	}
}

func (h *Handler) CreateOrUpdateContainerApp(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	app := h.buildContainerAppResponse(sub, rg, name, input)
	k := h.containerAppKey(sub, rg, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, app)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(app)
}

func (h *Handler) GetContainerApp(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.containerAppKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/containerApps", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteContainerApp(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	if !h.store.Delete(h.containerAppKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.App/containerApps", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListContainerApps(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("containerapp:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) ListRevisions(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.containerAppKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/containerApps/revisions", name)
		return
	}

	app, _ := v.(map[string]interface{})
	props, _ := app["properties"].(map[string]interface{})
	revisionName, _ := props["latestRevisionName"].(string)
	revisionFqdn, _ := props["latestRevisionFqdn"].(string)

	revision := map[string]interface{}{
		"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.App/containerApps/" + name + "/revisions/" + revisionName,
		"name": revisionName,
		"type": "Microsoft.App/containerApps/revisions",
		"properties": map[string]interface{}{
			"createdTime":    "2024-01-01T00:00:00Z",
			"lastActiveTime": "2024-01-01T00:00:00Z",
			"fqdn":           revisionFqdn,
			"active":         true,
			"replicas":       1,
			"runningState":   "Running",
		},
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"value": []interface{}{revision}})
}

func (h *Handler) GetRevision(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")
	revisionName := chi.URLParam(r, "revisionName")

	v, ok := h.store.Get(h.containerAppKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/containerApps/revisions", revisionName)
		return
	}

	app, _ := v.(map[string]interface{})
	props, _ := app["properties"].(map[string]interface{})
	revisionFqdn, _ := props["latestRevisionFqdn"].(string)

	revision := map[string]interface{}{
		"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.App/containerApps/" + name + "/revisions/" + revisionName,
		"name": revisionName,
		"type": "Microsoft.App/containerApps/revisions",
		"properties": map[string]interface{}{
			"createdTime":    "2024-01-01T00:00:00Z",
			"lastActiveTime": "2024-01-01T00:00:00Z",
			"fqdn":           revisionFqdn,
			"active":         true,
			"replicas":       1,
			"runningState":   "Running",
		},
	}

	json.NewEncoder(w).Encode(revision)
}

func (h *Handler) StartContainerApp(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.containerAppKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/containerApps", name)
		return
	}

	app, ok := v.(map[string]interface{})
	if !ok {
		azerr.NotFound(w, "Microsoft.App/containerApps", name)
		return
	}

	props, _ := app["properties"].(map[string]interface{})
	props["provisioningState"] = "Succeeded"
	app["properties"] = props
	h.store.Set(h.containerAppKey(sub, rg, name), app)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(app)
}

func (h *Handler) StopContainerApp(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.containerAppKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/containerApps", name)
		return
	}

	app, ok := v.(map[string]interface{})
	if !ok {
		azerr.NotFound(w, "Microsoft.App/containerApps", name)
		return
	}

	props, _ := app["properties"].(map[string]interface{})
	props["provisioningState"] = "Degraded"
	app["properties"] = props
	h.store.Set(h.containerAppKey(sub, rg, name), app)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(app)
}
