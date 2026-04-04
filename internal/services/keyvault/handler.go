package keyvault

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/azerr"
	"github.com/moabukar/local-azure/internal/store"
)

type Secret struct {
	ID         string            `json:"id"`
	Value      string            `json:"value"`
	Attributes map[string]string `json:"attributes"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/keyvault/{vaultName}/secrets", func(r chi.Router) {
		r.Get("/", h.ListSecrets)
		r.Route("/{secretName}", func(r chi.Router) {
			r.Put("/", h.SetSecret)
			r.Get("/", h.GetSecret)
			r.Delete("/", h.DeleteSecret)
		})
	})
}

func (h *Handler) key(vault, name string) string {
	return "kv:" + vault + ":" + name
}

func (h *Handler) SetSecret(w http.ResponseWriter, r *http.Request) {
	vault := chi.URLParam(r, "vaultName")
	name := chi.URLParam(r, "secretName")

	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		azerr.BadRequest(w, "Invalid request body: "+err.Error())
		return
	}

	secret := Secret{
		ID:    "https://" + vault + ".vault.azure.net/secrets/" + name,
		Value: body.Value,
		Attributes: map[string]string{
			"enabled": "true",
			"created": time.Now().UTC().Format(time.RFC3339),
			"updated": time.Now().UTC().Format(time.RFC3339),
		},
	}

	h.store.Set(h.key(vault, name), secret)
	json.NewEncoder(w).Encode(secret)
}

func (h *Handler) GetSecret(w http.ResponseWriter, r *http.Request) {
	vault := chi.URLParam(r, "vaultName")
	name := chi.URLParam(r, "secretName")

	v, ok := h.store.Get(h.key(vault, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.KeyVault/vaults/secrets", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteSecret(w http.ResponseWriter, r *http.Request) {
	vault := chi.URLParam(r, "vaultName")
	name := chi.URLParam(r, "secretName")
	h.store.Delete(h.key(vault, name))
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListSecrets(w http.ResponseWriter, r *http.Request) {
	vault := chi.URLParam(r, "vaultName")
	items := h.store.ListByPrefix("kv:" + vault + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}
