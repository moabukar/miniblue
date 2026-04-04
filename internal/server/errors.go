package server

import (
	"encoding/json"
	"net/http"
)

type AzureError struct {
	Error AzureErrorDetail `json:"error"`
}

type AzureErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteError(w http.ResponseWriter, code int, errorCode string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(AzureError{
		Error: AzureErrorDetail{
			Code:    errorCode,
			Message: message,
		},
	})
}
