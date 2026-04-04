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
	r.Get("/metadata/instance", h.Instance)
}

func (h *Handler) Endpoints(w http.ResponseWriter, r *http.Request) {
	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "4566"
	}
	httpBase := "http://localhost:" + httpPort

	// This MUST match the metaDataResponse struct in:
	// github.com/hashicorp/go-azure-sdk/sdk/internal/metadata/client.go
	// The azurerm Terraform provider unmarshals this response using those exact JSON tags.
	endpoints := map[string]interface{}{
		"name":    "local-azure",
		"portal":  httpBase,
		"authentication": map[string]interface{}{
			"loginEndpoint":    httpBase + "/",
			"audiences":        []string{httpBase + "/"},
			"tenant":           "00000000-0000-0000-0000-000000000001",
			"identityProvider": "AAD",
		},
		"media":         httpBase + "/",
		"graphAudience": httpBase + "/",
		"graph":         httpBase + "/",
		"suffixes": map[string]interface{}{
			"azureDataLakeStoreFileSystem":        "localhost",
			"acrLoginServer":                      "localhost",
			"sqlServerHostname":                   "localhost",
			"azureDataLakeAnalyticsCatalogAndJob": "localhost",
			"keyVaultDns":                         "localhost",
			"storage":                             "localhost:" + httpPort,
			"azureFrontDoorEndpointSuffix":        "localhost",
			"storageSyncEndpointSuffix":           "localhost",
			"mhsmDns":                             "localhost",
			"mysqlServerEndpoint":                 "localhost",
			"postgresqlServerEndpoint":            "localhost",
			"mariadbServerEndpoint":               "localhost",
			"synapseAnalytics":                    "localhost",
			"attestationEndpoint":                 "localhost",
		},
		"batch":                                 httpBase + "/",
		"resourceManager":                       httpBase + "/",
		"vmImageAliasDoc":                       "",
		"activeDirectoryDataLake":               httpBase + "/",
		"sqlManagement":                         httpBase + "/",
		"microsoftGraphResourceId":              httpBase + "/",
		"appInsightsResourceId":                 httpBase + "/",
		"appInsightsTelemetryChannelResourceId": httpBase + "/",
		"attestationResourceId":                 httpBase + "/",
		"synapseAnalyticsResourceId":            httpBase + "/",
		"logAnalyticsResourceId":                httpBase + "/",
		"ossrDbmsResourceId":                    httpBase + "/",
		"gallery":                               httpBase + "/",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoints)
}

func (h *Handler) Instance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"compute": map[string]interface{}{
			"location":          "eastus",
			"name":              "local-azure-vm",
			"resourceGroupName": "local-azure-rg",
			"subscriptionId":    "00000000-0000-0000-0000-000000000000",
			"vmId":              "local-azure-vm-id",
			"azEnvironment":     "local-azure",
		},
	})
}
