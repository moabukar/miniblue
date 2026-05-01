package resourcegroups

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/services/aks"
	"github.com/moabukar/miniblue/internal/store"
)

type ResourceGroup struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties struct {
		ProvisioningState string `json:"provisioningState"`
	} `json:"properties"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourcegroups", func(r chi.Router) {
		r.Get("/", h.List)
		r.Route("/{resourceGroupName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdate)
			r.Patch("/", h.Update)
			r.Get("/", h.Get)
			r.Delete("/", h.Delete)
			r.Head("/", h.CheckExistence)
			r.Get("/resources", h.ListResources)
		})
	})
	// Case-sensitive duplicate (Azure ARM is case-insensitive on 'resourceGroups')
	r.Get("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/resources", h.ListResources)
	// Async operation status polling endpoint
	r.Get("/subscriptions/{subscriptionId}/operationresults/*", h.OperationResult)
}

func (h *Handler) key(sub, name string) string {
	return "rg:" + sub + ":" + name
}

func (h *Handler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")

	var input struct {
		Location string            `json:"location"`
		Tags     map[string]string `json:"tags,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		azerr.BadRequest(w, "The request content was invalid: "+err.Error())
		return
	}
	if input.Location == "" {
		azerr.BadRequest(w, "The location property is required for resource group creation.")
		return
	}

	k := h.key(sub, name)
	_, exists := h.store.Get(k)

	rg := ResourceGroup{
		ID:       "/subscriptions/" + sub + "/resourceGroups/" + name,
		Name:     name,
		Type:     "Microsoft.Resources/resourceGroups",
		Location: input.Location,
		Tags:     input.Tags,
	}
	rg.Properties.ProvisioningState = "Succeeded"

	h.store.Set(k, rg)

	if exists {
		// Azure returns 200 for update of existing resource group
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(rg)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")

	k := h.key(sub, name)
	v, ok := h.store.Get(k)
	if !ok {
		azerr.NotFound(w, "Microsoft.Resources/resourceGroups", name)
		return
	}

	rg := v.(ResourceGroup)
	var patch struct {
		Tags map[string]string `json:"tags,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&patch)
	if patch.Tags != nil {
		rg.Tags = patch.Tags
	}

	h.store.Set(k, rg)
	json.NewEncoder(w).Encode(rg)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")

	v, ok := h.store.Get(h.key(sub, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Resources/resourceGroups", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) CheckExistence(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")

	if h.store.Exists(h.key(sub, name)) {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "resourceGroupName")

	k := h.key(sub, name)
	if !h.store.Delete(k) {
		azerr.NotFound(w, "Microsoft.Resources/resourceGroups", name)
		return
	}

	// Also clean up resources in this resource group
	p := sub + ":" + name + ":"
	h.store.DeleteByPrefix("vnet:" + p)
	h.store.DeleteByPrefix("subnet:" + p)
	h.store.DeleteByPrefix("func:" + p)
	h.store.DeleteByPrefix("acr:registry:" + p)
	h.store.DeleteByPrefix("eg:topic:" + p)
	h.store.DeleteByPrefix("dns:zone:" + p)
	h.store.DeleteByPrefix("dns:record:" + p)
	h.store.DeleteByPrefix("redis:" + p)
	h.store.DeleteByPrefix("pgflex:server:" + p)
	h.store.DeleteByPrefix("pgflex:db:" + p)
	h.store.DeleteByPrefix("dbmysql:server:" + p)
	h.store.DeleteByPrefix("dbmysql:db:" + p)
	h.store.DeleteByPrefix("sqldb:server:" + p)
	h.store.DeleteByPrefix("sqldb:db:" + p)
	h.store.DeleteByPrefix("aci:containergroup:" + p)
	// Tear down any real AKS k3s containers BEFORE clearing the store, so we
	// can read the backend handles. No-op in stub mode.
	aks.CleanupClustersInRG(h.store, sub, name)
	h.store.DeleteByPrefix("aks:cluster:" + p)
	h.store.DeleteByPrefix("nsg:" + p)
	h.store.DeleteByPrefix("nsgrule:" + p)
	h.store.DeleteByPrefix("publicip:" + p)
	h.store.DeleteByPrefix("lb:" + p)
	h.store.DeleteByPrefix("appgw:" + p)
	h.store.DeleteByPrefix("blob:account:" + p)
	h.store.DeleteByPrefix("blob:armcontainer:" + p)
	h.store.DeleteByPrefix("cosmos:account:" + p)
	h.store.DeleteByPrefix("cosmos:sqldb:" + p)
	h.store.DeleteByPrefix("cosmos:container:" + p)
	h.store.DeleteByPrefix("sb:namespace:" + p)
	h.store.DeleteByPrefix("sb:armqueue:" + p)
	h.store.DeleteByPrefix("sb:armtopic:" + p)
	h.store.DeleteByPrefix("appconfig:store:" + p)

	// Async operation - return Location header for polling (Terraform requires this on 202)
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	opURL := fmt.Sprintf("%s://%s/subscriptions/%s/operationresults/delete-rg-%s?api-version=2020-06-01", scheme, r.Host, sub, name)
	w.Header().Set("Location", opURL)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	items := h.store.ListByPrefix("rg:" + sub + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// ListResources returns resources within a resource group.
// The azurerm TF provider calls this before deleting a group.
func (h *Handler) ListResources(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"value": []interface{}{}})
}

// OperationResult returns 200 (completed) for async operation polling.
// Since miniblue operations are synchronous, all ops are already done.
func (h *Handler) OperationResult(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
