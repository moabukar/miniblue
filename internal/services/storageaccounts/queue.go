package storageaccounts

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetQueueServiceProperties(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	account := chi.URLParam(r, "accountName")

	if _, ok := h.store.Get(h.storageAccountKey(sub, rg, account)); !ok {
		writeServiceNotFound(w, "Microsoft.Storage/storageAccounts/queueServices", account)
		return
	}

	resp := h.buildServicePropertiesResponse(sub, rg, account, "queueServices")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) SetQueueServiceProperties(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	account := chi.URLParam(r, "accountName")

	if _, ok := h.store.Get(h.storageAccountKey(sub, rg, account)); !ok {
		writeServiceNotFound(w, "Microsoft.Storage/storageAccounts/queueServices", account)
		return
	}

	resp := h.buildServicePropertiesResponse(sub, rg, account, "queueServices")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) PatchQueueServiceProperties(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	account := chi.URLParam(r, "accountName")

	if _, ok := h.store.Get(h.storageAccountKey(sub, rg, account)); !ok {
		writeServiceNotFound(w, "Microsoft.Storage/storageAccounts/queueServices", account)
		return
	}

	resp := h.buildServicePropertiesResponse(sub, rg, account, "queueServices")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
