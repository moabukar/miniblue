package blob

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/storageauth"
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
	r.Route("/blob/{accountName}", func(r chi.Router) {
		r.Use(h.blobSharedKeyAuth)
		r.Get("/", h.blobAccountRoot)
		r.Head("/", h.blobAccountRoot)
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

func (h *Handler) blobSharedKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		account := chi.URLParam(r, "accountName")
		k1, k2, hasKeys := storageauth.AccountKeyBytes(h.store, account)
		if !hasKeys {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if storageauth.VerifyBlobSharedKey(r, account, k1, k2) {
			next.ServeHTTP(w, r)
			return
		}
		writeBlobAuthFailure(w, r)
	})
}

func writeBlobAuthFailure(w http.ResponseWriter, r *http.Request) {
	reqID := uuid.New().String()
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("x-ms-request-id", reqID)
	w.WriteHeader(http.StatusForbidden)
	msg := "Server failed to authenticate the request. Make sure the value of Authorization header is formed correctly including the signature."
	_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="utf-8" standalone="yes"?>`+
		`<Error><Code>AuthenticationFailed</Code><Message>%s`+
		`RequestId:%s`+
		`Time:%s</Message></Error>`, msg, reqID, time.Now().UTC().Format("2006-01-02T15:04:05.0000000Z"))
}

func (h *Handler) blobAccountRoot(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	comp := q.Get("comp")
	restype := q.Get("restype")

	switch {
	case comp == "properties" && restype == "service":
		h.getBlobServiceProperties(w, r)
	case comp == "list" && (r.Method == http.MethodGet || r.Method == http.MethodHead):
		h.listContainersXML(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) getBlobServiceProperties(w http.ResponseWriter, r *http.Request) {
	h.setBlobResponseHeaders(w, r)
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	const xmlBody = `<?xml version="1.0" encoding="utf-8"?>
<StorageServiceProperties>
  <Logging><Version>1.0</Version><Read>false</Read><Write>false</Write><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></Logging>
  <HourMetrics><Version>1.0</Version><Enabled>false</Enabled><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></HourMetrics>
  <MinuteMetrics><Version>1.0</Version><Enabled>false</Enabled><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></MinuteMetrics>
  <Cors />
  <DefaultServiceVersion>2021-12-02</DefaultServiceVersion>
</StorageServiceProperties>`
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xmlBody))
}

func (h *Handler) setBlobResponseHeaders(w http.ResponseWriter, r *http.Request) {
	if v := r.Header.Get("X-Ms-Version"); v != "" {
		w.Header().Set("x-ms-version", v)
	}
	w.Header().Set("Date", time.Now().UTC().Format(http.TimeFormat))
}

type enumerationResults struct {
	XMLName          xml.Name        `xml:"EnumerationResults"`
	ServiceEndpoint  string          `xml:"ServiceEndpoint,attr"`
	Containers       containersBlock `xml:"Containers"`
}

type containersBlock struct {
	Items []containerXML `xml:"Container"`
}

type containerXML struct {
	Name       string            `xml:"Name"`
	Properties containerPropsXML `xml:"Properties"`
}

type containerPropsXML struct {
	LastModified string `xml:"Last-Modified"`
	Etag         string `xml:"Etag"`
}

func (h *Handler) listContainersXML(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	prefix := "blob:container:" + account + ":"
	items := h.store.ListByPrefix(prefix)

	var names []string
	seen := map[string]bool{}
	for _, v := range items {
		if c, ok := v.(Container); ok && c.Name != "" {
			if !seen[c.Name] {
				seen[c.Name] = true
				names = append(names, c.Name)
			}
		}
	}
	sort.Strings(names)

	h.setBlobResponseHeaders(w, r)
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	endpoint := "https://" + account + ".blob.core.windows.net/"
	res := enumerationResults{
		ServiceEndpoint: endpoint,
		Containers:      containersBlock{Items: make([]containerXML, 0, len(names))},
	}
	now := time.Now().UTC().Format(http.TimeFormat)
	for _, n := range names {
		res.Containers.Items = append(res.Containers.Items, containerXML{
			Name: n,
			Properties: containerPropsXML{
				LastModified: now,
				Etag:         "\"0x8" + fmt.Sprintf("%X", time.Now().UnixNano()) + "\"",
			},
		})
	}
	out, err := xml.MarshalIndent(res, "", "  ")
	if err != nil {
		azerr.WriteError(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(out)
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
	h.store.DeleteByPrefix("blob:blob:" + account + ":" + container + ":")
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
