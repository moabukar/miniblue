package tests

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/services/identity"
	"github.com/moabukar/miniblue/internal/store"
)

// ---------------------------------------------------------------------------
// userAssignedIdentities ARM resource (HTTP level, stub mode)
// ---------------------------------------------------------------------------

func identityURL(ts *httptest.Server, sub, rg, name string) string {
	return ts.URL + "/subscriptions/" + sub + "/resourceGroups/" + rg +
		"/providers/Microsoft.ManagedIdentity/userAssignedIdentities/" + name
}

func TestIdentityCRUD(t *testing.T) {
	ts := setupVMServer(t)
	defer ts.Close()
	createRG(t, ts, "sub1", "rg1")

	resp := doRequest(t, "PUT", identityURL(ts, "sub1", "rg1", "app-id")+vmAV, `{"location":"eastus"}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()
	props := m["properties"].(map[string]interface{})
	principalID, _ := props["principalId"].(string)
	clientID, _ := props["clientId"].(string)
	if principalID == "" || clientID == "" {
		t.Fatalf("expected generated principalId/clientId, got %v / %v", principalID, clientID)
	}

	// principalId/clientId must be stable across updates.
	resp = doRequest(t, "PUT", identityURL(ts, "sub1", "rg1", "app-id")+vmAV, `{"location":"westus"}`)
	expectStatus(t, resp, 200)
	m = decodeJSON(t, resp)
	resp.Body.Close()
	props = m["properties"].(map[string]interface{})
	if props["principalId"] != principalID || props["clientId"] != clientID {
		t.Fatal("principalId/clientId changed on update")
	}

	resp = doRequest(t, "GET", ts.URL+"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentities"+vmAV, "")
	expectStatus(t, resp, 200)
	m = decodeJSON(t, resp)
	resp.Body.Close()
	if got := len(m["value"].([]interface{})); got != 1 {
		t.Fatalf("expected 1 identity, got %d", got)
	}

	resp = doRequest(t, "DELETE", identityURL(ts, "sub1", "rg1", "app-id")+vmAV, "")
	expectStatus(t, resp, 204)
	resp.Body.Close()
	resp = doRequest(t, "GET", identityURL(ts, "sub1", "rg1", "app-id")+vmAV, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
}

func TestIdentityCreateMissingResourceGroup(t *testing.T) {
	ts := setupVMServer(t)
	defer ts.Close()

	resp := doRequest(t, "PUT", identityURL(ts, "sub1", "nope", "app-id")+vmAV, `{"location":"eastus"}`)
	expectStatus(t, resp, 404)
	e := decodeError(t, resp)
	resp.Body.Close()
	if e.Error.Code != "ResourceGroupNotFound" {
		t.Fatalf("expected ResourceGroupNotFound, got %s", e.Error.Code)
	}
}

// ---------------------------------------------------------------------------
// VM identity assignment (HTTP level, stub mode)
// ---------------------------------------------------------------------------

func TestVMIdentityAssignment(t *testing.T) {
	ts := setupVMServer(t)
	defer ts.Close()
	createRG(t, ts, "sub1", "rg1")

	armID := "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentities/app-id"

	// Unknown identity → 400.
	resp := doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "web")+vmAV,
		`{"location":"eastus","identity":{"type":"UserAssigned","userAssignedIdentities":{"`+armID+`":{}}}}`)
	expectStatus(t, resp, 400)
	resp.Body.Close()

	resp = doRequest(t, "PUT", identityURL(ts, "sub1", "rg1", "app-id")+vmAV, `{"location":"eastus"}`)
	expectStatus(t, resp, 201)
	resp.Body.Close()

	// Known identity → resolved principalId/clientId in the VM response.
	resp = doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "web")+vmAV,
		`{"location":"eastus","identity":{"type":"UserAssigned","userAssignedIdentities":{"`+armID+`":{}}}}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()
	block := m["identity"].(map[string]interface{})
	entry := block["userAssignedIdentities"].(map[string]interface{})[armID].(map[string]interface{})
	if pid, _ := entry["principalId"].(string); pid == "" {
		t.Fatal("expected resolved principalId in identity assignment")
	}

	// "None" clears the assignments.
	resp = doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "web")+vmAV,
		`{"location":"eastus","identity":{"type":"None"}}`)
	expectStatus(t, resp, 200)
	m = decodeJSON(t, resp)
	resp.Body.Close()
	if _, has := m["identity"]; has {
		t.Fatal("expected identity block cleared after type None")
	}
}

