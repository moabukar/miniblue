package auth

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
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
	// OAuth2 token endpoints
	r.Post("/{tenantId}/oauth2/v2.0/token", h.Token)
	r.Post("/{tenantId}/oauth2/token", h.Token)

	// OpenID Connect discovery
	r.Get("/{tenantId}/.well-known/openid-configuration", h.OpenIDConfig)
	r.Get("/{tenantId}/v2.0/.well-known/openid-configuration", h.OpenIDConfig)
	r.Get("/common/.well-known/openid-configuration", h.OpenIDConfigCommon)

	// Authorization endpoint
	r.Get("/{tenantId}/oauth2/v2.0/authorize", h.Authorize)
	r.Get("/{tenantId}/oauth2/authorize", h.Authorize)

	// Instance discovery
	r.Get("/common/discovery/instance", h.DiscoveryInstance)
	r.Get("/{tenantId}/discovery/instance", h.DiscoveryInstance)
}

func httpBase() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4566"
	}
	return "http://localhost:" + port
}

func baseFromRequest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func (h *Handler) Token(w http.ResponseWriter, r *http.Request) {
	tenantId := chi.URLParam(r, "tenantId")
	token := map[string]interface{}{
		"access_token":   "local-azure-mock-access-token",
		"token_type":     "Bearer",
		"expires_in":     86400,
		"expires_on":     time.Now().Add(24 * time.Hour).Unix(),
		"not_before":     time.Now().Unix(),
		"resource":       r.FormValue("resource"),
		"refresh_token":  "local-azure-mock-refresh-token",
		"scope":          r.FormValue("scope"),
		"ext_expires_in": 86400,
		"foci":           "1",
		"id_token":       "eyJ0eXAiOiJKV1QiLCJhbGciOiJub25lIn0.eyJhdWQiOiJsb2NhbC1henVyZSIsImlzcyI6Imh0dHBzOi8vc3RzLndpbmRvd3MubmV0LyIsInRpZCI6IiIsInN1YiI6ImxvY2FsLWF6dXJlLXVzZXIifQ.",
		"client_info":    "eyJ1aWQiOiJsb2NhbC1henVyZS11c2VyIiwidXRpZCI6IiJ9",
		"tenant_id":      tenantId,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(token)
}

func (h *Handler) OpenIDConfig(w http.ResponseWriter, r *http.Request) {
	tenantId := chi.URLParam(r, "tenantId")
	if tenantId == "" {
		tenantId = "common"
	}
	h.writeOpenIDConfig(w, r, tenantId)
}

func (h *Handler) OpenIDConfigCommon(w http.ResponseWriter, r *http.Request) {
	h.writeOpenIDConfig(w, r, "common")
}

func (h *Handler) writeOpenIDConfig(w http.ResponseWriter, r *http.Request, tenantId string) {
	base := baseFromRequest(r)
	config := map[string]interface{}{
		"issuer":                                base + "/" + tenantId,
		"authorization_endpoint":                base + "/" + tenantId + "/oauth2/v2.0/authorize",
		"token_endpoint":                        base + "/" + tenantId + "/oauth2/v2.0/token",
		"device_authorization_endpoint":         base + "/" + tenantId + "/oauth2/v2.0/devicecode",
		"jwks_uri":                              base + "/" + tenantId + "/discovery/v2.0/keys",
		"userinfo_endpoint":                     base + "/" + tenantId + "/openid/userinfo",
		"tenant_region_scope":                   "NA",
		"cloud_instance_name":                   "local-azure",
		"cloud_graph_host_name":                 "localhost",
		"msgraph_host":                          "localhost",
		"rbac_url":                              base,
		"response_types_supported":              []string{"code", "id_token", "code id_token", "token id_token", "token"},
		"subject_types_supported":               []string{"pairwise"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "private_key_jwt", "client_secret_basic"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "name", "email", "oid", "tid"},
		"grant_types_supported":                 []string{"authorization_code", "client_credentials", "refresh_token", "urn:ietf:params:oauth:grant-type:device_code"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")
	if redirectURI != "" {
		sep := "?"
		if strings.Contains(redirectURI, "?") {
			sep = "&"
		}
		http.Redirect(w, r, redirectURI+sep+"code=local-azure-mock-code&state="+state, http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"code": "local-azure-mock-code"})
}

func (h *Handler) DiscoveryInstance(w http.ResponseWriter, r *http.Request) {
	base := baseFromRequest(r)
	port := os.Getenv("PORT")
	if port == "" {
		port = "4566"
	}
	tlsPort := os.Getenv("TLS_PORT")
	if tlsPort == "" {
		tlsPort = "4567"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_discovery_endpoint": base + "/common/.well-known/openid-configuration",
		"metadata": []map[string]interface{}{
			{
				"preferred_network": "localhost",
				"preferred_cache":   "localhost",
				"aliases":           []string{"localhost", "localhost:" + port, "localhost:" + tlsPort},
			},
		},
	})
}
