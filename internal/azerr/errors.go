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

func Conflict(w http.ResponseWriter, resourceType, name string) {
	WriteError(w, http.StatusConflict, "Conflict",
		"The resource '"+resourceType+"/"+name+"' already exists.")
}

func Gone(w http.ResponseWriter, resourceType, name string) {
	WriteError(w, http.StatusGone, "ResourceGone",
		"The resource '"+resourceType+"/"+name+"' has been deleted.")
}

func MethodNotAllowed(w http.ResponseWriter, method string) {
	WriteError(w, http.StatusMethodNotAllowed, "MethodNotAllowed",
		"The request method '"+method+"' is not allowed.")
}
