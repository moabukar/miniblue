package keyvault

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
)

// Azure Key Vault naming rule: 3-24 alphanumerics-or-hyphens, must start
// with a letter, must end with letter or digit, no consecutive hyphens.
// Reject early so PUTs that would fail at real Azure also fail here.
var validVaultName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{1,22}[a-zA-Z0-9]$`).MatchString

// registerARM mounts the Microsoft.KeyVault ARM management surface.
// Mirrors the case-insensitive resourceGroups handling used in
// internal/services/aks/handler.go.
func (h *Handler) registerARM(r chi.Router) {
	for _, rgSeg := range []string{"resourcegroups", "resourceGroups"} {
		base := "/subscriptions/{subscriptionId}/" + rgSeg + "/{resourceGroupName}/providers/Microsoft.KeyVault"
		r.Route(base+"/vaults", func(r chi.Router) {
			r.Get("/", h.ListVaultsByRG)
			r.Route("/{vaultName}", func(r chi.Router) {
				r.Put("/", h.CreateOrUpdateVault)
				r.Patch("/", h.UpdateVault)
				r.Get("/", h.GetVault)
				r.Delete("/", h.DeleteVault)
				r.Put("/accessPolicies/{operationKind}", h.UpdateVaultAccessPolicy)
			})
		})
	}

	// Subscription-scoped operations.
	r.Get("/subscriptions/{subscriptionId}/providers/Microsoft.KeyVault/vaults", h.ListVaultsBySubscription)
	r.Post("/subscriptions/{subscriptionId}/providers/Microsoft.KeyVault/checkNameAvailability", h.CheckNameAvailability)
	r.Get("/subscriptions/{subscriptionId}/providers/Microsoft.KeyVault/deletedVaults", h.ListDeletedVaults)
	r.Get("/subscriptions/{subscriptionId}/providers/Microsoft.KeyVault/locations/{location}/deletedVaults/{vaultName}", h.GetDeletedVault)
	r.Post("/subscriptions/{subscriptionId}/providers/Microsoft.KeyVault/locations/{location}/deletedVaults/{vaultName}/purge", h.PurgeDeletedVault)

	// Provider-scoped operations metadata.
	r.Get("/providers/Microsoft.KeyVault/operations", h.ListOperationsMetadata)
}

// vaultKey returns the store key for an active vault.
func vaultKey(sub, rg, name string) string {
	return "kv:vault:" + sub + ":" + rg + ":" + name
}

func vaultPrefixRG(sub, rg string) string {
	return "kv:vault:" + sub + ":" + rg + ":"
}

func vaultPrefixSub(sub string) string {
	return "kv:vault:" + sub + ":"
}

// deletedVaultKey is set when a soft-delete-enabled vault is removed; the
// purge endpoint clears it. Real Azure also indexes by location, but
// miniblue keeps a single bucket per (sub, name) since regions are cosmetic.
func deletedVaultKey(sub, name string) string {
	return "kv:deleted:" + sub + ":" + name
}

// CreateOrUpdateVault implements PUT .../Microsoft.KeyVault/vaults/{vaultName}.
//
// In real Azure this is a long-running operation. The Azure Go SDK will
// happily accept a synchronous response, so miniblue persists the resource
// immediately and returns 201 (create) or 200 (update) the same way the
// AKS handler does — see internal/services/aks/handler.go.
func (h *Handler) CreateOrUpdateVault(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "vaultName")

	if !validVaultName(name) {
		azerr.BadRequest(w, "Vault name must be 3-24 chars, start with a letter, end with letter or digit, contain only letters/digits/hyphens.")
		return
	}

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		azerr.BadRequest(w, "Could not parse request body: "+err.Error())
		return
	}

	key := vaultKey(sub, rg, name)
	_, exists := h.store.Get(key)

	vault := buildVaultResponse(sub, rg, name, input)
	h.store.Set(key, vault)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(vault)
}

// UpdateVault implements PATCH .../vaults/{vaultName}. Per the spec, only a
// subset of properties may be patched. Miniblue applies any tags/properties
// supplied and re-serializes the resource.
func (h *Handler) UpdateVault(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "vaultName")

	v, ok := h.store.Get(vaultKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.KeyVault/vaults", name)
		return
	}
	current, _ := v.(map[string]interface{})
	if current == nil {
		azerr.NotFound(w, "Microsoft.KeyVault/vaults", name)
		return
	}

	var patch map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		azerr.BadRequest(w, "Could not parse request body: "+err.Error())
		return
	}

	if tags, ok := patch["tags"].(map[string]interface{}); ok {
		current["tags"] = tags
	}
	if patchProps, ok := patch["properties"].(map[string]interface{}); ok {
		curProps, _ := current["properties"].(map[string]interface{})
		if curProps == nil {
			curProps = map[string]interface{}{}
		}
		for k, val := range patchProps {
			curProps[k] = val
		}
		current["properties"] = curProps
	}

	h.store.Set(vaultKey(sub, rg, name), current)
	_ = json.NewEncoder(w).Encode(current)
}

// GetVault implements GET .../vaults/{vaultName}.
func (h *Handler) GetVault(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "vaultName")

	v, ok := h.store.Get(vaultKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.KeyVault/vaults", name)
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// DeleteVault implements DELETE .../vaults/{vaultName}.
//
// Soft-delete is on by default in modern Azure: deletes move the vault into
// a "deletedVaults" bucket from which it can be purged or recovered.
// miniblue mirrors that contract so Terraform's purge_protection_enabled
// flow and the Go SDK's recovery polling work locally.
func (h *Handler) DeleteVault(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "vaultName")

	v, ok := h.store.Get(vaultKey(sub, rg, name))
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if vmap, _ := v.(map[string]interface{}); vmap != nil {
		if softDeleteEnabled(vmap) {
			h.store.Set(deletedVaultKey(sub, name), buildDeletedVaultResponse(sub, name, vmap))
		}
	}
	h.store.Delete(vaultKey(sub, rg, name))
	w.WriteHeader(http.StatusOK)
}

// ListVaultsByRG implements GET .../resourceGroups/{rg}/providers/Microsoft.KeyVault/vaults.
func (h *Handler) ListVaultsByRG(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix(vaultPrefixRG(sub, rg))
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// ListVaultsBySubscription implements GET /subscriptions/{}/providers/Microsoft.KeyVault/vaults.
func (h *Handler) ListVaultsBySubscription(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	items := h.store.ListByPrefix(vaultPrefixSub(sub))
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// CheckNameAvailability implements POST /subscriptions/{}/providers/Microsoft.KeyVault/checkNameAvailability.
//
// Real Azure verifies the name is globally unique. Miniblue treats every
// name as available unless the same name is currently registered in any
// resource group inside the subscription.
func (h *Handler) CheckNameAvailability(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	var input struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		azerr.BadRequest(w, "Could not parse request body: "+err.Error())
		return
	}
	if input.Name == "" {
		azerr.BadRequest(w, "name is required")
		return
	}

	available := true
	reason := ""
	message := ""
	if !validVaultName(input.Name) {
		available = false
		reason = "Invalid"
		message = "The vault name does not match Azure naming requirements."
	} else {
		for _, item := range h.store.ListByPrefix(vaultPrefixSub(sub)) {
			vm, _ := item.(map[string]interface{})
			if vm == nil {
				continue
			}
			if n, _ := vm["name"].(string); strings.EqualFold(n, input.Name) {
				available = false
				reason = "AlreadyExists"
				message = "The vault name is already in use."
				break
			}
		}
	}
	resp := map[string]interface{}{"nameAvailable": available}
	if !available {
		resp["reason"] = reason
		resp["message"] = message
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// ListDeletedVaults implements GET /subscriptions/{}/providers/Microsoft.KeyVault/deletedVaults.
func (h *Handler) ListDeletedVaults(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	items := h.store.ListByPrefix("kv:deleted:" + sub + ":")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// GetDeletedVault implements GET .../locations/{location}/deletedVaults/{vaultName}.
func (h *Handler) GetDeletedVault(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "vaultName")
	v, ok := h.store.Get(deletedVaultKey(sub, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.KeyVault/deletedVaults", name)
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// PurgeDeletedVault implements POST .../deletedVaults/{vaultName}/purge.
// Real Azure runs this as an LRO; miniblue clears the entry synchronously
// and returns 200 OK.
func (h *Handler) PurgeDeletedVault(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	name := chi.URLParam(r, "vaultName")
	h.store.Delete(deletedVaultKey(sub, name))
	w.WriteHeader(http.StatusOK)
}

// UpdateVaultAccessPolicy implements PUT .../vaults/{vaultName}/accessPolicies/{operationKind}.
// operationKind is "add" | "replace" | "remove". Miniblue applies the
// supplied access policies onto the vault and returns the updated set.
func (h *Handler) UpdateVaultAccessPolicy(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "vaultName")
	kind := chi.URLParam(r, "operationKind")

	v, ok := h.store.Get(vaultKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.KeyVault/vaults", name)
		return
	}
	current, _ := v.(map[string]interface{})
	if current == nil {
		azerr.NotFound(w, "Microsoft.KeyVault/vaults", name)
		return
	}

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		azerr.BadRequest(w, "Could not parse request body: "+err.Error())
		return
	}
	props, _ := current["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	supplied, _ := input["properties"].(map[string]interface{})
	if supplied == nil {
		supplied = map[string]interface{}{}
	}
	policies, _ := supplied["accessPolicies"].([]interface{})

	switch kind {
	case "replace":
		props["accessPolicies"] = policies
	case "remove":
		props["accessPolicies"] = []interface{}{}
	default:
		// "add" or anything unknown: append.
		existing, _ := props["accessPolicies"].([]interface{})
		props["accessPolicies"] = append(existing, policies...)
	}
	current["properties"] = props
	h.store.Set(vaultKey(sub, rg, name), current)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"properties": map[string]interface{}{
			"accessPolicies": props["accessPolicies"],
		},
	})
}

// ListOperationsMetadata implements GET /providers/Microsoft.KeyVault/operations.
// Returns the canonical Azure operations metadata that the resource
// provider describes for itself. The set we return is small but enough
// for `az provider operation list` and azurerm role validation paths to
// not blow up.
func (h *Handler) ListOperationsMetadata(w http.ResponseWriter, r *http.Request) {
	ops := []map[string]interface{}{
		{
			"name":         "Microsoft.KeyVault/vaults/read",
			"display":      kvOpDisplay("Read Key Vault"),
			"isDataAction": false,
		},
		{
			"name":         "Microsoft.KeyVault/vaults/write",
			"display":      kvOpDisplay("Create or Update Key Vault"),
			"isDataAction": false,
		},
		{
			"name":         "Microsoft.KeyVault/vaults/delete",
			"display":      kvOpDisplay("Delete Key Vault"),
			"isDataAction": false,
		},
		{
			"name":         "Microsoft.KeyVault/vaults/secrets/read",
			"display":      kvOpDisplay("View Secret"),
			"isDataAction": true,
		},
		{
			"name":         "Microsoft.KeyVault/vaults/secrets/write",
			"display":      kvOpDisplay("Create or Update Secret"),
			"isDataAction": true,
		},
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": ops})
}

func kvOpDisplay(operation string) map[string]interface{} {
	return map[string]interface{}{
		"provider":  "Microsoft Key Vault",
		"resource":  "Key Vault",
		"operation": operation,
	}
}

// buildVaultResponse builds the ARM JSON for a vault from the PUT body.
// Anything missing in the input gets a sensible default so a minimal
// `azurerm_key_vault` block round-trips cleanly.
func buildVaultResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.KeyVault/vaults/" + name

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	tags, _ := input["tags"].(map[string]interface{})

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	tenantID, _ := props["tenantId"].(string)
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	sku, _ := props["sku"].(map[string]interface{})
	if sku == nil {
		sku = map[string]interface{}{
			"family": "A",
			"name":   "standard",
		}
	}

	accessPolicies, _ := props["accessPolicies"].([]interface{})
	if accessPolicies == nil {
		accessPolicies = []interface{}{}
	}

	enableSoftDelete := boolOrDefault(props["enableSoftDelete"], true)
	softDeleteRetentionDays := intOrDefault(props["softDeleteRetentionInDays"], 90)

	resp := map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.KeyVault/vaults",
		"location": location,
		"properties": map[string]interface{}{
			"tenantId":                     tenantID,
			"sku":                          sku,
			"accessPolicies":               accessPolicies,
			"enabledForDeployment":         boolOrDefault(props["enabledForDeployment"], false),
			"enabledForDiskEncryption":     boolOrDefault(props["enabledForDiskEncryption"], false),
			"enabledForTemplateDeployment": boolOrDefault(props["enabledForTemplateDeployment"], false),
			"enableSoftDelete":             enableSoftDelete,
			"softDeleteRetentionInDays":    softDeleteRetentionDays,
			"enableRbacAuthorization":      boolOrDefault(props["enableRbacAuthorization"], false),
			"enablePurgeProtection":        boolOrDefault(props["enablePurgeProtection"], false),
			"vaultUri":                     "https://" + name + ".vault.azure.net/",
			"provisioningState":            "Succeeded",
			"publicNetworkAccess":          stringOrDefault(props["publicNetworkAccess"], "Enabled"),
			"hsmPoolResourceId":            nil,
			"createMode":                   stringOrDefault(props["createMode"], "default"),
			"networkAcls":                  props["networkAcls"],
		},
	}
	if tags != nil {
		resp["tags"] = tags
	}
	return resp
}

// buildDeletedVaultResponse produces the body returned by GET deletedVaults.
func buildDeletedVaultResponse(sub, name string, original map[string]interface{}) map[string]interface{} {
	location, _ := original["location"].(string)
	props, _ := original["properties"].(map[string]interface{})
	tenantID := ""
	vaultURI := ""
	purgeProtection := false
	if props != nil {
		if v, ok := props["tenantId"].(string); ok {
			tenantID = v
		}
		if v, ok := props["vaultUri"].(string); ok {
			vaultURI = v
		}
		purgeProtection = boolOrDefault(props["enablePurgeProtection"], false)
	}
	return map[string]interface{}{
		"id":   "/subscriptions/" + sub + "/providers/Microsoft.KeyVault/locations/" + location + "/deletedVaults/" + name,
		"name": name,
		"type": "Microsoft.KeyVault/deletedVaults",
		"properties": map[string]interface{}{
			"vaultId":                       original["id"],
			"location":                      location,
			"tenantId":                      tenantID,
			"vaultUri":                      vaultURI,
			"deletionDate":                  time.Now().UTC().Format(time.RFC3339),
			"scheduledPurgeDate":            time.Now().UTC().AddDate(0, 0, 90).Format(time.RFC3339),
			"purgeProtectionEnabled":        purgeProtection,
		},
	}
}

func softDeleteEnabled(vault map[string]interface{}) bool {
	props, _ := vault["properties"].(map[string]interface{})
	if props == nil {
		return true
	}
	return boolOrDefault(props["enableSoftDelete"], true)
}

func boolOrDefault(v interface{}, dflt bool) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return dflt
}

func intOrDefault(v interface{}, dflt int) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return dflt
}

func stringOrDefault(v interface{}, dflt string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return dflt
}
