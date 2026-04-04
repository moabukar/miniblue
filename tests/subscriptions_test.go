package tests

import (
	"testing"
)

func TestSubscriptionsAndTenants(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/subscriptions?api-version=2022-12-01", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	subs := m["value"].([]interface{})
	if len(subs) == 0 {
		t.Fatal("expected at least 1 subscription")
	}

	resp = doRequest(t, "GET", ts.URL+"/tenants?api-version=2022-12-01", "")
	m = decodeJSON(t, resp)
	tenants := m["value"].([]interface{})
	if len(tenants) == 0 {
		t.Fatal("expected at least 1 tenant")
	}
}

func TestManagedIdentityToken(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/metadata/identity/oauth2/token?resource=https://management.azure.com/", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["token_type"] != "Bearer" {
		t.Fatalf("expected token_type=Bearer, got %v", m["token_type"])
	}
	if m["access_token"] == nil || m["access_token"] == "" {
		t.Fatal("expected non-empty access_token")
	}
}

func TestMetadataEndpoints(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/metadata/endpoints", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
}
