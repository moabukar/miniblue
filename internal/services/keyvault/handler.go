package keyvault

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

// Secret - A secret consisting of a value, id and its attributes.
type Secret struct {
  // The secret management attributes.
	Attributes      SecretAttributes    `json:"attributes,omitempty"`
  // The content type of the secret.
	ContentType     string              `json:"contentType,omitempty"`
  // The secret id.
	ID              string              `json:"id,omitempty"`
	// The version of the previous certificate, if applicable. Applies only to certificates created after June 1, 2025. Certificates
	// created before this date are not retroactively updated.
  PreviousVersion string              `json:"previousVersion,omitempty"`
  // Application specific metadata in the form of key-value pairs.
	Tags            map[string]string   `json:"tags,omitempty"`
	// The secret value.
  Value           string              `json:"value,omitempty"`
	// READ-ONLY; If this is a secret backing a KV certificate, then this field specifies the corresponding key backing the KV
	// certificate.
	KID             string              `json:"kid,omitempty"`
	// READ-ONLY; True if the secret's lifetime is managed by key vault. If this is a secret backing a certificate, then managed
	// will be true.
	Managed         bool                `json:"managed,omitempty"`
}

// SecretAttributes - The secret management attributes.
type SecretAttributes struct {
  // Determines whether the object is enabled.
	Enabled         bool       `json:"enabled,omitempty"`
  // Expiry date in UTC.
	Expires         int64      `json:"expires,omitempty"`
  // Not before date in UTC.
	NotBefore       int64      `json:"notBefore,omitempty"`
  // READ-ONLY; Creation time in UTC.
	Created         int64      `json:"created,omitempty"`
  // READ-ONLY; softDelete data retention days. Value should be >=7 and <=90 when softDelete enabled, otherwise 0.
	RecoverableDays int32      `json:"recoverableDays,omitempty"`
	// READ-ONLY; Reflects the deletion recovery level currently in effect for secrets in the current vault. If it contains 'Purgeable',
	// the secret can be permanently deleted by a privileged user; otherwise, only the system can purge the secret, at the end
	// of the retention interval.
	RecoveryLevel   string     `json:"recoveryLevel,omitempty"`
  // READ-ONLY; Last updated time in UTC.
	Updated         int64      `json:"updated,omitempty"`
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
    Attributes: SecretAttributes{
      Enabled:         true,
      //Expires:         time.Date(2027, 12, 31, 23, 59, 59, 0, time.UTC),
      //NotBefore:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
      Created:         time.Now().UTC().Unix(),
      //RecoverableDays: 90,
      //RecoveryLevel:   "Recoverable+Purgeable",
      Updated:         time.Now().UTC().Unix(),
    },
    ContentType: "text/plain",
    ID:          "https://" + vault + ".vault.azure.net/secrets/" + name,
    /*Tags: map[string]string{
      "environment": "production",
      "application": "myapp",
    },*/
    Value:   body.Value,
    //Managed: false,
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
	// Azure Key Vault list returns metadata only, NOT the secret value
	redacted := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if s, ok := item.(Secret); ok {
			redacted = append(redacted, map[string]interface{}{
				"id":         s.ID,
				"attributes": s.Attributes,
			})
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"value": redacted})
}
