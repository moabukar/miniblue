package sites

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
)

type ServerFarm struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Kind       string            `json:"kind,omitempty"`
	Properties ServerFarmProps   `json:"properties"`
	Tags       map[string]string `json:"tags,omitempty"`
}

type ServerFarmProps struct {
	ProvisioningState string `json:"provisioningState"`
	WorkerSize        string `json:"workerSize,omitempty"`
	WorkerSizeID      int    `json:"workerSizeId,omitempty"`
	NumberOfWorkers   int    `json:"numberOfWorkers,omitempty"`
	SkuName           string `json:"skuName,omitempty"`
	SkuTier           string `json:"skuTier,omitempty"`
	OSType            string `json:"ostype,omitempty"`
}

func (h *Handler) serverFarmKey(sub, rg, name string) string {
	return "serverfarm:" + sub + ":" + rg + ":" + name
}

func (h *Handler) RegisterServerFarms(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Web/serverFarms", func(r chi.Router) {
		r.Get("/", h.ListServerFarms)
		r.Route("/{name}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateServerFarm)
			r.Get("/", h.GetServerFarm)
			r.Delete("/", h.DeleteServerFarm)
		})
	})
}

func (h *Handler) buildServerFarmResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/serverFarms/" + name

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	kind, _ := input["kind"].(string)
	if kind == "" {
		kind = "linux"
	}

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	tags, _ := input["tags"].(map[string]interface{})
	if tags == nil {
		tags = map[string]interface{}{}
	}

	skuName, _ := props["skuName"].(string)
	skuTier := "Free"
	if skuName != "" {
		switch skuName[0] {
		case 'F', 'f':
			skuTier = "Free"
		case 'D', 'd':
			skuTier = "Shared"
		case 'B', 'b':
			skuTier = "Basic"
		case 'S', 's':
			skuTier = "Standard"
		case 'P', 'p':
			skuTier = "Premium"
		case 'I', 'i':
			skuTier = "Isolated"
		case 'Y', 'y':
			skuTier = "Dynamic"
		}
	}

	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.Web/serverFarms",
		"kind":     kind,
		"location": location,
		"tags":     tags,
		"properties": map[string]interface{}{
			"provisioningState": "Succeeded",
			"workerSize":        props["workerSize"],
			"workerSizeId":      props["workerSizeId"],
			"numberOfWorkers":   props["numberOfWorkers"],
			"reserved":          kind == "linux",
			"isXenon":           false,
			"hyperV":            false,
			"status":            "Ready",
			"subscription":      sub,
			"skuName":           skuName,
			"skuTier":           skuTier,
		},
	}
}

func (h *Handler) CreateOrUpdateServerFarm(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	sf := h.buildServerFarmResponse(sub, rg, name, input)
	k := h.serverFarmKey(sub, rg, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, sf)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(sf)
}

func (h *Handler) GetServerFarm(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.serverFarmKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/serverFarms", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteServerFarm(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	if !h.store.Delete(h.serverFarmKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Web/serverFarms", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListServerFarms(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("serverfarm:" + sub + ":" + rg)
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
