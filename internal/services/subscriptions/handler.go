package subscriptions

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

const defaultSubscriptionID = "00000000-0000-0000-0000-000000000000"

type Subscription struct {
	ID                   string            `json:"id"`
	SubscriptionID       string            `json:"subscriptionId"`
	TenantID             string            `json:"tenantId"`
	DisplayName          string            `json:"displayName"`
	State                string            `json:"state"`
	SubscriptionPolicies map[string]string `json:"subscriptionPolicies"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/subscriptions", h.List)
	r.Get("/subscriptions/{subscriptionId}", h.Get)
}

func mockSubscription(subID string) Subscription {
	return Subscription{
		ID:             "/subscriptions/" + subID,
		SubscriptionID: subID,
		TenantID:       "00000000-0000-0000-0000-000000000001",
		DisplayName:    "local-azure",
		State:          "Enabled",
		SubscriptionPolicies: map[string]string{
			"locationPlacementId": "Internal_2014-09-01",
			"quotaId":             "Internal_2014-09-01",
			"spendingLimit":       "Off",
		},
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"value": []Subscription{mockSubscription(defaultSubscriptionID)},
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	subID := chi.URLParam(r, "subscriptionId")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mockSubscription(subID))
}
