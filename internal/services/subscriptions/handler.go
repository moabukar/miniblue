package subscriptions

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type Subscription struct {
	ID                   string `json:"id"`
	SubscriptionId       string `json:"subscriptionId"`
	DisplayName          string `json:"displayName"`
	State                string `json:"state"`
	TenantId             string `json:"tenantId"`
	AuthorizationSource  string `json:"authorizationSource"`
}

type Tenant struct {
	ID       string `json:"id"`
	TenantId string `json:"tenantId"`
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
}

func (h *Handler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	subs := []Subscription{
		{
			ID:                  "/subscriptions/00000000-0000-0000-0000-000000000000",
			SubscriptionId:     "00000000-0000-0000-0000-000000000000",
			DisplayName:        "local-azure",
			State:              "Enabled",
			TenantId:           "00000000-0000-0000-0000-000000000001",
			AuthorizationSource: "Legacy",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": subs})
}

func (h *Handler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	subId := chi.URLParam(r, "subscriptionId")
	sub := Subscription{
		ID:                  "/subscriptions/" + subId,
		SubscriptionId:     subId,
		DisplayName:        "local-azure",
		State:              "Enabled",
		TenantId:           "00000000-0000-0000-0000-000000000001",
		AuthorizationSource: "Legacy",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (h *Handler) ListTenants(w http.ResponseWriter, r *http.Request) {
	tenants := []Tenant{
		{
			ID:       "/tenants/00000000-0000-0000-0000-000000000001",
			TenantId: "00000000-0000-0000-0000-000000000001",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": tenants})
}
