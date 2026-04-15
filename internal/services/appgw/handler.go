package appgw

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
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/applicationGateways", func(r chi.Router) {
		r.Get("/", h.List)
		r.Route("/{applicationGatewayName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdate)
			r.Get("/", h.Get)
			r.Delete("/", h.Delete)
		})
	})
}

func (h *Handler) key(sub, rg, name string) string {
	return "appgw:" + sub + ":" + rg + ":" + name
}

func getArray(props map[string]interface{}, field string) []interface{} {
	if v, ok := props[field].([]interface{}); ok {
		return v
	}
	return []interface{}{}
}

func getMap(props map[string]interface{}, field string) map[string]interface{} {
	if v, ok := props[field].(map[string]interface{}); ok {
		return v
	}
	return nil
}

func buildResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/applicationGateways/" + name

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	sku := getMap(props, "sku")
	if sku == nil {
		sku = map[string]interface{}{
			"name":     "Standard_v2",
			"tier":     "Standard_v2",
			"capacity": float64(2),
		}
	}

	tags, _ := input["tags"].(map[string]interface{})
	if tags == nil {
		tags = map[string]interface{}{}
	}

	result := map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.Network/applicationGateways",
		"location": location,
		"tags":     tags,
		"etag":     "W/\"miniblue\"",
		"properties": map[string]interface{}{
			"provisioningState":            "Succeeded",
			"resourceGuid":                uuid.New().String(),
			"operationalState":            "Running",
			"sku":                          sku,
			"gatewayIPConfigurations":      getArray(props, "gatewayIPConfigurations"),
			"sslCertificates":             getArray(props, "sslCertificates"),
			"trustedRootCertificates":     getArray(props, "trustedRootCertificates"),
			"frontendIPConfigurations":     getArray(props, "frontendIPConfigurations"),
			"frontendPorts":                getArray(props, "frontendPorts"),
			"backendAddressPools":          getArray(props, "backendAddressPools"),
			"backendHttpSettingsCollection": getArray(props, "backendHttpSettingsCollection"),
			"httpListeners":                getArray(props, "httpListeners"),
			"urlPathMaps":                  getArray(props, "urlPathMaps"),
			"requestRoutingRules":          getArray(props, "requestRoutingRules"),
			"redirectConfigurations":       getArray(props, "redirectConfigurations"),
			"probes":                       getArray(props, "probes"),
			"rewriteRuleSets":             getArray(props, "rewriteRuleSets"),
			"autoscaleConfiguration":       getMap(props, "autoscaleConfiguration"),
			"webApplicationFirewallConfiguration": getMap(props, "webApplicationFirewallConfiguration"),
		},
	}

	return result
}

func (h *Handler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "applicationGatewayName")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	gw := buildResponse(sub, rg, name, input)
	k := h.key(sub, rg, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, gw)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(gw)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "applicationGatewayName")

	v, ok := h.store.Get(h.key(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/applicationGateways", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "applicationGatewayName")

	if !h.store.Delete(h.key(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Network/applicationGateways", name)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("appgw:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