// ---------------------------------------------------------------------------
// VM-attested token flow (handler level)
//
// The per-VM secret is internal and never returned by the API: these tests
// seed the store directly — exactly the state CreateOrUpdateVM writes — and
// drive the identity handler over HTTP the same way an in-VM workload would
// (X-IDENTITY-HEADER + query parameters). The full in-container path is
// validated by the quickstart against a real Docker daemon.
// ---------------------------------------------------------------------------

const (
	tokSub    = "sub1"
	tokRG     = "rg1"
	tokSecret = "test-vm-secret"
	tokVMID   = "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/web"
)

func tokenTestServer(t *testing.T, assignments map[string]interface{}) (*httptest.Server, *store.Store) {
	t.Helper()
	s := store.New()

	vmRec := map[string]interface{}{
		"id":   tokVMID,
		"name": "web",
		"type": "Microsoft.Compute/virtualMachines",
	}
	if assignments != nil {
		vmRec["identity"] = map[string]interface{}{
			"type":                   "UserAssigned",
			"userAssignedIdentities": assignments,
		}
	}
	s.Set("vm:"+tokSub+":"+tokRG+":web", vmRec)
	s.Set("vmidsecret:"+tokSecret, map[string]interface{}{"vmKey": "vm:" + tokSub + ":" + tokRG + ":web"})

	r := chi.NewRouter()
	identity.NewHandler(s).Register(r)
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	return ts, s
}

func seedIdentity(s *store.Store, name, principalID, clientID string) string {
	armID := "/subscriptions/" + tokSub + "/resourceGroups/" + tokRG +
		"/providers/Microsoft.ManagedIdentity/userAssignedIdentities/" + name
	s.Set("msi:identity:"+tokSub+":"+tokRG+":"+name, map[string]interface{}{
		"id":   armID,
		"name": name,
		"properties": map[string]interface{}{
			"principalId": principalID,
			"clientId":    clientID,
		},
	})
	return armID
}

func requestToken(t *testing.T, ts *httptest.Server, secret, query string) (map[string]interface{}, int) {
	t.Helper()
	url := ts.URL + "/metadata/identity/oauth2/token"
	if query != "" {
		url += "?" + query
	}
	req, _ := http.NewRequest("GET", url, nil)
	if secret != "" {
		req.Header.Set("X-IDENTITY-HEADER", secret)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var m map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&m)
	return m, resp.StatusCode
}

func decodeJWTClaims(t *testing.T, token string) map[string]interface{} {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected a JWT-shaped token, got %q", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("failed to decode claims: %v", err)
	}
	var claims map[string]interface{}
	json.Unmarshal(payload, &claims)
	return claims
}

func TestTokenSingleIdentity(t *testing.T) {
	armID := "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentities/app-id"
	ts, st := tokenTestServer(t, map[string]interface{}{
		armID: map[string]interface{}{"principalId": "principal-1", "clientId": "client-1"},
	})
	seedIdentity(st, "app-id", "principal-1", "client-1")

	m, status := requestToken(t, ts, tokSecret, "resource=https://vault.local/")
	if status != 200 {
		t.Fatalf("expected 200, got %d: %v", status, m)
	}
	if m["client_id"] != "client-1" {
		t.Fatalf("expected client_id client-1, got %v", m["client_id"])
	}
	claims := decodeJWTClaims(t, m["access_token"].(string))
	if claims["xms_mirid"] != armID {
		t.Fatalf("expected xms_mirid=%s, got %v", armID, claims["xms_mirid"])
	}
	if claims["miniblue_vm"] != tokVMID {
		t.Fatalf("expected miniblue_vm=%s, got %v", tokVMID, claims["miniblue_vm"])
	}
	if claims["sub"] != "principal-1" {
		t.Fatalf("expected sub=principal-1, got %v", claims["sub"])
	}
	if claims["aud"] != "https://vault.local/" {
		t.Fatalf("expected aud echo, got %v", claims["aud"])
	}

	// Introspection resolves the token.
	intro, status := introspect(t, ts, m["access_token"].(string))
	if status != 200 || intro["active"] != true {
		t.Fatalf("expected active introspection, got %d %v", status, intro)
	}
	if intro["identityId"] != armID || intro["vmId"] != tokVMID {
		t.Fatalf("introspection mismatch: %v", intro)
	}
}

