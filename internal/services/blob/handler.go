package blob

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/azerr"
	"github.com/moabukar/local-azure/internal/store"
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
