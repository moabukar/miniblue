package identity

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/store"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresOn    string `json:"expires_on"`
	Resource     string `json:"resource"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	// Managed Identity token endpoint (IMDS)
	r.Get("/metadata/identity/oauth2/token", h.GetToken)
	// Instance Metadata Service
	r.Get("/metadata/instance", h.GetInstanceMetadata)
}

func (h *Handler) GetToken(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	if resource == "" {
		resource = "https://management.azure.com/"
	}
	
	token := TokenResponse{
		AccessToken:  "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsIng1dCI6ImxvY2FsLWF6dXJlIn0.miniblue-mock-token",
		TokenType:    "Bearer",
		ExpiresIn:    86400,
		ExpiresOn:    time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		Resource:     resource,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(token)
}

func (h *Handler) GetInstanceMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := map[string]interface{}{
		"compute": map[string]interface{}{
			"location":       "eastus",
			"name":           "miniblue-vm",
			"resourceGroupName": "miniblue-rg",
			"subscriptionId": "00000000-0000-0000-0000-000000000000",
			"vmId":           "miniblue-vm-id",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metadata)
}
