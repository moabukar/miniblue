package server

import (
	"net/http"

	"github.com/google/uuid"
)

// CORS adds permissive CORS headers for local development.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-ms-version, x-ms-date, x-ms-client-request-id")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

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

// APIVersionCheck stores the api-version if provided but does not reject requests without it.
// Real Azure requires api-version on ARM calls but miniblue is lenient for local development.
func APIVersionCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiVersion := r.URL.Query().Get("api-version"); apiVersion != "" {
			w.Header().Set("x-ms-api-version", apiVersion)
		}
		next.ServeHTTP(w, r)
	})
}
