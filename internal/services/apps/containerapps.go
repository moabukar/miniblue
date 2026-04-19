package apps

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
)

func sanitizeEmptyStrings(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			result[k] = sanitizeEmptyStrings(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = sanitizeEmptyStrings(v)
		}
		return result
	case string:
		if val == "" {
			return nil
		}
		return val
	default:
		return v
	}
}

func (h *Handler) containerAppKey(sub, rg, name string) string {
	return "containerapp:" + sub + ":" + rg + ":" + name
}

func (h *Handler) RegisterContainerApps(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.App/containerApps", func(r chi.Router) {
		r.Get("/", h.ListContainerApps)
		r.Route("/{name}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateContainerApp)
			r.Patch("/", h.UpdateContainerApp)
			r.Get("/", h.GetContainerApp)
			r.Delete("/", h.DeleteContainerApp)

			r.Post("/getAuthToken", h.GetAuthToken)
			r.Post("/analyzeCustomDomain", h.AnalyzeCustomDomain)

			r.Route("/revisions", func(r chi.Router) {
				r.Get("/", h.ListRevisions)
				r.Route("/{revisionName}", func(r chi.Router) {
					r.Get("/", h.GetRevision)
				})
			})

			r.Route("/listSecrets", func(r chi.Router) {
				r.Get("/", h.ListContainerAppSecrets)
				r.Post("/", h.CreateContainerAppSecrets)
			})

			r.Post("/start", h.StartContainerApp)
			r.Post("/stop", h.StopContainerApp)
		})
	})

	r.Route("/subscriptions/{subscriptionId}/providers/Microsoft.App/containerApps", func(r chi.Router) {
		r.Get("/", h.ListContainerAppsBySubscription)
	})
}

func (h *Handler) buildContainerAppResponse(sub, rg, name string, input map[string]any) map[string]any {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.App/containerApps/" + name

	props, _ := input["properties"].(map[string]any)
	if props == nil {
		props = map[string]any{}
	}

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	tags, _ := input["tags"].(map[string]any)
	if tags == nil {
		tags = map[string]any{}
	}

	environmentID := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.App/managedEnvironments/default"
	if envID, ok := props["managedEnvironmentId"].(string); ok && envID != "" {
		environmentID = envID
	}

	provisioningState := "Succeeded"
	if state, ok := props["provisioningState"].(string); ok && state != "" {
		provisioningState = state
	}

	fqdn := name + ".hashed.eastus.containerApps.k4apps.io"
	if inputFqdn, ok := props["fqdn"].(string); ok && inputFqdn != "" {
		fqdn = inputFqdn
	}

	response := map[string]any{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.App/containerApps",
		"location": location,
		"tags":     tags,
		"properties": map[string]any{
			"provisioningState":          provisioningState,
			"managedEnvironmentId":       environmentID,
			"latestRevisionName":         name + "-v1",
			"latestRevisionFqdn":         fqdn,
			"fqdn":                       fqdn,
			"exposedPort":                props["exposedPort"],
			"customDomainVerificationId": "miniblue",
			"configuration":              props["configuration"],
			"template":                   props["template"],
			"managedBy":                  props["managedBy"],
			"workloadProfileName":        props["workloadProfileName"],
		},
	}

	return sanitizeEmptyStrings(response).(map[string]any)
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

func (h *Handler) UpdateContainerApp(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.containerAppKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/containerApps", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	app := v.(map[string]interface{})

	if tags, ok := input["tags"].(map[string]interface{}); ok {
		app["tags"] = tags
	}

	if props, ok := input["properties"].(map[string]interface{}); ok {
		existingProps, _ := app["properties"].(map[string]interface{})
		for k, v := range props {
			if v != nil {
				existingProps[k] = v
			}
		}
		app["properties"] = existingProps
	}

	h.store.Set(h.containerAppKey(sub, rg, name), app)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(app)
}

func (h *Handler) GetAuthToken(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	_, ok := h.store.Get(h.containerAppKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/containerApps", name)
		return
	}

	token := map[string]interface{}{
		"token":            "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.miniblue-token",
		"expiresOn":        "2027-01-01T00:00:00Z",
		"containerAppName": name,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(token)
}

func (h *Handler) AnalyzeCustomDomain(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	_, ok := h.store.Get(h.containerAppKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/containerApps", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	hostname, _ := input["hostname"].(string)
	if hostname == "" {
		hostname = name + ".custom.example.com"
	}

	analysis := map[string]interface{}{
		"hostname":                       hostname,
		"isApiHostname":                  true,
		"isSslEnabled":                   true,
		"customDomainVerificationResult": "Verified",
		"certificateNotAfter":            "2027-01-01T00:00:00Z",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(analysis)
}

func (h *Handler) ListContainerAppSecrets(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	_, ok := h.store.Get(h.containerAppKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/containerApps", name)
		return
	}

	app, _ := h.store.Get(h.containerAppKey(sub, rg, name))
	props, _ := app.(map[string]any)["properties"].(map[string]any)
	config, _ := props["configuration"].(map[string]any)

	var secrets []map[string]any
	if secretsVal, ok := config["secrets"].([]any); ok {
		for _, s := range secretsVal {
			if secret, ok := s.(map[string]any); ok {
				secrets = append(secrets, map[string]any{
					"name":  secret["name"],
					"value": "***",
					"type":  secret["type"],
				})
			}
		}
	}

	if secrets == nil {
		secrets = []map[string]any{}
	}

	json.NewEncoder(w).Encode(map[string]any{"value": secrets})
}

func (h *Handler) ListContainerAppsBySubscription(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	items := h.store.ListByPrefix("containerapp:" + sub + ":")
	json.NewEncoder(w).Encode(map[string]any{"value": items})
}

func (h *Handler) CreateContainerAppSecrets(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	_, ok := h.store.Get(h.containerAppKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/containerApps", name)
		return
	}

	var input struct {
		Properties struct {
			Secrets []map[string]any `json:"secrets"`
		} `json:"properties"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	secrets := input.Properties.Secrets
	if secrets == nil {
		secrets = []map[string]any{}
	}

	app, _ := h.store.Get(h.containerAppKey(sub, rg, name))
	props, _ := app.(map[string]any)["properties"].(map[string]any)
	config, _ := props["configuration"].(map[string]any)

	if config == nil {
		config = map[string]any{}
	}

	config["secrets"] = secrets
	props["configuration"] = config
	app.(map[string]any)["properties"] = props
	h.store.Set(h.containerAppKey(sub, rg, name), app)

	var response []map[string]any
	for _, s := range secrets {
		response = append(response, map[string]any{
			"name":  s["name"],
			"value": "***",
			"type":  s["type"],
		})
	}

	json.NewEncoder(w).Encode(map[string]any{"value": response})
}
