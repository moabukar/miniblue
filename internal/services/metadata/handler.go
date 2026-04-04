package metadata

import (
	"encoding/json"
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

func (h *Handler) Endpoints(w http.ResponseWriter, r *http.Request) {
	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "4566"
	}

	httpBase := "http://localhost:" + httpPort

	endpoints := map[string]interface{}{
		"galleryEndpoint":                       nil,
		"graphEndpoint":                         httpBase,
		"portalEndpoint":                        httpBase,
		"authentication": map[string]interface{}{
			"loginEndpoint": httpBase,
			"audiences":     []string{httpBase},
		},
		"media":                                 httpBase,
		"vmImageAliasDoc":                       "",
		"resourceManagerEndpoint":               httpBase,
		"sqlManagementEndpoint":                 httpBase,
		"batchResourceId":                       httpBase,
		"storageEndpointSuffix":                 "localhost",
		"keyVaultDnsSuffix":                     "localhost",
		"sqlServerHostnameSuffix":               "localhost",
		"mysqlServerEndpoint":                   httpBase,
		"postgresqlServerEndpoint":              httpBase,
		"cosmosDBDnsSuffix":                     "localhost",
		"containerRegistryDnsSuffix":            "localhost",
		"serviceBusEndpointSuffix":              "localhost",
		"activeDirectoryEndpoint":               httpBase,
		"activeDirectoryResourceId":             httpBase,
		"activeDirectoryGraphResourceId":        httpBase,
		"microsoftGraphResourceId":              httpBase,
		"appInsightsResourceId":                 httpBase,
		"appInsightsTelemetryChannelResourceId": httpBase,
		"logAnalyticsResourceId":                httpBase,
		"attestationResourceId":                 httpBase,
		"synapseAnalyticsResourceId":            httpBase,
		"ossrdbmsResourceId":                    httpBase,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoints)
}
