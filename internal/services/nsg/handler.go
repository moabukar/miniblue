package nsg

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
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkSecurityGroups", func(r chi.Router) {
		r.Get("/", h.ListNSGs)
		r.Route("/{nsgName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdate)
			r.Get("/", h.Get)
			r.Delete("/", h.Delete)
			r.Route("/securityRules", func(r chi.Router) {
				r.Get("/", h.ListRules)
				r.Route("/{ruleName}", func(r chi.Router) {
					r.Put("/", h.CreateOrUpdateRule)
					r.Get("/", h.GetRule)
					r.Delete("/", h.DeleteRule)
				})
			})
		})
	})
}

func (h *Handler) nsgKey(sub, rg, name string) string {
	return "nsg:" + sub + ":" + rg + ":" + name
}

func (h *Handler) ruleKey(sub, rg, nsg, rule string) string {
	return "nsgrule:" + sub + ":" + rg + ":" + nsg + ":" + rule
}

func defaultSecurityRules() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"name": "AllowVnetInBound",
			"properties": map[string]interface{}{
				"provisioningState":      "Succeeded",
				"priority":               float64(65000),
				"direction":              "Inbound",
				"access":                 "Allow",
				"protocol":               "*",
				"sourceAddressPrefix":    "VirtualNetwork",
				"destinationAddressPrefix": "VirtualNetwork",
				"sourcePortRange":        "*",
				"destinationPortRange":   "*",
			},
		},
		map[string]interface{}{
			"name": "AllowAzureLoadBalancerInBound",
			"properties": map[string]interface{}{
				"provisioningState":      "Succeeded",
				"priority":               float64(65001),
				"direction":              "Inbound",
				"access":                 "Allow",
				"protocol":               "*",
				"sourceAddressPrefix":    "AzureLoadBalancer",
				"destinationAddressPrefix": "*",
				"sourcePortRange":        "*",
				"destinationPortRange":   "*",
			},
		},
		map[string]interface{}{
			"name": "DenyAllInBound",
			"properties": map[string]interface{}{
				"provisioningState":      "Succeeded",
				"priority":               float64(65500),
				"direction":              "Inbound",
				"access":                 "Deny",
				"protocol":               "*",
				"sourceAddressPrefix":    "*",
				"destinationAddressPrefix": "*",
				"sourcePortRange":        "*",
				"destinationPortRange":   "*",
			},
		},
		map[string]interface{}{
			"name": "AllowVnetOutBound",
			"properties": map[string]interface{}{
				"provisioningState":      "Succeeded",
				"priority":               float64(65000),
				"direction":              "Outbound",
				"access":                 "Allow",
				"protocol":               "*",
				"sourceAddressPrefix":    "VirtualNetwork",
				"destinationAddressPrefix": "VirtualNetwork",
				"sourcePortRange":        "*",
				"destinationPortRange":   "*",
			},
		},
		map[string]interface{}{
			"name": "AllowInternetOutBound",
			"properties": map[string]interface{}{
				"provisioningState":      "Succeeded",
				"priority":               float64(65001),
				"direction":              "Outbound",
				"access":                 "Allow",
				"protocol":               "*",
				"sourceAddressPrefix":    "*",
				"destinationAddressPrefix": "Internet",
				"sourcePortRange":        "*",
				"destinationPortRange":   "*",
			},
		},
		map[string]interface{}{
			"name": "DenyAllOutBound",
			"properties": map[string]interface{}{
				"provisioningState":      "Succeeded",
				"priority":               float64(65500),
				"direction":              "Outbound",
				"access":                 "Deny",
				"protocol":               "*",
				"sourceAddressPrefix":    "*",
				"destinationAddressPrefix": "*",
				"sourcePortRange":        "*",
				"destinationPortRange":   "*",
			},
		},
	}
}