func TestTokenNoIdentityAssigned(t *testing.T) {
	ts, _ := tokenTestServer(t, nil)
	m, status := requestToken(t, ts, tokSecret, "")
	if status != 400 || m["error"] != "invalid_request" {
		t.Fatalf("expected 400 invalid_request, got %d %v", status, m)
	}
}

func TestTokenUnknownSecret(t *testing.T) {
	ts, _ := tokenTestServer(t, nil)
	_, status := requestToken(t, ts, "wrong-secret", "")
	if status != 401 {
		t.Fatalf("expected 401, got %d", status)
	}
}

func TestTokenMultipleIdentitiesSelector(t *testing.T) {
	armA := "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id-a"
	armB := "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id-b"
	ts, st := tokenTestServer(t, map[string]interface{}{
		armA: map[string]interface{}{"principalId": "pa", "clientId": "ca"},
		armB: map[string]interface{}{"principalId": "pb", "clientId": "cb"},
	})
	seedIdentity(st, "id-a", "pa", "ca")
	seedIdentity(st, "id-b", "pb", "cb")

	// No selector with two identities → 400.
	m, status := requestToken(t, ts, tokSecret, "")
	if status != 400 || m["error"] != "invalid_request" {
		t.Fatalf("expected 400 invalid_request, got %d %v", status, m)
	}

	// client_id selector.
	m, status = requestToken(t, ts, tokSecret, "client_id=cb")
	if status != 200 || m["client_id"] != "cb" {
		t.Fatalf("expected token for cb, got %d %v", status, m)
	}
	claims := decodeJWTClaims(t, m["access_token"].(string))
	if claims["xms_mirid"] != armB {
		t.Fatalf("expected xms_mirid=%s, got %v", armB, claims["xms_mirid"])
	}

	// mi_res_id selector.
	m, status = requestToken(t, ts, tokSecret, "mi_res_id="+armA)
	if status != 200 || m["client_id"] != "ca" {
		t.Fatalf("expected token for ca, got %d %v", status, m)
	}

	// Selector that matches nothing → 400.
	_, status = requestToken(t, ts, tokSecret, "client_id=does-not-exist")
	if status != 400 {
		t.Fatalf("expected 400 for unknown client_id, got %d", status)
	}
}

func TestTokenDeletedIdentity(t *testing.T) {
	armID := "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentities/gone"
	ts, st := tokenTestServer(t, map[string]interface{}{
		armID: map[string]interface{}{"principalId": "p", "clientId": "c"},
	})
	// The identity record never existed (dangling reference) → 400.
	_ = st
	m, status := requestToken(t, ts, tokSecret, "")
	if status != 400 || m["error"] != "invalid_request" {
		t.Fatalf("expected 400 invalid_request for dangling identity, got %d %v", status, m)
	}
}

func TestIntrospectUnknownToken(t *testing.T) {
	ts, _ := tokenTestServer(t, nil)
	m, status := introspect(t, ts, "garbage.token.here")
	if status != 200 || m["active"] != false {
		t.Fatalf("expected active=false, got %d %v", status, m)
	}
}

// ---------- small local helpers ----------

func introspect(t *testing.T, ts *httptest.Server, token string) (map[string]interface{}, int) {
	t.Helper()
	resp := doRequest(t, "POST", ts.URL+"/metadata/identity/introspect", `{"token":"`+token+`"}`)
	defer resp.Body.Close()
	var m map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&m)
	return m, resp.StatusCode
}
