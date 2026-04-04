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
	if m["version"] != "0.1.0" {
		t.Fatalf("expected version=0.1.0, got %v", m["version"])
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

func TestAPIVersionRequired(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	// ARM endpoint without api-version should fail
	resp := doRequest(t, "GET", ts.URL+"/subscriptions/sub1/resourcegroups", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 400)

	e := decodeError(t, resp)
	if e.Error.Code != "MissingApiVersionParameter" {
		t.Fatalf("expected MissingApiVersionParameter, got %s", e.Error.Code)
	}
}

func TestAPIVersionAccepted(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	// With api-version should work
	resp := doRequest(t, "GET", ts.URL+"/subscriptions/sub1/resourcegroups?api-version=2023-07-01", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
}
