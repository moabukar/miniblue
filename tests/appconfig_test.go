package tests

import (
	"testing"
)

func TestAppConfigCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/appconfig/mystore/kv"

	// Set
	resp := doRequest(t, "PUT", base+"/db-host", `{"value":"localhost:5432"}`)
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	if m["value"] != "localhost:5432" {
		t.Fatalf("expected value=localhost:5432, got %v", m["value"])
	}
	if m["etag"] == nil {
		t.Fatal("expected etag")
	}

	// Delete then get = 404
	doRequest(t, "DELETE", base+"/db-host", "").Body.Close()
	resp = doRequest(t, "GET", base+"/db-host", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}
