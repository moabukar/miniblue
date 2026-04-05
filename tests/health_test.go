package tests

import (
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/health", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["status"] != "running" {
		t.Fatalf("expected status=running, got %v", m["status"])
	}
	if m["version"] == nil || m["version"] == "" {
		t.Fatal("expected version to be set")
	}
	if m["service_count"] == nil {
		t.Fatal("expected service_count to be set")
	}
}

func TestAzureResponseHeaders(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/health", "")
	defer resp.Body.Close()

	headers := []string{"x-ms-request-id", "x-ms-correlation-request-id", "x-ms-version"}
	for _, h := range headers {
		if resp.Header.Get(h) == "" {
			t.Fatalf("missing header: %s", h)
		}
	}
	if resp.Header.Get("x-ms-version") != "2023-11-03" {
		t.Fatalf("wrong x-ms-version: %s", resp.Header.Get("x-ms-version"))
	}
}

func TestAPIVersionOptional(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	// Without api-version should still work (miniblue is lenient)
	resp := doRequest(t, "GET", ts.URL+"/subscriptions/sub1/resourcegroups", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
}

func TestAPIVersionAccepted(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	// With api-version should also work
	resp := doRequest(t, "GET", ts.URL+"/subscriptions/sub1/resourcegroups?api-version=2023-07-01", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
}

func TestResetEndpoint(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	// Create a resource
	doRequest(t, "PUT", ts.URL+"/subscriptions/sub1/resourcegroups/rg1?api-version=2023-01-01",
		`{"location":"eastus"}`).Body.Close()

	// Reset
	resp := doRequest(t, "POST", ts.URL+"/_miniblue/reset", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	// Resource should be gone
	resp2 := doRequest(t, "GET", ts.URL+"/subscriptions/sub1/resourcegroups/rg1?api-version=2023-01-01", "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 404)
}
