package publicip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

var ipCounter uint32

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/publicIPAddresses", func(r chi.Router) {
		r.Get("/", h.List)
		r.Route("/{publicIpAddressName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdate)
			r.Get("/", h.Get)
			r.Delete("/", h.Delete)
		})
	})
}

func (h *Handler) key(sub, rg, name string) string {
	return "publicip:" + sub + ":" + rg + ":" + name
}

func nextIP() string {
	n := atomic.AddUint32(&ipCounter, 1)
	return fmt.Sprintf("20.0.%d.%d", (n/256)%256, n%256)
}

func buildResponse(sub, rg, name string, input map[string]interface{}, existingIP string) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/publicIPAddresses/" + name

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	allocationMethod, _ := props["publicIPAllocationMethod"].(string)
	if allocationMethod == "" {
		allocationMethod = "Static"
	}

	addressVersion, _ := props["publicIPAddressVersion"].(string)
	if addressVersion == "" {
		addressVersion = "IPv4"
	}

	idleTimeout := 4.0
	if v, ok := props["idleTimeoutInMinutes"].(float64); ok {
		idleTimeout = v
	}

	ipAddress := existingIP
	if ipAddress == "" {
		ipAddress = nextIP()
	}

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	sku, _ := input["sku"].(map[string]interface{})
	if sku == nil {
		sku = map[string]interface{}{"name": "Standard", "tier": "Regional"}
	}

	tags, _ := input["tags"].(map[string]interface{})
	if tags == nil {
		tags = map[string]interface{}{}
	}

	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.Network/publicIPAddresses",
		"location": location,
		"tags":     tags,
		"sku":      sku,
		"properties": map[string]interface{}{
			"provisioningState":       "Succeeded",
			"publicIPAllocationMethod": allocationMethod,
			"publicIPAddressVersion":  addressVersion,
			"ipAddress":               ipAddress,
			"idleTimeoutInMinutes":    idleTimeout,
			"ipConfiguration":         nil,
			"dnsSettings":             props["dnsSettings"],
		},
	}
}

func (h *Handler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "publicIpAddressName")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	var existingIP string
	k := h.key(sub, rg, name)
	existing, exists := h.store.Get(k)
	if exists {
		if m, ok := existing.(map[string]interface{}); ok {
			if p, ok := m["properties"].(map[string]interface{}); ok {
				existingIP, _ = p["ipAddress"].(string)
			}
		}
	}

	pip := buildResponse(sub, rg, name, input, existingIP)
	h.store.Set(k, pip)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(pip)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "publicIpAddressName")

	v, ok := h.store.Get(h.key(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/publicIPAddresses", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "publicIpAddressName")

	if !h.store.Delete(h.key(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Network/publicIPAddresses", name)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("publicip:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
