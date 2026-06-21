package identity

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

// tenantID is the constant emulator tenant.
const tenantID = "00000000-0000-0000-0000-000000000001"

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	ExpiresOn   string `json:"expires_on"`
	Resource    string `json:"resource"`
	ClientID    string `json:"client_id,omitempty"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	// Managed Identity token endpoint (IMDS / App Service style)
	r.Get("/metadata/identity/oauth2/token", h.GetToken)
	// Token introspection for emulated services and tests
	r.Post("/metadata/identity/introspect", h.Introspect)
	// Instance Metadata Service
	r.Get("/metadata/instance", h.GetInstanceMetadata)

	// User-assigned managed identities (ARM resource)
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ManagedIdentity/userAssignedIdentities", func(r chi.Router) {
		r.Get("/", h.ListIdentities)
		r.Route("/{identityName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateIdentity)
			r.Get("/", h.GetIdentity)
			r.Delete("/", h.DeleteIdentity)
		})
	})
}

// ---------------------------------------------------------------------------
// userAssignedIdentities ARM resource
// ---------------------------------------------------------------------------

func identityKey(sub, rg, name string) string {
	return "msi:identity:" + sub + ":" + rg + ":" + name
}

func (h *Handler) CreateOrUpdateIdentity(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "identityName")

	if !h.store.Exists("rg:" + sub + ":" + rg) {
		azerr.WriteError(w, http.StatusNotFound, "ResourceGroupNotFound",
			"Resource group '"+rg+"' could not be found.")
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)
	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	key := identityKey(sub, rg, name)
	principalID := uuid.NewString()
	clientID := uuid.NewString()
	_, exists := h.store.Get(key)
	if exists {
		// principalId/clientId are stable across updates.
		if v, ok := h.store.Get(key); ok {
			if rec, ok := v.(map[string]interface{}); ok {
				if props, ok := rec["properties"].(map[string]interface{}); ok {
					if p, ok := props["principalId"].(string); ok {
						principalID = p
					}
					if c, ok := props["clientId"].(string); ok {
						clientID = c
					}
				}
			}
		}
	}

	rec := map[string]interface{}{
		"id":       "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/" + name,
		"name":     name,
		"type":     "Microsoft.ManagedIdentity/userAssignedIdentities",
		"location": location,
		"properties": map[string]interface{}{
			"principalId": principalID,
			"clientId":    clientID,
			"tenantId":    tenantID,
		},
	}
	if tags, ok := input["tags"]; ok {
		rec["tags"] = tags
	}
	h.store.Set(key, rec)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(rec)
}

func (h *Handler) GetIdentity(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "identityName")

	v, ok := h.store.Get(identityKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.ManagedIdentity/userAssignedIdentities", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) ListIdentities(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("msi:identity:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) DeleteIdentity(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "identityName")

	if !h.store.Delete(identityKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.ManagedIdentity/userAssignedIdentities", name)
		return
	}
	// VMs referencing this identity keep the dangling reference (Azure
	// behavior); token issuance re-validates existence and will reject it.
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Token endpoint
// ---------------------------------------------------------------------------

// oauthError writes an OAuth2-style error (the shape SDK credential chains
// expect from a managed-identity endpoint), as opposed to ARM's envelope.
func oauthError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": description,
	})
}

func (h *Handler) GetToken(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	if resource == "" {
		resource = "https://management.azure.com/"
	}

	secret := r.Header.Get("X-IDENTITY-HEADER")
	if secret == "" {
		// Legacy host-level flow: static token, no VM attestation.
		token := TokenResponse{
			AccessToken: "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsIng1dCI6ImxvY2FsLWF6dXJlIn0.miniblue-mock-token",
			TokenType:   "Bearer",
			ExpiresIn:   86400,
			ExpiresOn:   time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
			Resource:    resource,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(token)
		return
	}

	// VM-attested flow: resolve the VM via its per-VM secret, select among
	// its assigned identities, and issue a token naming both.
	idx, ok := h.store.Get("vmidsecret:" + secret)
	if !ok {
		oauthError(w, http.StatusUnauthorized, "invalid_client", "The identity header does not match any virtual machine.")
		return
	}
	idxRec, _ := idx.(map[string]interface{})
	vmStoreKey, _ := idxRec["vmKey"].(string)
	vmVal, ok := h.store.Get(vmStoreKey)
	if !ok {
		oauthError(w, http.StatusUnauthorized, "invalid_client", "The virtual machine for this identity header no longer exists.")
		return
	}
	vmRec, _ := vmVal.(map[string]interface{})
	vmID, _ := vmRec["id"].(string)

	identityBlock, _ := vmRec["identity"].(map[string]interface{})
	assignments, _ := identityBlock["userAssignedIdentities"].(map[string]interface{})
	if len(assignments) == 0 {
		oauthError(w, http.StatusBadRequest, "invalid_request", "No managed identity assigned to this virtual machine.")
		return
	}

	clientIDParam := r.URL.Query().Get("client_id")
	miResID := r.URL.Query().Get("mi_res_id")

	var armID string
	var entry map[string]interface{}
	switch {
	case miResID != "":
		for k, v := range assignments {
			if strings.EqualFold(k, miResID) {
				armID = k
				entry, _ = v.(map[string]interface{})
			}
		}
		if armID == "" {
			oauthError(w, http.StatusBadRequest, "invalid_request", "The identity '"+miResID+"' is not assigned to this virtual machine.")
			return
		}
	case clientIDParam != "":
		for k, v := range assignments {
			e, _ := v.(map[string]interface{})
			if c, _ := e["clientId"].(string); strings.EqualFold(c, clientIDParam) {
				armID = k
				entry = e
			}
		}
		if armID == "" {
			oauthError(w, http.StatusBadRequest, "invalid_request", "No assigned identity matches client_id '"+clientIDParam+"'.")
			return
		}
	default:
		if len(assignments) > 1 {
			oauthError(w, http.StatusBadRequest, "invalid_request", "Multiple identities are assigned to this virtual machine; specify client_id or mi_res_id.")
			return
		}
		for k, v := range assignments {
			armID = k
			entry, _ = v.(map[string]interface{})
		}
	}

	// The identity must still exist: assignments can dangle after deletion.
	idKey, err := identityKeyFromARMID(armID)
	if err != nil || !h.store.Exists(idKey) {
		oauthError(w, http.StatusBadRequest, "invalid_request", "The managed identity '"+armID+"' no longer exists.")
		return
	}

	principalID, _ := entry["principalId"].(string)
	clientID, _ := entry["clientId"].(string)
	now := time.Now()
	exp := now.Add(24 * time.Hour)

	claims := map[string]interface{}{
		"sub":         principalID,
		"appid":       clientID,
		"xms_mirid":   armID,
		"miniblue_vm": vmID,
		"aud":         resource,
		"iss":         "https://sts.windows.net/" + tenantID + "/",
		"tid":         tenantID,
		"iat":         now.Unix(),
		"exp":         exp.Unix(),
	}
	token := buildMockJWT(claims)

	h.store.Set("msi:token:"+tokenHash(token), map[string]interface{}{
		"identityId": armID,
		"vmId":       vmID,
		"clientId":   clientID,
		"expiresOn":  exp.UTC().Format(time.RFC3339),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   86400,
		ExpiresOn:   exp.UTC().Format(time.RFC3339),
		Resource:    resource,
		ClientID:    clientID,
	})
}

// identityKeyFromARMID converts a userAssignedIdentities ARM id to its store
// key. Kept in sync with the VM package's equivalent parser.
func identityKeyFromARMID(armID string) (string, error) {
	parts := strings.Split(strings.Trim(armID, "/"), "/")
	if len(parts) != 8 {
		return "", fmt.Errorf("'%s' is not a valid userAssignedIdentities resource id", armID)
	}
	return "msi:identity:" + parts[1] + ":" + parts[3] + ":" + parts[7], nil
}

// buildMockJWT assembles an unsigned but introspectable JWT-shaped token:
// real cryptographic signing is intentionally out of scope for the emulator.
func buildMockJWT(claims map[string]interface{}) string {
	enc := func(v interface{}) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	header := map[string]string{"typ": "JWT", "alg": "none", "kid": "miniblue-local"}
	return enc(header) + "." + enc(claims) + ".miniblue-mock-signature"
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ---------------------------------------------------------------------------
// Introspection
// ---------------------------------------------------------------------------

func (h *Handler) Introspect(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Token == "" {
		azerr.BadRequest(w, "The request body must contain a token field.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	v, ok := h.store.Get("msi:token:" + tokenHash(input.Token))
	if !ok {
		json.NewEncoder(w).Encode(map[string]interface{}{"active": false})
		return
	}
	rec, _ := v.(map[string]interface{})
	if expStr, _ := rec["expiresOn"].(string); expStr != "" {
		if exp, err := time.Parse(time.RFC3339, expStr); err == nil && time.Now().After(exp) {
			h.store.Delete("msi:token:" + tokenHash(input.Token))
			json.NewEncoder(w).Encode(map[string]interface{}{"active": false})
			return
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active":     true,
		"identityId": rec["identityId"],
		"vmId":       rec["vmId"],
		"clientId":   rec["clientId"],
		"expiresOn":  rec["expiresOn"],
	})
}

// ---------------------------------------------------------------------------
// Instance metadata
// ---------------------------------------------------------------------------

func (h *Handler) GetInstanceMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := map[string]interface{}{
		"compute": map[string]interface{}{
			"location":          "eastus",
			"name":              "miniblue-vm",
			"resourceGroupName": "miniblue-rg",
			"subscriptionId":    "00000000-0000-0000-0000-000000000000",
			"vmId":              "miniblue-vm-id",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metadata)
}
