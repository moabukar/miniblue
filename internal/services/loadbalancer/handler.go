package loadbalancer

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/loadBalancers", func(r chi.Router) {
		r.Get("/", h.List)
		r.Route("/{loadBalancerName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdate)
			r.Get("/", h.Get)
			r.Delete("/", h.Delete)
		})
	})
}

func (h *Handler) key(sub, rg, name string) string {
	return "lb:" + sub + ":" + rg + ":" + name
}

func getArray(props map[string]interface{}, field string) []interface{} {
	if v, ok := props[field].([]interface{}); ok {
		return v
	}
	return []interface{}{}
}

func buildResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/loadBalancers/" + name

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	sku, _ := input["sku"].(map[string]interface{})
	if sku == nil {
		sku = map[string]interface{}{"name": "Standard", "tier": "Regional"}
	}

	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.Network/loadBalancers",
		"location": location,
		"etag":     "W/\"miniblue\"",
		"sku":      sku,
		"properties": map[string]interface{}{
			"provisioningState":        "Succeeded",
			"resourceGuid":            uuid.New().String(),
			"frontendIPConfigurations": getArray(props, "frontendIPConfigurations"),
			"backendAddressPools":      getArray(props, "backendAddressPools"),
			"loadBalancingRules":       getArray(props, "loadBalancingRules"),
			"probes":                   getArray(props, "probes"),
			"inboundNatRules":          getArray(props, "inboundNatRules"),
			"inboundNatPools":          getArray(props, "inboundNatPools"),
			"outboundRules":            getArray(props, "outboundRules"),
		},
	}
}

func (h *Handler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "loadBalancerName")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	lb := buildResponse(sub, rg, name, input)
	k := h.key(sub, rg, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, lb)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(lb)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "loadBalancerName")

	v, ok := h.store.Get(h.key(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/loadBalancers", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "loadBalancerName")

	if !h.store.Delete(h.key(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Network/loadBalancers", name)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("lb:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
