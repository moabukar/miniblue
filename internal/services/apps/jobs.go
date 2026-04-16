package apps

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
)

func (h *Handler) jobKey(sub, rg, name string) string {
	return "containerjob:" + sub + ":" + rg + ":" + name
}

func (h *Handler) RegisterJobs(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.App/jobs", func(r chi.Router) {
		r.Get("/", h.ListJobs)
		r.Route("/{name}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateJob)
			r.Get("/", h.GetJob)
			r.Delete("/", h.DeleteJob)
			r.Post("/start", h.StartJob)
			r.Post("/stop", h.StopJob)
		})
	})
}

func (h *Handler) buildJobResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.App/jobs/" + name

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

	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.App/jobs",
		"location": location,
		"tags":     tags,
		"properties": map[string]interface{}{
			"provisioningState":   "Succeeded",
			"environmentId":       environmentID,
			"configuration":       props["configuration"],
			"template":            props["template"],
			"workloadProfileName": props["workloadProfileName"],
		},
	}
}

func (h *Handler) CreateOrUpdateJob(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	job := h.buildJobResponse(sub, rg, name, input)
	k := h.jobKey(sub, rg, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, job)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(job)
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.jobKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/jobs", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	if !h.store.Delete(h.jobKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.App/jobs", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("containerjob:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) StartJob(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.jobKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/jobs", name)
		return
	}

	job, ok := v.(map[string]interface{})
	if !ok {
		azerr.NotFound(w, "Microsoft.App/jobs", name)
		return
	}

	props, _ := job["properties"].(map[string]interface{})
	props["provisioningState"] = "Running"
	job["properties"] = props
	h.store.Set(h.jobKey(sub, rg, name), job)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(job)
}

func (h *Handler) StopJob(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.jobKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.App/jobs", name)
		return
	}

	job, ok := v.(map[string]interface{})
	if !ok {
		azerr.NotFound(w, "Microsoft.App/jobs", name)
		return
	}

	props, _ := job["properties"].(map[string]interface{})
	props["provisioningState"] = "Stopped"
	job["properties"] = props
	h.store.Set(h.jobKey(sub, rg, name), job)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(job)
}
