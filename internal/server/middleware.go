package server

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/moabukar/local-azure/internal/azerr"
)

// AzureHeaders adds standard Azure response headers to every request.
func AzureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		correlationID := uuid.New().String()

		w.Header().Set("x-ms-version", "2023-11-03")
		w.Header().Set("x-ms-request-id", requestID)
		w.Header().Set("x-ms-correlation-request-id", correlationID)
		w.Header().Set("x-ms-routing-request-id", "LOCALAZURE:"+requestID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		next.ServeHTTP(w, r)
	})
}

// apiVersionPaths are ARM paths that require api-version in real Azure.
var apiVersionPrefixes = []string{
	"/subscriptions/",
}

// APIVersionCheck validates that ARM requests include ?api-version= parameter.
// Non-ARM paths (health, blob data plane, cosmosdb, etc.) are exempt.
func APIVersionCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Only enforce on ARM management paths
		needsVersion := false
		for _, prefix := range apiVersionPrefixes {
			if strings.HasPrefix(path, prefix) {
				needsVersion = true
				break
			}
		}

		if needsVersion {
			apiVersion := r.URL.Query().Get("api-version")
			if apiVersion == "" {
				azerr.WriteError(w, http.StatusBadRequest,
					"MissingApiVersionParameter",
					"The api-version query parameter (?api-version=) is required for all API calls.")
				return
			}
			// Store the api-version for handlers that need it
			w.Header().Set("x-ms-api-version", apiVersion)
		}

		next.ServeHTTP(w, r)
	})
}
