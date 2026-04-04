package network

import (
	"encoding/json"
	"github.com/moabukar/local-azure/internal/azerr"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type VNet struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Location   string `json:"location"`
	Properties struct {
		AddressSpace struct {
			AddressPrefixes []string `json:"addressPrefixes"`
		} `json:"addressSpace"`
		Subnets []Subnet `json:"subnets,omitempty"`
	} `json:"properties"`
}

type Subnet struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Properties struct {
		AddressPrefix string `json:"addressPrefix"`
	} `json:"properties"`
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
			r.Route("/subnets/{subnetName}", func(r chi.Router) {
				r.Put("/", h.CreateOrUpdateSubnet)
				r.Get("/", h.GetSubnet)
				r.Delete("/", h.DeleteSubnet)
			})
		})
	})
}

func (h *Handler) vnetKey(sub, rg, name string) string {
	return "vnet:" + sub + ":" + rg + ":" + name
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
	
	h.store.Set(h.vnetKey(sub, rg, name), vnet)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(vnet)
}

func (h *Handler) GetVNet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "vnetName")
	
	v, ok := h.store.Get(h.vnetKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/virtualNetworks", "resource")
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteVNet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "vnetName")
	h.store.Delete(h.vnetKey(sub, rg, name))
	w.WriteHeader(http.StatusOK)
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
	
	var subnet Subnet
	json.NewDecoder(r.Body).Decode(&subnet)
	subnet.ID = "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetName
	subnet.Name = subnetName
	
	key := "subnet:" + sub + ":" + rg + ":" + vnetName + ":" + subnetName
	h.store.Set(key, subnet)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(subnet)
}

func (h *Handler) GetSubnet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vnetName := chi.URLParam(r, "vnetName")
	subnetName := chi.URLParam(r, "subnetName")
	
	key := "subnet:" + sub + ":" + rg + ":" + vnetName + ":" + subnetName
	v, ok := h.store.Get(key)
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/virtualNetworks", "resource")
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteSubnet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vnetName := chi.URLParam(r, "vnetName")
	subnetName := chi.URLParam(r, "subnetName")
	
	key := "subnet:" + sub + ":" + rg + ":" + vnetName + ":" + subnetName
	h.store.Delete(key)
	w.WriteHeader(http.StatusOK)
}
