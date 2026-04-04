package server

import (
	"net/http"

	"github.com/google/uuid"
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
