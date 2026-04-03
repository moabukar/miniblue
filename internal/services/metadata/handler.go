package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

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
	r.Get("/metadata/endpoints", h.Endpoints)
}

func (h *Handler) baseURL() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4566"
	}
	host := os.Getenv("LOCAL_AZURE_HOST")
	if host == "" {
		host = "http://localhost"
	}
	return fmt.Sprintf("%s:%s", host, port)
}

func (h *Handler) Endpoints(w http.ResponseWriter, r *http.Request) {
	base := h.baseURL()

	endpoints := map[string]interface{}{
		"galleryEndpoint":                        nil,
		"graphEndpoint":                          base,
		"portalEndpoint":                         base,
		"authentication": map[string]interface{}{
			"loginEndpoint": base + "/auth",
			"audiences":     []string{base},
		},
		"media":                                  base,
		"vmImageAliasDoc":                        "",
		"resourceManagerEndpoint":                base,
		"sqlManagementEndpoint":                  base,
		"batchResourceId":                        base,
		"storageEndpointSuffix":                  "localhost",
		"keyVaultDnsSuffix":                      "localhost",
		"sqlServerHostnameSuffix":                "localhost",
		"mysqlServerEndpoint":                    base,
		"postgresqlServerEndpoint":               base,
		"cosmosDBDnsSuffix":                      "localhost",
		"containerRegistryDnsSuffix":             "localhost",
		"serviceBusEndpointSuffix":               "localhost",
		"activeDirectoryEndpoint":                base + "/auth",
		"activeDirectoryResourceId":              base,
		"activeDirectoryGraphResourceId":         base,
		"microsoftGraphResourceId":               base,
		"appInsightsResourceId":                  base,
		"appInsightsTelemetryChannelResourceId":  base,
		"logAnalyticsResourceId":                 base,
		"attestationResourceId":                  base,
		"synapseAnalyticsResourceId":             base,
		"ossrdbmsResourceId":                     base,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoints)
}
