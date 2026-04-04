package acr

import (
	"encoding/json"
	"github.com/moabukar/local-azure/internal/azerr"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type Registry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Location   string `json:"location"`
	Properties struct {
		LoginServer string `json:"loginServer"`
	} `json:"properties"`
}

type Manifest struct {
	Tag    string `json:"tag"`
	Digest string `json:"digest"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	// ARM API for registry management
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerRegistry/registries", func(r chi.Router) {
		r.Get("/", h.ListRegistries)
		r.Route("/{registryName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateRegistry)
			r.Get("/", h.GetRegistry)
			r.Delete("/", h.DeleteRegistry)
		})
	})
	// Data plane for manifests
	r.Route("/acr/{registryName}/v2/{repository}/manifests", func(r chi.Router) {
		r.Get("/", h.ListManifests)
		r.Get("/{reference}", h.GetManifest)
	})
	r.Route("/acr/{registryName}/v2/{repository}/tags/list", func(r chi.Router) {
		r.Get("/", h.ListTags)
	})
}

func (h *Handler) registryKey(sub, rg, name string) string {
	return "acr:registry:" + sub + ":" + rg + ":" + name
}

func (h *Handler) CreateOrUpdateRegistry(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "registryName")
	
	var reg Registry
	json.NewDecoder(r.Body).Decode(&reg)
	reg.ID = "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.ContainerRegistry/registries/" + name
	reg.Name = name
	reg.Type = "Microsoft.ContainerRegistry/registries"
	reg.Properties.LoginServer = name + ".azurecr.io"
	
	h.store.Set(h.registryKey(sub, rg, name), reg)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reg)
}

func (h *Handler) GetRegistry(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "registryName")
	
	v, ok := h.store.Get(h.registryKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.ContainerRegistry/registries", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteRegistry(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "registryName")
	h.store.Delete(h.registryKey(sub, rg, name))
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListRegistries(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("acr:registry:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) ListManifests(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"manifests": []Manifest{}})
}

func (h *Handler) GetManifest(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Manifest{Tag: chi.URLParam(r, "reference"), Digest: "sha256:abc123"})
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"name": chi.URLParam(r, "repository"), "tags": []string{}})
}