func buildNSGResponse(sub, rg, name string, input map[string]interface{}, securityRules []interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/networkSecurityGroups/" + name

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	if securityRules == nil {
		securityRules = []interface{}{}
	}

	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.Network/networkSecurityGroups",
		"location": location,
		"etag":     "W/\"miniblue\"",
		"properties": map[string]interface{}{
			"provisioningState":    "Succeeded",
			"resourceGuid":        uuid.New().String(),
			"securityRules":       securityRules,
			"defaultSecurityRules": defaultSecurityRules(),
			"subnets":             []interface{}{},
			"networkInterfaces":   []interface{}{},
		},
	}
}

func buildRuleResponse(sub, rg, nsgName, ruleName string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/networkSecurityGroups/" + nsgName + "/securityRules/" + ruleName

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	ruleProps := map[string]interface{}{
		"provisioningState": "Succeeded",
	}
	for _, field := range []string{
		"priority", "direction", "access", "protocol",
		"sourcePortRange", "destinationPortRange",
		"sourceAddressPrefix", "destinationAddressPrefix",
		"sourcePortRanges", "destinationPortRanges",
		"sourceAddressPrefixes", "destinationAddressPrefixes",
		"description",
	} {
		if v, ok := props[field]; ok {
			ruleProps[field] = v
		}
	}

	return map[string]interface{}{
		"id":         id,
		"name":       ruleName,
		"type":       "Microsoft.Network/networkSecurityGroups/securityRules",
		"properties": ruleProps,
	}
}

func (h *Handler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "nsgName")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	k := h.nsgKey(sub, rg, name)
	_, exists := h.store.Get(k)

	rules := h.store.ListByPrefix(h.ruleKey(sub, rg, name, ""))
	nsgResp := buildNSGResponse(sub, rg, name, input, rules)
	h.store.Set(k, nsgResp)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(nsgResp)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "nsgName")

	v, ok := h.store.Get(h.nsgKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/networkSecurityGroups", name)
		return
	}

	if nsg, ok := v.(map[string]interface{}); ok {
		ruleItems := h.store.ListByPrefix(h.ruleKey(sub, rg, name, ""))
		if props, ok := nsg["properties"].(map[string]interface{}); ok {
			if len(ruleItems) > 0 {
				props["securityRules"] = ruleItems
			} else {
				props["securityRules"] = []interface{}{}
			}
		}
		json.NewEncoder(w).Encode(nsg)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "nsgName")

	if !h.store.Delete(h.nsgKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Network/networkSecurityGroups", name)
		return
	}
	h.store.DeleteByPrefix(h.ruleKey(sub, rg, name, ""))
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListNSGs(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("nsg:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) CreateOrUpdateRule(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	nsgName := chi.URLParam(r, "nsgName")
	ruleName := chi.URLParam(r, "ruleName")

	if !h.store.Exists(h.nsgKey(sub, rg, nsgName)) {
		azerr.NotFound(w, "Microsoft.Network/networkSecurityGroups", nsgName)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	rule := buildRuleResponse(sub, rg, nsgName, ruleName, input)
	k := h.ruleKey(sub, rg, nsgName, ruleName)
	_, exists := h.store.Get(k)
	h.store.Set(k, rule)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(rule)
}

func (h *Handler) GetRule(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	nsgName := chi.URLParam(r, "nsgName")
	ruleName := chi.URLParam(r, "ruleName")

	v, ok := h.store.Get(h.ruleKey(sub, rg, nsgName, ruleName))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/networkSecurityGroups/securityRules", ruleName)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	nsgName := chi.URLParam(r, "nsgName")
	ruleName := chi.URLParam(r, "ruleName")

	if !h.store.Delete(h.ruleKey(sub, rg, nsgName, ruleName)) {
		azerr.NotFound(w, "Microsoft.Network/networkSecurityGroups/securityRules", ruleName)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	nsgName := chi.URLParam(r, "nsgName")
	items := h.store.ListByPrefix(h.ruleKey(sub, rg, nsgName, ""))
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
