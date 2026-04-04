package network

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/azerr"
	"github.com/moabukar/local-azure/internal/store"
)

type VNet struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Location   string     `json:"location"`
	Properties VNetProps  `json:"properties"`
}

type VNetProps struct {
	ProvisioningState string `json:"provisioningState"`
	AddressSpace      struct {
		AddressPrefixes []string `json:"addressPrefixes"`
	} `json:"addressSpace"`
	Subnets []SubnetRef `json:"subnets,omitempty"`
}

type SubnetRef struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Properties SubnetProps `json:"properties"`
}

type SubnetProps struct {
	ProvisioningState              string              `json:"provisioningState"`
	AddressPrefix                  string              `json:"addressPrefix"`
	AddressPrefixes                []string            `json:"addressPrefixes"`
	NetworkSecurityGroup           *SubResourceRef     `json:"networkSecurityGroup,omitempty"`
	RouteTable                     *SubResourceRef     `json:"routeTable,omitempty"`
	ServiceEndpoints               []interface{}       `json:"serviceEndpoints"`
	Delegations                    []interface{}       `json:"delegations"`
	PrivateEndpointNetworkPolicies string              `json:"privateEndpointNetworkPolicies"`
	PrivateLinkServiceNetworkPolicies string           `json:"privateLinkServiceNetworkPolicies"`
}

