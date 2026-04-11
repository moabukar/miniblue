package storageaccounts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetBlobServiceProperties(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	account := chi.URLParam(r, "accountName")

	if _, ok := h.store.Get(h.storageAccountKey(sub, rg, account)); !ok {
		writeServiceNotFound(w, "Microsoft.Storage/storageAccounts/blobServices", account)
		return
	}

	resp := h.buildServicePropertiesResponse(sub, rg, account, "blobServices")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) SetBlobServiceProperties(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	account := chi.URLParam(r, "accountName")

	if _, ok := h.store.Get(h.storageAccountKey(sub, rg, account)); !ok {
		writeServiceNotFound(w, "Microsoft.Storage/storageAccounts/blobServices", account)
		return
	}

	resp := h.buildServicePropertiesResponse(sub, rg, account, "blobServices")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) PatchBlobServiceProperties(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	account := chi.URLParam(r, "accountName")

	if _, ok := h.store.Get(h.storageAccountKey(sub, rg, account)); !ok {
		writeServiceNotFound(w, "Microsoft.Storage/storageAccounts/blobServices", account)
		return
	}

	resp := h.buildServicePropertiesResponse(sub, rg, account, "blobServices")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) armContainerKey(sub, rg, account, name string) string {
	return "blob:armcontainer:" + sub + ":" + rg + ":" + account + ":" + name
}

func (h *Handler) buildARMContainerResponse(sub, rg, account, name string) map[string]interface{} {
	return map[string]interface{}{
		"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Storage/storageAccounts/" + account + "/blobServices/default/containers/" + name,
		"name": name,
		"type": "Microsoft.Storage/storageAccounts/blobServices/containers",
		"properties": map[string]interface{}{
			"publicAccess":          "None",
			"leaseStatus":           "Unlocked",
			"leaseState":            "Available",
			"lastModifiedTime":      fmt.Sprintf("\"0x%X\"", time.Now().UnixNano()),
			"hasImmutabilityPolicy": false,
			"hasLegalHold":          false,
		},
	}
}

func (h *Handler) CreateContainerARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	account := chi.URLParam(r, "accountName")
	name := chi.URLParam(r, "containerName")

	k := h.armContainerKey(sub, rg, account, name)
	c := h.buildARMContainerResponse(sub, rg, account, name)
	h.store.Set(k, c)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(c)
}

func (h *Handler) GetContainerARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	account := chi.URLParam(r, "accountName")
	name := chi.URLParam(r, "containerName")

	v, ok := h.store.Get(h.armContainerKey(sub, rg, account, name))
	if !ok {
		writeServiceNotFound(w, "Microsoft.Storage/storageAccounts/blobServices/containers", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteContainerARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	account := chi.URLParam(r, "accountName")
	name := chi.URLParam(r, "containerName")

	if !h.store.Delete(h.armContainerKey(sub, rg, account, name)) {
		writeServiceNotFound(w, "Microsoft.Storage/storageAccounts/blobServices/containers", name)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListContainersARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	account := chi.URLParam(r, "accountName")
	items := h.store.ListByPrefix("blob:armcontainer:" + sub + ":" + rg + ":" + account + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
