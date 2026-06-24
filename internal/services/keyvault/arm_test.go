package keyvault

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/store"
)

func newARMTestServer(t *testing.T) http.Handler {
	t.Helper()
	r := chi.NewRouter()
	NewHandler(store.New()).Register(r)
	return r
}

func doJSON(t *testing.T, h http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

const armBase = "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.KeyVault/vaults"

func TestVaultCreateGetDelete(t *testing.T) {
	h := newARMTestServer(t)
	put := map[string]interface{}{
		"location": "westeurope",
		"properties": map[string]interface{}{
			"tenantId": "11111111-2222-3333-4444-555555555555",
			"sku":      map[string]interface{}{"family": "A", "name": "premium"},
			"accessPolicies": []interface{}{
				map[string]interface{}{
					"tenantId": "11111111-2222-3333-4444-555555555555",
					"objectId": "00000000-0000-0000-0000-000000000999",
					"permissions": map[string]interface{}{
						"secrets": []string{"get", "list", "set"},
					},
				},
			},
		},
	}
	r := doJSON(t, h, "PUT", armBase+"/myvault", put)
	if r.Code != http.StatusCreated {
		t.Fatalf("PUT new vault: want 201, got %d – %s", r.Code, r.Body.String())
	}
	var got map[string]interface{}
	if err := json.Unmarshal(r.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["name"] != "myvault" {
		t.Errorf("name: want myvault, got %v", got["name"])
	}
	if got["type"] != "Microsoft.KeyVault/vaults" {
		t.Errorf("type: want Microsoft.KeyVault/vaults, got %v", got["type"])
	}
	props := got["properties"].(map[string]interface{})
	if props["tenantId"] != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("tenantId not preserved: got %v", props["tenantId"])
	}
	if !strings.Contains(props["vaultUri"].(string), "myvault.vault.azure.net") {
		t.Errorf("vaultUri: got %v", props["vaultUri"])
	}
	if props["provisioningState"] != "Succeeded" {
		t.Errorf("provisioningState: got %v", props["provisioningState"])
	}

	if r := doJSON(t, h, "PUT", armBase+"/myvault", put); r.Code != http.StatusOK {
		t.Fatalf("PUT update: want 200, got %d", r.Code)
	}

	if r := doJSON(t, h, "GET", armBase+"/myvault", nil); r.Code != http.StatusOK {
		t.Fatalf("GET: want 200, got %d", r.Code)
	}

	if r := doJSON(t, h, "DELETE", armBase+"/myvault", nil); r.Code != http.StatusOK {
		t.Fatalf("DELETE: want 200, got %d", r.Code)
	}
	if r := doJSON(t, h, "GET", armBase+"/myvault", nil); r.Code != http.StatusNotFound {
		t.Fatalf("GET after delete: want 404, got %d", r.Code)
	}

	r = doJSON(t, h, "GET", "/subscriptions/sub1/providers/Microsoft.KeyVault/deletedVaults", nil)
	if r.Code != http.StatusOK {
		t.Fatalf("list deleted: %d", r.Code)
	}
	var list struct {
		Value []map[string]interface{} `json:"value"`
	}
	_ = json.Unmarshal(r.Body.Bytes(), &list)
	if len(list.Value) != 1 {
		t.Fatalf("want 1 deleted vault, got %d", len(list.Value))
	}
}

func TestVaultPatchMergesProperties(t *testing.T) {
	h := newARMTestServer(t)
	doJSON(t, h, "PUT", armBase+"/patchvault", map[string]interface{}{
		"location": "eastus",
		"properties": map[string]interface{}{
			"enableSoftDelete": true,
		},
	})
	r := doJSON(t, h, "PATCH", armBase+"/patchvault", map[string]interface{}{
		"tags": map[string]string{"env": "prod"},
		"properties": map[string]interface{}{
			"enablePurgeProtection": true,
		},
	})
	if r.Code != http.StatusOK {
		t.Fatalf("PATCH: %d %s", r.Code, r.Body.String())
	}

	r = doJSON(t, h, "GET", armBase+"/patchvault", nil)
	var got map[string]interface{}
	_ = json.Unmarshal(r.Body.Bytes(), &got)
	tags := got["tags"].(map[string]interface{})
	if tags["env"] != "prod" {
		t.Errorf("tags.env merged: got %v", tags["env"])
	}
	props := got["properties"].(map[string]interface{})
	if props["enablePurgeProtection"] != true {
		t.Errorf("enablePurgeProtection merged: got %v", props["enablePurgeProtection"])
	}
	if props["enableSoftDelete"] != true {
		t.Errorf("enableSoftDelete preserved: got %v", props["enableSoftDelete"])
	}
}

func TestVaultListsAtBothScopes(t *testing.T) {
	h := newARMTestServer(t)
	for _, n := range []string{"a-vault", "b-vault"} {
		doJSON(t, h, "PUT",
			"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.KeyVault/vaults/"+n,
			map[string]interface{}{"location": "eastus"})
	}
	doJSON(t, h, "PUT",
		"/subscriptions/sub1/resourceGroups/rg2/providers/Microsoft.KeyVault/vaults/c-vault",
		map[string]interface{}{"location": "eastus"})

	cases := []struct {
		path string
		want int
	}{
		{"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.KeyVault/vaults", 2},
		{"/subscriptions/sub1/resourceGroups/rg2/providers/Microsoft.KeyVault/vaults", 1},
		{"/subscriptions/sub1/providers/Microsoft.KeyVault/vaults", 3},
	}
	for _, tc := range cases {
		r := doJSON(t, h, "GET", tc.path, nil)
		if r.Code != http.StatusOK {
			t.Fatalf("GET %s: %d %s", tc.path, r.Code, r.Body.String())
		}
		var body struct {
			Value []map[string]interface{} `json:"value"`
		}
		_ = json.Unmarshal(r.Body.Bytes(), &body)
		if len(body.Value) != tc.want {
			t.Errorf("list %s: want %d, got %d", tc.path, tc.want, len(body.Value))
		}
	}
}

// TestInvalidVaultNameRejected mirrors the AKS naming-rule guard so PUTs
// that would fail at real Azure also fail here.
func TestInvalidVaultNameRejected(t *testing.T) {
	h := newARMTestServer(t)
	bad := []string{
		"a",                          // too short
		"1startsWithDigit",           // must start with letter
		"-startsWithHyphen",          // must start with letter
		"endsWithHyphen-",            // must end with alphanumeric
		"has_underscore",             // underscores not allowed
		strings.Repeat("a", 25),      // too long
	}
	for _, n := range bad {
		r := doJSON(t, h, "PUT", armBase+"/"+n, map[string]interface{}{"location": "eastus"})
		if r.Code != http.StatusBadRequest {
			t.Errorf("PUT name=%q: want 400, got %d (%s)", n, r.Code, r.Body.String())
		}
	}
	good := []string{"abc", "my-vault-1", strings.Repeat("a", 24)}
	for _, n := range good {
		r := doJSON(t, h, "PUT", armBase+"/"+n, map[string]interface{}{"location": "eastus"})
		if r.Code != http.StatusCreated {
			t.Errorf("PUT name=%q: want 201, got %d (%s)", n, r.Code, r.Body.String())
		}
	}
}

func TestCheckNameAvailability(t *testing.T) {
	h := newARMTestServer(t)
	doJSON(t, h, "PUT", armBase+"/taken-vault", map[string]interface{}{"location": "eastus"})

	check := func(name string) map[string]interface{} {
		r := doJSON(t, h, "POST",
			"/subscriptions/sub1/providers/Microsoft.KeyVault/checkNameAvailability",
			map[string]interface{}{"name": name, "type": "Microsoft.KeyVault/vaults"})
		if r.Code != http.StatusOK {
			t.Fatalf("checkNameAvailability(%q): %d %s", name, r.Code, r.Body.String())
		}
		var body map[string]interface{}
		_ = json.Unmarshal(r.Body.Bytes(), &body)
		return body
	}

	avail := check("free-vault")
	if avail["nameAvailable"] != true {
		t.Errorf("free-vault should be available: %+v", avail)
	}

	taken := check("taken-vault")
	if taken["nameAvailable"] != false {
		t.Errorf("taken-vault should be unavailable: %+v", taken)
	}
	if taken["reason"] != "AlreadyExists" {
		t.Errorf("expected AlreadyExists, got %v", taken["reason"])
	}

	bad := check("a") // too short
	if bad["nameAvailable"] != false {
		t.Errorf("invalid name should be unavailable: %+v", bad)
	}
	if bad["reason"] != "Invalid" {
		t.Errorf("expected Invalid, got %v", bad["reason"])
	}
}

// TestSoftDeleteAndPurge is the round trip Terraform's azurerm_key_vault
// uses when the user toggles purge_protection_enabled / soft_delete.
func TestSoftDeleteAndPurge(t *testing.T) {
	h := newARMTestServer(t)
	doJSON(t, h, "PUT", armBase+"/sd-vault", map[string]interface{}{
		"location": "eastus",
		"properties": map[string]interface{}{
			"enableSoftDelete": true,
		},
	})
	if r := doJSON(t, h, "DELETE", armBase+"/sd-vault", nil); r.Code != http.StatusOK {
		t.Fatalf("DELETE: %d", r.Code)
	}

	r := doJSON(t, h, "GET",
		"/subscriptions/sub1/providers/Microsoft.KeyVault/locations/eastus/deletedVaults/sd-vault", nil)
	if r.Code != http.StatusOK {
		t.Fatalf("GET deletedVault: %d %s", r.Code, r.Body.String())
	}

	r = doJSON(t, h, "POST",
		"/subscriptions/sub1/providers/Microsoft.KeyVault/locations/eastus/deletedVaults/sd-vault/purge", nil)
	if r.Code != http.StatusOK {
		t.Fatalf("purge: %d", r.Code)
	}

	r = doJSON(t, h, "GET",
		"/subscriptions/sub1/providers/Microsoft.KeyVault/locations/eastus/deletedVaults/sd-vault", nil)
	if r.Code != http.StatusNotFound {
		t.Fatalf("GET after purge: want 404, got %d", r.Code)
	}
}

func TestUpdateVaultAccessPolicy(t *testing.T) {
	h := newARMTestServer(t)
	doJSON(t, h, "PUT", armBase+"/p-vault", map[string]interface{}{"location": "eastus"})

	add := map[string]interface{}{
		"properties": map[string]interface{}{
			"accessPolicies": []interface{}{
				map[string]interface{}{
					"tenantId": "11111111-2222-3333-4444-555555555555",
					"objectId": "deadbeef-cafe-d00d-cafe-deadbeefcafe",
					"permissions": map[string]interface{}{
						"secrets": []string{"get"},
					},
				},
			},
		},
	}
	r := doJSON(t, h, "PUT", armBase+"/p-vault/accessPolicies/add", add)
	if r.Code != http.StatusOK {
		t.Fatalf("add policy: %d %s", r.Code, r.Body.String())
	}

	r = doJSON(t, h, "GET", armBase+"/p-vault", nil)
	var got map[string]interface{}
	_ = json.Unmarshal(r.Body.Bytes(), &got)
	props := got["properties"].(map[string]interface{})
	policies := props["accessPolicies"].([]interface{})
	if len(policies) != 1 {
		t.Fatalf("want 1 access policy, got %d (%v)", len(policies), policies)
	}
}

func TestOperationsList(t *testing.T) {
	h := newARMTestServer(t)
	r := doJSON(t, h, "GET", "/providers/Microsoft.KeyVault/operations", nil)
	if r.Code != http.StatusOK {
		t.Fatalf("operations list: %d", r.Code)
	}
	var body struct {
		Value []map[string]interface{} `json:"value"`
	}
	_ = json.Unmarshal(r.Body.Bytes(), &body)
	if len(body.Value) == 0 {
		t.Fatalf("expected non-empty operations list")
	}
	for _, op := range body.Value {
		name, _ := op["name"].(string)
		if !strings.HasPrefix(name, "Microsoft.KeyVault/") {
			t.Errorf("operation name should be prefixed with Microsoft.KeyVault/: %q", name)
		}
	}
}
