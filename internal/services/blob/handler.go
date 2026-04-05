package blob

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

type Container struct {
	Name       string            `json:"name"`
	Properties map[string]string `json:"properties"`
}

type Blob struct {
	Name       string            `json:"name"`
	Properties map[string]string `json:"properties"`
	Content    []byte            `json:"-"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	// ARM-style paths: used by Azure SDKs to enumerate and manage storage accounts
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts", func(r chi.Router) {
		r.Get("/", h.ListStorageAccounts)
		r.Route("/{accountName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateStorageAccount)
			r.Get("/", h.GetStorageAccount)
			r.Delete("/", h.DeleteStorageAccount)
			r.Route("/blobServices/default/containers", func(r chi.Router) {
				r.Get("/", h.ListContainersARM)
				r.Route("/{containerName}", func(r chi.Router) {
					r.Put("/", h.CreateContainerARM)
					r.Get("/", h.GetContainerARM)
					r.Delete("/", h.DeleteContainerARM)
				})
			})
		})
	})

	// Data-plane paths: used for blob operations
	r.Route("/blob/{accountName}", func(r chi.Router) {
		r.Route("/{containerName}", func(r chi.Router) {
			r.Put("/", h.CreateContainer)
			r.Get("/", h.ListBlobs)
			r.Delete("/", h.DeleteContainer)
			r.Route("/{blobName}", func(r chi.Router) {
				r.Put("/", h.UploadBlob)
				r.Get("/", h.DownloadBlob)
				r.Delete("/", h.DeleteBlob)
			})
		})
	})
}

func (h *Handler) containerKey(account, container string) string {
	return "blob:container:" + account + ":" + container
}

func (h *Handler) blobKey(account, container, blob string) string {
	return "blob:blob:" + account + ":" + container + ":" + blob
}

func (h *Handler) CreateContainer(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	container := chi.URLParam(r, "containerName")

	c := Container{
		Name: container,
		Properties: map[string]string{
			"lastModified": time.Now().UTC().Format(time.RFC1123),
			"etag":         fmt.Sprintf("\"0x%X\"", time.Now().UnixNano()),
		},
	}
	h.store.Set(h.containerKey(account, container), c)
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) DeleteContainer(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	container := chi.URLParam(r, "containerName")
	h.store.Delete(h.containerKey(account, container))
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListBlobs(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	container := chi.URLParam(r, "containerName")
	prefix := "blob:blob:" + account + ":" + container + ":"
	items := h.store.ListByPrefix(prefix)
	json.NewEncoder(w).Encode(map[string]interface{}{"blobs": items})
}

func (h *Handler) UploadBlob(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	container := chi.URLParam(r, "containerName")
	blobName := chi.URLParam(r, "blobName")

	data, _ := io.ReadAll(r.Body)
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	b := Blob{
		Name: blobName,
		Properties: map[string]string{
			"lastModified":  time.Now().UTC().Format(time.RFC1123),
			"contentLength": fmt.Sprintf("%d", len(data)),
			"contentType":   ct,
			"etag":          fmt.Sprintf("\"0x%X\"", time.Now().UnixNano()),
		},
		Content: data,
	}
	h.store.Set(h.blobKey(account, container, blobName), b)
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) DownloadBlob(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	container := chi.URLParam(r, "containerName")
	blobName := chi.URLParam(r, "blobName")

	v, ok := h.store.Get(h.blobKey(account, container, blobName))
	if !ok {
		azerr.NotFound(w, "blob", blobName)
		return
	}
	b := v.(Blob)
	w.Header().Set("Content-Type", b.Properties["contentType"])
	w.Header().Set("Content-Length", b.Properties["contentLength"])
	w.Header().Set("ETag", b.Properties["etag"])
	w.Write(b.Content)
}

func (h *Handler) DeleteBlob(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	container := chi.URLParam(r, "containerName")
	blobName := chi.URLParam(r, "blobName")
	h.store.Delete(h.blobKey(account, container, blobName))
	w.WriteHeader(http.StatusAccepted)
}

// --- ARM storage account handlers ---

func (h *Handler) storageAccountKey(sub, rg, name string) string {
	return "blob:account:" + sub + ":" + rg + ":" + name
}

func (h *Handler) buildStorageAccountResponse(sub, rg, name string) map[string]interface{} {
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
			"provisioningState":        "Succeeded",
			"primaryEndpoints": map[string]interface{}{
				"blob":  "https://" + name + ".blob.core.windows.net/",
				"queue": "https://" + name + ".queue.core.windows.net/",
				"table": "https://" + name + ".table.core.windows.net/",
				"file":  "https://" + name + ".file.core.windows.net/",
			},
			"primaryLocation":          "eastus",
			"statusOfPrimary":          "available",
			"supportsHttpsTrafficOnly": true,
			"creationTime":             "2026-01-01T00:00:00Z",
		},
	}
}

func (h *Handler) CreateOrUpdateStorageAccount(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "accountName")

	k := h.storageAccountKey(sub, rg, name)
	_, exists := h.store.Get(k)

	acct := h.buildStorageAccountResponse(sub, rg, name)
	h.store.Set(k, acct)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
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
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListStorageAccounts(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("blob:account:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// --- ARM blob container handlers ---

func (h *Handler) armContainerKey(sub, rg, account, name string) string {
	return "blob:armcontainer:" + sub + ":" + rg + ":" + account + ":" + name
}

func (h *Handler) buildARMContainerResponse(sub, rg, account, name string) map[string]interface{} {
	return map[string]interface{}{
		"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Storage/storageAccounts/" + account + "/blobServices/default/containers/" + name,
		"name": name,
		"type": "Microsoft.Storage/storageAccounts/blobServices/containers",
		"properties": map[string]interface{}{
			"publicAccess":      "None",
			"leaseStatus":       "Unlocked",
			"leaseState":        "Available",
			"lastModifiedTime":  fmt.Sprintf("\"0x%X\"", time.Now().UnixNano()),
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
	_, exists := h.store.Get(k)

	c := h.buildARMContainerResponse(sub, rg, account, name)
	h.store.Set(k, c)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(c)
}

func (h *Handler) GetContainerARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	account := chi.URLParam(r, "accountName")
	name := chi.URLParam(r, "containerName")

	v, ok := h.store.Get(h.armContainerKey(sub, rg, account, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Storage/storageAccounts/blobServices/containers", name)
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
		azerr.NotFound(w, "Microsoft.Storage/storageAccounts/blobServices/containers", name)
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
