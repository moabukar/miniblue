package azerr

import (
	"encoding/json"
	"net/http"
)

type AzureErrorResponse struct {
	Error AzureErrorDetail `json:"error"`
}

type AzureErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteError(w http.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(AzureErrorResponse{
		Error: AzureErrorDetail{Code: code, Message: message},
	})
}

func NotFound(w http.ResponseWriter, resourceType, name string) {
	WriteError(w, http.StatusNotFound, "ResourceNotFound",
		"The Resource '"+resourceType+"/"+name+"' under resource group could not be found.")
}

func BadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, "InvalidRequestContent", message)
}
