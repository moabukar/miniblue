package subscriptions

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type Subscription struct {
	ID                  string `json:"id"`
	SubscriptionId      string `json:"subscriptionId"`
	DisplayName         string `json:"displayName"`
	State               string `json:"state"`
	TenantId            string `json:"tenantId"`
	AuthorizationSource string `json:"authorizationSource"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/subscriptions", h.ListSubscriptions)
	r.Get("/subscriptions/{subscriptionId}", h.GetSubscription)
	r.Get("/tenants", h.ListTenants)

	// Provider registration (Terraform calls this even with skip_provider_registration)
	r.Get("/subscriptions/{subscriptionId}/providers", h.ListProviders)
	r.Get("/subscriptions/{subscriptionId}/providers/{providerNamespace}", h.GetProvider)
	r.Post("/subscriptions/{subscriptionId}/providers/{providerNamespace}/register", h.RegisterProvider)
}

func (h *Handler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	subs := []Subscription{defaultSub()}
	json.NewEncoder(w).Encode(map[string]interface{}{"value": subs})
}

func (h *Handler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	subId := chi.URLParam(r, "subscriptionId")
	sub := defaultSub()
	sub.ID = "/subscriptions/" + subId
	sub.SubscriptionId = subId
	json.NewEncoder(w).Encode(sub)
}

func (h *Handler) ListTenants(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"value": []map[string]interface{}{
			{
				"id":          "/tenants/00000000-0000-0000-0000-000000000001",
				"tenantId":    "00000000-0000-0000-0000-000000000001",
				"displayName": "local-azure",
				"tenantType":  "AAD",
			},
		},
	})
}

func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers := []map[string]interface{}{}
	for _, ns := range supportedProviders {
		providers = append(providers, providerEntry(ns))
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"value": providers})
}

func (h *Handler) GetProvider(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "providerNamespace")
	json.NewEncoder(w).Encode(providerEntry(ns))
}

func (h *Handler) RegisterProvider(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "providerNamespace")
	json.NewEncoder(w).Encode(providerEntry(ns))
}

func defaultSub() Subscription {
	return Subscription{
		ID:                  "/subscriptions/00000000-0000-0000-0000-000000000000",
		SubscriptionId:      "00000000-0000-0000-0000-000000000000",
		DisplayName:         "local-azure",
		State:               "Enabled",
		TenantId:            "00000000-0000-0000-0000-000000000001",
		AuthorizationSource: "Legacy",
	}
}

var supportedProviders = []string{
	"Microsoft.Resources",
	"Microsoft.Storage",
	"Microsoft.Network",
	"Microsoft.KeyVault",
	"Microsoft.DocumentDB",
	"Microsoft.ServiceBus",
	"Microsoft.Web",
	"Microsoft.ContainerRegistry",
	"Microsoft.EventGrid",
	"Microsoft.ManagedIdentity",
	"Microsoft.AppConfiguration",
}

func providerEntry(namespace string) map[string]interface{} {
	return map[string]interface{}{
		"id":                "/subscriptions/00000000-0000-0000-0000-000000000000/providers/" + namespace,
		"namespace":         namespace,
		"registrationState": "Registered",
		"resourceTypes":     []interface{}{},
	}
}
