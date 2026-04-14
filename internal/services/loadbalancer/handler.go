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

			r.Route("/backendAddressPools", func(r chi.Router) {
				r.Get("/", h.ListBackendAddressPools)
				r.Route("/{backendAddressPoolName}", func(r chi.Router) {
					r.Put("/", h.CreateOrUpdateBackendAddressPool)
					r.Get("/", h.GetBackendAddressPool)
					r.Delete("/", h.DeleteBackendAddressPool)
				})
			})

			r.Route("/probes", func(r chi.Router) {
				r.Get("/", h.ListProbes)
				r.Route("/{probeName}", func(r chi.Router) {
					r.Put("/", h.CreateOrUpdateProbe)
					r.Get("/", h.GetProbe)
					r.Delete("/", h.DeleteProbe)
				})
			})

			r.Route("/loadBalancingRules", func(r chi.Router) {
				r.Get("/", h.ListLoadBalancingRules)
				r.Route("/{ruleName}", func(r chi.Router) {
					r.Put("/", h.CreateOrUpdateLoadBalancingRule)
					r.Get("/", h.GetLoadBalancingRule)
					r.Delete("/", h.DeleteLoadBalancingRule)
				})
			})
		})
	})
}

func (h *Handler) key(sub, rg, name string) string {
	return "lb:" + sub + ":" + rg + ":" + name
}

func (h *Handler) backendPoolKey(sub, rg, lb, pool string) string {
	return "lb:bap:" + sub + ":" + rg + ":" + lb + ":" + pool
}

func (h *Handler) probeKey(sub, rg, lb, probe string) string {
	return "lb:probe:" + sub + ":" + rg + ":" + lb + ":" + probe
}

func (h *Handler) lbRuleKey(sub, rg, lb, rule string) string {
	return "lb:rule:" + sub + ":" + rg + ":" + lb + ":" + rule
}

func getArray(props map[string]interface{}, field string) []interface{} {
	if v, ok := props[field].([]interface{}); ok {
		return v
	}
	return []interface{}{}
}

// func getMap(props map[string]interface{}, field string) map[string]interface{} {
// 	if v, ok := props[field].(map[string]interface{}); ok {
// 		return v
// 	}
// 	return nil
// }

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
			"resourceGuid":             uuid.New().String(),
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

func buildBackendAddressPoolResponse(sub, rg, lb, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/loadBalancers/" + lb + "/backendAddressPools/" + name

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	backendAddresses, _ := props["backendAddresses"].([]interface{})
	if backendAddresses == nil {
		backendAddresses = []interface{}{}
	}

	return map[string]interface{}{
		"id":   id,
		"name": name,
		"type": "Microsoft.Network/loadBalancers/backendAddressPools",
		"etag": "W/\"miniblue\"",
		"properties": map[string]interface{}{
			"provisioningState":            "Succeeded",
			"resourceGuid":                 uuid.New().String(),
			"loadBalancerBackendAddresses": backendAddresses,
		},
	}
}

func buildProbeResponse(sub, rg, lb, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/loadBalancers/" + lb + "/probes/" + name

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	protocol, _ := props["protocol"].(string)
	if protocol == "" {
		protocol = "Tcp"
	}

	port, _ := props["port"].(float64)
	if port == 0 {
		port = 80
	}

	return map[string]interface{}{
		"id":   id,
		"name": name,
		"type": "Microsoft.Network/loadBalancers/probes",
		"etag": "W/\"miniblue\"",
		"properties": map[string]interface{}{
			"provisioningState": "Succeeded",
			"resourceGuid":      uuid.New().String(),
			"protocol":          protocol,
			"port":              port,
			"intervalInSeconds": 15,
			"numberOfProbes":    4,
		},
	}
}

func buildLoadBalancingRuleResponse(sub, rg, lb, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/loadBalancers/" + lb + "/loadBalancingRules/" + name

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	return map[string]interface{}{
		"id":   id,
		"name": name,
		"type": "Microsoft.Network/loadBalancers/loadBalancingRules",
		"etag": "W/\"miniblue\"",
		"properties": map[string]interface{}{
			"provisioningState": "Succeeded",
			"resourceGuid":      uuid.New().String(),
			"protocol":          props["protocol"],
			"frontendPort":      props["frontendPort"],
			"backendPort":       props["backendPort"],
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

func (h *Handler) ListBackendAddressPools(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	lb := chi.URLParam(r, "loadBalancerName")
	items := h.store.ListByPrefix("lb:bap:" + sub + ":" + rg + ":" + lb + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) CreateOrUpdateBackendAddressPool(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	lb := chi.URLParam(r, "loadBalancerName")
	name := chi.URLParam(r, "backendAddressPoolName")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	pool := buildBackendAddressPoolResponse(sub, rg, lb, name, input)
	k := h.backendPoolKey(sub, rg, lb, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, pool)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(pool)
}

func (h *Handler) GetBackendAddressPool(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	lb := chi.URLParam(r, "loadBalancerName")
	name := chi.URLParam(r, "backendAddressPoolName")

	v, ok := h.store.Get(h.backendPoolKey(sub, rg, lb, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/loadBalancers/backendAddressPools", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteBackendAddressPool(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	lb := chi.URLParam(r, "loadBalancerName")
	name := chi.URLParam(r, "backendAddressPoolName")

	if !h.store.Delete(h.backendPoolKey(sub, rg, lb, name)) {
		azerr.NotFound(w, "Microsoft.Network/loadBalancers/backendAddressPools", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListProbes(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	lb := chi.URLParam(r, "loadBalancerName")
	items := h.store.ListByPrefix("lb:probe:" + sub + ":" + rg + ":" + lb + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) CreateOrUpdateProbe(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	lb := chi.URLParam(r, "loadBalancerName")
	name := chi.URLParam(r, "probeName")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	probe := buildProbeResponse(sub, rg, lb, name, input)
	k := h.probeKey(sub, rg, lb, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, probe)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(probe)
}

func (h *Handler) GetProbe(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	lb := chi.URLParam(r, "loadBalancerName")
	name := chi.URLParam(r, "probeName")

	v, ok := h.store.Get(h.probeKey(sub, rg, lb, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/loadBalancers/probes", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteProbe(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	lb := chi.URLParam(r, "loadBalancerName")
	name := chi.URLParam(r, "probeName")

	if !h.store.Delete(h.probeKey(sub, rg, lb, name)) {
		azerr.NotFound(w, "Microsoft.Network/loadBalancers/probes", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListLoadBalancingRules(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	lb := chi.URLParam(r, "loadBalancerName")
	items := h.store.ListByPrefix("lb:rule:" + sub + ":" + rg + ":" + lb + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) CreateOrUpdateLoadBalancingRule(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	lb := chi.URLParam(r, "loadBalancerName")
	name := chi.URLParam(r, "ruleName")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	rule := buildLoadBalancingRuleResponse(sub, rg, lb, name, input)
	k := h.lbRuleKey(sub, rg, lb, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, rule)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(rule)
}

func (h *Handler) GetLoadBalancingRule(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	lb := chi.URLParam(r, "loadBalancerName")
	name := chi.URLParam(r, "ruleName")

	v, ok := h.store.Get(h.lbRuleKey(sub, rg, lb, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/loadBalancers/loadBalancingRules", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteLoadBalancingRule(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	lb := chi.URLParam(r, "loadBalancerName")
	name := chi.URLParam(r, "ruleName")

	if !h.store.Delete(h.lbRuleKey(sub, rg, lb, name)) {
		azerr.NotFound(w, "Microsoft.Network/loadBalancers/loadBalancingRules", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}
