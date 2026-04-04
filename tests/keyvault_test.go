package tests

import (
	"testing"
)

func TestKeyVaultCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/keyvault/myvault/secrets"

	// Set secret
	resp := doRequest(t, "PUT", base+"/db-pass", `{"value":"supersecret"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["value"] != "supersecret" {
		t.Fatalf("expected value=supersecret, got %v", m["value"])
	}

	// Get
	resp = doRequest(t, "GET", base+"/db-pass", "")
	m = decodeJSON(t, resp)
	if m["value"] != "supersecret" {
		t.Fatalf("get: expected value=supersecret, got %v", m["value"])
	}

	// List
	resp = doRequest(t, "GET", base, "")
	list := decodeJSON(t, resp)
	if list["value"] == nil {
		t.Fatal("expected value array in list response")
	}

	// Delete
	resp = doRequest(t, "DELETE", base+"/db-pass", "")
	resp.Body.Close()

	// Should 404 now
	resp = doRequest(t, "GET", base+"/db-pass", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}
