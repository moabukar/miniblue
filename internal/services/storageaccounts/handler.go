package storageaccounts

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/storageauth"
	"github.com/moabukar/miniblue/internal/store"
)

const (
	envStorageEndpoint = "MINIBLUE_STORAGE_ENDPOINT"
)

// Store key prefixes match the historical blob handler so existing persisted state stays valid.

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	// Subscription-scoped list (azurerm and Azure SDK use this after create/update).
	r.Get("/subscriptions/{subscriptionId}/providers/Microsoft.Storage/storageAccounts", h.ListStorageAccountsInSubscription)

	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts", func(r chi.Router) {
		r.Get("/", h.ListStorageAccounts)
		r.Route("/{accountName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateStorageAccount)
			r.Get("/", h.GetStorageAccount)
			r.Delete("/", h.DeleteStorageAccount)
			r.Post("/listKeys", h.ListKeys)

			// Blob service
			r.Get("/blobServices/default", h.GetBlobServiceProperties)
			r.Put("/blobServices/default", h.SetBlobServiceProperties)
			r.Patch("/blobServices/default", h.PatchBlobServiceProperties)
			r.Route("/blobServices/default/containers", func(r chi.Router) {
				r.Get("/", h.ListContainersARM)
				r.Route("/{containerName}", func(r chi.Router) {
					r.Put("/", h.CreateContainerARM)
					r.Get("/", h.GetContainerARM)
					r.Delete("/", h.DeleteContainerARM)
				})
			})

			// File service
			r.Get("/fileServices/default", h.GetFileServiceProperties)
			r.Put("/fileServices/default", h.SetFileServiceProperties)
			r.Patch("/fileServices/default", h.PatchFileServiceProperties)

			// Queue service
			r.Get("/queueServices/default", h.GetQueueServiceProperties)
			r.Put("/queueServices/default", h.SetQueueServiceProperties)
			r.Patch("/queueServices/default", h.PatchQueueServiceProperties)

			// Table service
			r.Get("/tableServices/default", h.GetTableServiceProperties)
			r.Put("/tableServices/default", h.SetTableServiceProperties)
			r.Patch("/tableServices/default", h.PatchTableServiceProperties)
		})
	})
}

func (h *Handler) CreateOrUpdateStorageAccount(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "accountName")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	k := h.storageAccountKey(sub, rg, name)
	acct := h.buildStorageAccountResponse(sub, rg, name)
	if input != nil {
		if tags, ok := input["tags"].(map[string]interface{}); ok {
			acct["tags"] = tags
		}
		if loc, ok := input["location"].(string); ok && loc != "" {
			acct["location"] = loc
		}
		if kind, ok := input["kind"].(string); ok && kind != "" {
			acct["kind"] = kind
		}
		if sku, ok := input["sku"].(map[string]interface{}); ok {
			acct["sku"] = sku
		}
	}
	if acct["tags"] == nil {
		acct["tags"] = map[string]interface{}{}
	}
	h.store.Set(k, acct)
	storageauth.PersistSharedKeyContext(h.store, sub, rg, name)

	// Azure ARM and the azurerm Terraform provider expect 200 OK for PUT create/update,
	// not 201 Created (the Go SDK rejects 201 for this operation).
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(acct)
}

func (h *Handler) GetStorageAccount(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "accountName")

	v, ok := h.store.Get(h.storageAccountKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Storage/storageAccounts", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteStorageAccount(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "accountName")

	if !h.store.Delete(h.storageAccountKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Storage/storageAccounts", name)
		return
	}
	storageauth.DeleteSharedKeyContext(h.store, name)
	w.WriteHeader(http.StatusAccepted)
}

// ListKeys implements POST .../storageAccounts/{accountName}/listKeys (Azure Storage RP).
// azurerm calls this to obtain account keys for the data-plane poller.
func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "accountName")

	if _, ok := h.store.Get(h.storageAccountKey(sub, rg, name)); !ok {
		azerr.NotFound(w, "Microsoft.Storage/storageAccounts", name)
		return
	}

	keys := []map[string]interface{}{
		{"keyName": "key1", "value": storageauth.DeterministicAccountKey(sub, rg, name, "1"), "permissions": "Full"},
		{"keyName": "key2", "value": storageauth.DeterministicAccountKey(sub, rg, name, "2"), "permissions": "Full"},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"keys": keys})
}

func (h *Handler) ListStorageAccounts(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("blob:account:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) ListStorageAccountsInSubscription(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	// Keys are blob:account:{sub}:{rg}:{name}
	items := h.store.ListByPrefix("blob:account:" + sub + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// Private helper functions
func (h *Handler) storageAccountKey(sub, rg, name string) string {
	return "blob:account:" + sub + ":" + rg + ":" + name
}

func writeServiceNotFound(w http.ResponseWriter, resourceType, name string) {
	azerr.NotFound(w, resourceType, name)
}

func (h *Handler) buildServicePropertiesResponse(sub, rg, account, serviceType string) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Storage/storageAccounts/" + account + "/" + serviceType + "/default"

	props := map[string]interface{}{
		"cors": map[string]interface{}{
			"corsRules": []interface{}{},
		},
	}

	switch serviceType {
	case "blobServices":
		props["cors"] = map[string]interface{}{
			"corsRules": []interface{}{},
		}
		props["deleteRetentionPolicy"] = map[string]interface{}{
			"enabled": true,
			"days":    7,
		}
		props["automaticSnapshotPolicyEnabled"] = false
	case "fileServices":
		props["shareDeleteRetentionPolicy"] = map[string]interface{}{
			"enabled": false,
		}
	}

	return map[string]interface{}{
		"id":         id,
		"name":       "default",
		"type":       "Microsoft.Storage/storageAccounts/" + serviceType,
		"properties": props,
	}
}

func (h *Handler) buildStorageAccountResponse(sub, rg, name string) map[string]interface{} {
	var blobEndpoint, queueEndpoint, tableEndpoint, fileEndpoint string
	var localEndpoint = os.Getenv(envStorageEndpoint)

	if localEndpoint != "" {
		blobEndpoint = localEndpoint + "/blob/" + name + "/"
		queueEndpoint = localEndpoint + "/queue/" + name + "/"
		tableEndpoint = localEndpoint + "/table/" + name + "/"
		fileEndpoint = localEndpoint + "/file/" + name + "/"
	} else {
		blobEndpoint = "https://" + name + ".blob.core.windows.net/"
		queueEndpoint = "https://" + name + ".queue.core.windows.net/"
		tableEndpoint = "https://" + name + ".table.core.windows.net/"
		fileEndpoint = "https://" + name + ".file.core.windows.net/"
	}

	return map[string]interface{}{
		"id":       "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Storage/storageAccounts/" + name,
		"name":     name,
		"type":     "Microsoft.Storage/storageAccounts",
		"location": "eastus",
		"sku": map[string]interface{}{
			"name": "Standard_LRS",
			"tier": "Standard",
		},
		"kind": "StorageV2",
		"properties": map[string]interface{}{
			"provisioningState": "Succeeded",
			"primaryEndpoints": map[string]interface{}{
				"blob":  blobEndpoint,
				"queue": queueEndpoint,
				"table": tableEndpoint,
				"file":  fileEndpoint,
			},
			"primaryLocation":          "eastus",
			"statusOfPrimary":          "available",
			"supportsHttpsTrafficOnly": true,
			"creationTime":             "2026-01-01T00:00:00Z",
		},
	}
}