type SubResourceRef struct {
	ID string `json:"id"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks", func(r chi.Router) {
		r.Get("/", h.ListVNets)
		r.Route("/{vnetName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateVNet)
			r.Get("/", h.GetVNet)
			r.Delete("/", h.DeleteVNet)
			r.Route("/subnets", func(r chi.Router) {
				r.Get("/", h.ListSubnets)
				r.Route("/{subnetName}", func(r chi.Router) {
					r.Put("/", h.CreateOrUpdateSubnet)
					r.Get("/", h.GetSubnet)
					r.Delete("/", h.DeleteSubnet)
				})
			})
		})
	})
}

func (h *Handler) vnetKey(sub, rg, name string) string {
	return "vnet:" + sub + ":" + rg + ":" + name
}

func (h *Handler) subnetKey(sub, rg, vnet, subnet string) string {
	return "subnet:" + sub + ":" + rg + ":" + vnet + ":" + subnet
}

func buildVNetResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/virtualNetworks/" + name

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}
	addrSpace, _ := props["addressSpace"].(map[string]interface{})
	if addrSpace == nil {
		addrSpace = map[string]interface{}{"addressPrefixes": []interface{}{}}
	}

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.Network/virtualNetworks",
		"location": location,
		"etag":     "W/\"local-azure\"",
		"properties": map[string]interface{}{
			"provisioningState":  "Succeeded",
			"resourceGuid":      "00000000-0000-0000-0000-000000000000",
			"addressSpace":      addrSpace,
			"dhcpOptions":       map[string]interface{}{"dnsServers": []interface{}{}},
			"subnets":           []interface{}{},
			"virtualNetworkPeerings": []interface{}{},
			"enableDdosProtection":   false,
			"enableVmProtection":     false,
		},
	}
}

func (h *Handler) CreateOrUpdateVNet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "vnetName")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	vnet := buildVNetResponse(sub, rg, name, input)
	k := h.vnetKey(sub, rg, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, vnet)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(vnet)
}

func (h *Handler) GetVNet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "vnetName")

	v, ok := h.store.Get(h.vnetKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/virtualNetworks", name)
		return
	}

	// Re-populate subnets from the store
	if vnet, ok := v.(map[string]interface{}); ok {
		subnetItems := h.store.ListByPrefix(h.subnetKey(sub, rg, name, ""))
		if props, ok := vnet["properties"].(map[string]interface{}); ok {
			if len(subnetItems) > 0 {
				props["subnets"] = subnetItems
			} else {
				props["subnets"] = []interface{}{}
			}
		}
		json.NewEncoder(w).Encode(vnet)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteVNet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "vnetName")
	if !h.store.Delete(h.vnetKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Network/virtualNetworks", name)
		return
	}
	// Clean up subnets
	h.store.DeleteByPrefix(h.subnetKey(sub, rg, name, ""))
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListVNets(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("vnet:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// buildSubnetResponse builds a raw map matching the Azure 2023-11-01 subnet schema exactly.
// Using a map instead of a struct gives us full control over JSON field names and nil handling.
func buildSubnetResponse(sub, rg, vnetName, subnetName string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetName

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	// Extract addressPrefix(es)
	prefix, _ := props["addressPrefix"].(string)
	prefixes, _ := props["addressPrefixes"].([]interface{})
	if prefix == "" && len(prefixes) > 0 {
		prefix, _ = prefixes[0].(string)
	}
	if len(prefixes) == 0 && prefix != "" {
		prefixes = []interface{}{prefix}
	}

	return map[string]interface{}{
		"id":   id,
		"name": subnetName,
		"etag": "W/\"local-azure\"",
		"type": "Microsoft.Network/virtualNetworks/subnets",
		"properties": map[string]interface{}{
			"provisioningState":                 "Succeeded",
			"addressPrefix":                     prefix,
			"addressPrefixes":                   prefixes,
			"serviceEndpoints":                  []interface{}{},
			"serviceEndpointPolicies":           []interface{}{},
			"ipConfigurations":                  []interface{}{},
			"delegations":                       []interface{}{},
			"privateEndpointNetworkPolicies":    "Disabled",
			"privateLinkServiceNetworkPolicies": "Enabled",
			"defaultOutboundAccess":             true,
		},
	}
}

func (h *Handler) CreateOrUpdateSubnet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vnetName := chi.URLParam(r, "vnetName")
	subnetName := chi.URLParam(r, "subnetName")

	if !h.store.Exists(h.vnetKey(sub, rg, vnetName)) {
		azerr.NotFound(w, "Microsoft.Network/virtualNetworks", vnetName)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	subnet := buildSubnetResponse(sub, rg, vnetName, subnetName, input)
	k := h.subnetKey(sub, rg, vnetName, subnetName)
	_, exists := h.store.Get(k)
	h.store.Set(k, subnet)
	h.updateVNetSubnets(sub, rg, vnetName)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(subnet)
}

func (h *Handler) GetSubnet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vnetName := chi.URLParam(r, "vnetName")
	subnetName := chi.URLParam(r, "subnetName")

	v, ok := h.store.Get(h.subnetKey(sub, rg, vnetName, subnetName))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/virtualNetworks/subnets", subnetName)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteSubnet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vnetName := chi.URLParam(r, "vnetName")
	subnetName := chi.URLParam(r, "subnetName")

	if !h.store.Delete(h.subnetKey(sub, rg, vnetName, subnetName)) {
		azerr.NotFound(w, "Microsoft.Network/virtualNetworks/subnets", subnetName)
		return
	}
	// Update the parent VNet's subnets array
	h.updateVNetSubnets(sub, rg, vnetName)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListSubnets(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vnetName := chi.URLParam(r, "vnetName")
	items := h.store.ListByPrefix(h.subnetKey(sub, rg, vnetName, ""))
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// updateVNetSubnets refreshes the subnets array stored inside the parent VNet.
func (h *Handler) updateVNetSubnets(sub, rg, vnetName string) {
	vk := h.vnetKey(sub, rg, vnetName)
	v, ok := h.store.Get(vk)
	if !ok {
		return
	}
	vnet, ok := v.(map[string]interface{})
	if !ok {
		return
	}
	subnetItems := h.store.ListByPrefix(h.subnetKey(sub, rg, vnetName, ""))
	if props, ok := vnet["properties"].(map[string]interface{}); ok {
		if len(subnetItems) > 0 {
			props["subnets"] = subnetItems
		} else {
			props["subnets"] = []interface{}{}
		}
	}
	h.store.Set(vk, vnet)
}
