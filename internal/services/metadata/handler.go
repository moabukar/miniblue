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

	// This must match go-autorest/autorest/azure.Environment struct exactly.
	// The azurerm Terraform provider unmarshals this into that struct.
	endpoints := map[string]interface{}{
		// Required: name must be non-nil or provider panics
		"name": "local-azure",

		// Core endpoints
		"managementPortalURL":       httpBase,
		"publishSettingsURL":        httpBase + "/publishsettings",
		"serviceManagementEndpoint": httpBase + "/",
		"resourceManagerEndpoint":   httpBase + "/",
		"activeDirectoryEndpoint":   httpBase + "/",
		"galleryEndpoint":           httpBase + "/",
		"keyVaultEndpoint":          httpBase + "/",
		"managedHSMEndpoint":        httpBase + "/",
		"graphEndpoint":             httpBase + "/",
		"serviceBusEndpoint":        httpBase + "/",
		"batchManagementEndpoint":   httpBase + "/",
		"microsoftGraphEndpoint":    httpBase + "/",

		// DNS suffixes
		"storageEndpointSuffix":        "localhost:" + httpPort,
		"cosmosDBDNSSuffix":            "localhost",
		"mariaDBDNSSuffix":             "localhost",
		"mySqlDatabaseDNSSuffix":       "localhost",
		"postgresqlDatabaseDNSSuffix":  "localhost",
		"sqlDatabaseDNSSuffix":         "localhost",
		"trafficManagerDNSSuffix":      "localhost",
		"keyVaultDNSSuffix":            "localhost",
		"managedHSMDNSSuffix":          "localhost",
		"serviceBusEndpointSuffix":     "localhost",
		"serviceManagementVMDNSSuffix": "localhost",
		"resourceManagerVMDNSSuffix":   "localhost",
		"containerRegistryDNSSuffix":   "localhost",
		"tokenAudience":                httpBase + "/",
		"apiManagementHostNameSuffix":  "localhost",
		"synapseEndpointSuffix":        "localhost",
		"datalakeSuffix":               "localhost",

		// Resource identifiers
		"resourceIdentifiers": map[string]interface{}{
			"graph":               httpBase + "/",
			"keyVault":            httpBase,
			"datalake":            httpBase + "/",
			"batch":               httpBase + "/",
			"operationalInsights": httpBase,
			"ossRDBMS":            httpBase,
			"storage":             httpBase + "/",
			"synapse":             httpBase,
			"serviceBus":          httpBase + "/",
			"sqlDatabase":         httpBase + "/",
			"cosmosDB":            httpBase,
			"managedHSM":          httpBase,
			"microsoftGraph":      httpBase + "/",
		},

		// Legacy fields some older provider versions may read
		"authentication": map[string]interface{}{
			"loginEndpoint": httpBase,
			"audiences":     []string{httpBase + "/"},
		},
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
