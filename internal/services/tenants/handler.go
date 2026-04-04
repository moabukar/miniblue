package tenants

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type Tenant struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenantId"`
	DisplayName string `json:"displayName"`
	TenantType  string `json:"tenantType"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/tenants", h.List)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"value": []Tenant{
			{
				ID:          "/tenants/00000000-0000-0000-0000-000000000001",
				TenantID:    "00000000-0000-0000-0000-000000000001",
				DisplayName: "local-azure",
				TenantType:  "AAD",
			},
		},
	})
}
