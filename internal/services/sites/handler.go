package sites

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/services/sites/config"
	"github.com/moabukar/miniblue/internal/store"
)

const (
	APIVersion = "2024-03-01"
)

type Site struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Kind       string            `json:"kind,omitempty"`
	Properties SiteProps         `json:"properties"`
	Tags       map[string]string `json:"tags,omitempty"`
}

type SiteProps struct {
	ProvisioningState string            `json:"provisioningState"`
	DefaultHostName   string            `json:"defaultHostName"`
	HostNames         []string          `json:"hostNames,omitempty"`
	State             string            `json:"state"`
	Enabled           bool              `json:"enabled"`
	ServerFarmID      string            `json:"serverFarmId,omitempty"`
	SiteConfig        *SiteConfig       `json:"siteConfig,omitempty"`
	AppSettings       map[string]string `json:"appSettings,omitempty"`
}

type SiteConfig struct {
	LinuxFxVersion   string `json:"linuxFxVersion,omitempty"`
	WindowsFxVersion string `json:"windowsFxVersion,omitempty"`
	FtpsState        string `json:"ftpsState,omitempty"`
	MinTlsVersion    string `json:"minTlsVersion,omitempty"`
	AlwaysOn         bool   `json:"alwaysOn,omitempty"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Web/sites", func(r chi.Router) {
		r.Get("/", h.List)
		r.Route("/{name}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdate)
			r.Get("/", h.Get)
			r.Delete("/", h.Delete)

			config.NewHandler(h.store).Register(r)

			r.Route("/basicPublishingCredentialsPolicies", func(r chi.Router) {
				r.Get("/", h.GetBasicPublishingCredentialsPolicies)
				r.Route("/ftp", func (r chi.Router) {
					r.Get("/", h.GetFtpAllowed)
					r.Put("/", h.UpdateFtpAllowed)
				})
				r.Route("/scm", func (r chi.Router) {
					r.Get("/", h.GetScmAllowed)
					r.Put("/", h.UpdateScmAllowed)
				})
			})

			r.Route("/slots", func(r chi.Router) {
				r.Get("/", h.ListSlots)
				r.Route("/{slotName}", func(r chi.Router) {
					r.Put("/", h.CreateOrUpdateSlot)
					r.Get("/", h.GetSlot)
					r.Delete("/", h.DeleteSlot)
				})
			})
		})
	})

	h.RegisterServerFarms(r)
	r.Post("/subscriptions/{subscriptionId}/providers/Microsoft.Web/checknameavailability", h.CheckNameAvailability)
}

func (h *Handler) siteResourceGroupKey(sub, rg, name string) string {
	return "func:" + sub + ":" + rg + ":" + name
}

func (h *Handler) siteSubscriptionKey(sub, name string) string {
	return "func:" + sub + ":" + name
}

func (h *Handler) buildSiteResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	kind, _ := input["kind"].(string)
	if kind == "" {
		kind = "functionapp"
	}

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	tags, _ := input["tags"].(map[string]interface{})
	if tags == nil {
		tags = map[string]interface{}{}
	}

	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.Web/sites",
		"kind":     kind,
		"location": location,
		"tags":     tags,
		"properties": map[string]interface{}{
			"provisioningState": "Succeeded",
			"defaultHostName":   name + ".azurewebsites.net",
			"hostNames":         []interface{}{name + ".azurewebsites.net"},
			"state":             "Running",
			"enabled":           true,
			"serverFarmId":      props["serverFarmId"],
		},
	}
}

func (h *Handler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	site := h.buildSiteResponse(sub, rg, name, input)
	kRG := h.siteResourceGroupKey(sub, rg, name)
	kSub := h.siteSubscriptionKey(sub,name)
	_, exists := h.store.Get(kRG)
	h.store.Set(kRG, site)
	h.store.Set(kSub, site)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(site)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteResourceGroupKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	if !h.store.Delete(h.siteResourceGroupKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	funcItems := h.store.ListByPrefix("func:" + sub + ":" + rg + ":")
	webappItems := h.store.ListByPrefix("webapp:" + sub + ":" + rg + ":")
	allItems := append(funcItems, webappItems...)
	json.NewEncoder(w).Encode(map[string]interface{}{"value": allItems})
}

func (h *Handler) CheckNameAvailability(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "InvalidRequestBody",
				"message": "Could not parse request body.",
			},
		})
		return
	}

	sub := chi.URLParam(r, "subscriptionId")
	nameAvailable := true
	reason := ""
	message := ""

	switch input.Type {
	case "Microsoft.Web/sites":
		site, _ := h.store.Get("func:" + sub + ":" + input.Name)
		if site != nil {
			nameAvailable = false
			reason = "AlreadyExists"
			message = "The site " + input.Name + " is already in use."
		}
	default:
		nameAvailable = true
	}

	resp := map[string]interface{}{
		"nameAvailable": nameAvailable,
	}
	if !nameAvailable {
		resp["reason"] = reason
		resp["message"] = message
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) basicPublishingCredentialsPoliciesKey(sub, rg, site string) string {
	return "webapp:basicpublishingcredentialspolicies:" + sub + ":" + rg + ":" + site
}

func (h *Handler) GetBasicPublishingCredentialsPolicies(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteResourceGroupKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	policies, ok := h.store.Get(h.basicPublishingCredentialsPoliciesKey(sub, rg, name))
	if !ok {
		policies = map[string]interface{}{
			"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/basicPublishingCredentialsPolicies",
			"name": name + "/basicPublishingCredentialsPolicies",
			"type": "Microsoft.Web/sites/config",
			"kind": nil,
			"properties": map[string]interface{}{
				"scm": map[string]interface{}{
					"allow": true,
				},
				"ftp": map[string]interface{}{
					"allow": true,
				},
			},
		}
	}

	json.NewEncoder(w).Encode(policies)
	_ = v
}

func (h *Handler) UpdateBasicPublishingCredentialsPolicies(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteResourceGroupKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{
			"scm": map[string]interface{}{
				"allow": true,
			},
			"ftp": map[string]interface{}{
				"allow": true,
			},
		}
	}

	policies := map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/basicPublishingCredentialsPolicies",
		"name":       name + "/basicPublishingCredentialsPolicies",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": props,
	}

	h.store.Set(h.basicPublishingCredentialsPoliciesKey(sub, rg, name), policies)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(policies)
	_ = v
}

func (h *Handler) scmAllowedKey(sub, rg, site string) string {
	return "webapp:scmallowed:" + sub + ":" + rg + ":" + site
}

func (h *Handler) GetScmAllowed(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteResourceGroupKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	scmAllowed, ok := h.store.Get(h.scmAllowedKey(sub, rg, name))
	if !ok {
		scmAllowed = map[string]interface{}{
			"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/basicPublishingCredentialsPolicies/scm",
			"name": name + "/scm",
			"type": "Microsoft.Web/sites/config",
			"kind": nil,
			"properties": map[string]interface{}{
				"allow": true,
			},
		}
	}

	json.NewEncoder(w).Encode(scmAllowed)
	_ = v
}

func (h *Handler) UpdateScmAllowed(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteResourceGroupKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{
			"allow": true,
		}
	}

	scmAllowed := map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/basicPublishingCredentialsPolicies/scm",
		"name":       name + "/scm",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": props,
	}

	h.store.Set(h.scmAllowedKey(sub, rg, name), scmAllowed)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(scmAllowed)
	_ = v
}

func (h *Handler) ftpAllowedKey(sub, rg, site string) string {
	return "webapp:ftpassigned:" + sub + ":" + rg + ":" + site
}

func (h *Handler) GetFtpAllowed(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteResourceGroupKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	ftpAllowed, ok := h.store.Get(h.ftpAllowedKey(sub, rg, name))
	if !ok {
		ftpAllowed = map[string]interface{}{
			"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/basicPublishingCredentialsPolicies/ftp",
			"name": name + "/ftp",
			"type": "Microsoft.Web/sites/config",
			"kind": nil,
			"properties": map[string]interface{}{
				"allow": true,
			},
		}
	}

	json.NewEncoder(w).Encode(ftpAllowed)
	_ = v
}

func (h *Handler) UpdateFtpAllowed(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteResourceGroupKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{
			"allow": true,
		}
	}

	ftpAllowed := map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/basicPublishingCredentialsPolicies/ftp",
		"name":       name + "/ftp",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": props,
	}

	h.store.Set(h.ftpAllowedKey(sub, rg, name), ftpAllowed)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ftpAllowed)
	_ = v
}
