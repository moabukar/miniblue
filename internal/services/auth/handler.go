package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/auth/{tenantId}/oauth2/v2.0/token", h.Token)
	r.Post("/auth/{tenantId}/oauth2/token", h.Token)
	r.Get("/auth/{tenantId}/.well-known/openid-configuration", h.OpenIDConfig)
	r.Get("/auth/common/.well-known/openid-configuration", h.OpenIDConfig)
}

func (h *Handler) Token(w http.ResponseWriter, r *http.Request) {
	token := map[string]interface{}{
		"access_token":  "local-azure-mock-access-token",
		"token_type":    "Bearer",
		"expires_in":    86400,
		"expires_on":    time.Now().Add(24 * time.Hour).Unix(),
		"not_before":    time.Now().Unix(),
		"resource":      r.FormValue("resource"),
		"refresh_token": "local-azure-mock-refresh-token",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(token)
}

func (h *Handler) OpenIDConfig(w http.ResponseWriter, r *http.Request) {
	tenantId := chi.URLParam(r, "tenantId")
	if tenantId == "" {
		tenantId = "common"
	}
	config := map[string]interface{}{
		"issuer":                   "https://sts.windows.net/" + tenantId + "/",
		"authorization_endpoint":   "http://localhost:4566/auth/" + tenantId + "/oauth2/v2.0/authorize",
		"token_endpoint":           "http://localhost:4566/auth/" + tenantId + "/oauth2/v2.0/token",
		"jwks_uri":                 "http://localhost:4566/auth/" + tenantId + "/discovery/v2.0/keys",
		"response_types_supported": []string{"code", "token"},
		"grant_types_supported":    []string{"authorization_code", "client_credentials", "refresh_token"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}
