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
	ProvisioningState string `json:"provisioningState"`
	AddressPrefix     string `json:"addressPrefix"`
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

func (h *Handler) CreateOrUpdateVNet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "vnetName")

	var vnet VNet
	json.NewDecoder(r.Body).Decode(&vnet)
	vnet.ID = "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/virtualNetworks/" + name
	vnet.Name = name
	vnet.Type = "Microsoft.Network/virtualNetworks"
	vnet.Properties.ProvisioningState = "Succeeded"

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

func (h *Handler) CreateOrUpdateSubnet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vnetName := chi.URLParam(r, "vnetName")
	subnetName := chi.URLParam(r, "subnetName")

	// Verify parent VNet exists
	if !h.store.Exists(h.vnetKey(sub, rg, vnetName)) {
		azerr.NotFound(w, "Microsoft.Network/virtualNetworks", vnetName)
		return
	}

	var subnet SubnetRef
	json.NewDecoder(r.Body).Decode(&subnet)
	subnet.ID = "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetName
	subnet.Name = subnetName
	subnet.Properties.ProvisioningState = "Succeeded"

	k := h.subnetKey(sub, rg, vnetName, subnetName)
	_, exists := h.store.Get(k)
	h.store.Set(k, subnet)

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
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListSubnets(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vnetName := chi.URLParam(r, "vnetName")
	items := h.store.ListByPrefix(h.subnetKey(sub, rg, vnetName, ""))
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
