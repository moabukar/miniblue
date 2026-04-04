package tests

import (
	"testing"
)

func TestCosmosDBFullDocument(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/cosmosdb/myaccount/dbs/mydb/colls/users/docs"

	resp := doRequest(t, "POST", base, `{"id":"u1","name":"Mo","role":"CTO","email":"mo@test.com"}`)
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Get - verify all fields
	resp = doRequest(t, "GET", base+"/u1", "")
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	if m["name"] != "Mo" {
		t.Fatalf("expected name=Mo, got %v", m["name"])
	}
	if m["role"] != "CTO" {
		t.Fatalf("expected role=CTO, got %v", m["role"])
	}
}

func TestCosmosDBReplace(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/cosmosdb/myaccount/dbs/mydb/colls/users/docs"

	doRequest(t, "POST", base, `{"id":"u1","name":"Mo"}`).Body.Close()

	// Replace
	resp := doRequest(t, "PUT", base+"/u1", `{"name":"Mo Updated","role":"Founder"}`)
	resp.Body.Close()
	expectStatus(t, resp, 200)

	// Verify update
	resp = doRequest(t, "GET", base+"/u1", "")
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	if m["name"] != "Mo Updated" {
		t.Fatalf("expected name='Mo Updated', got %v", m["name"])
	}
}

func TestCosmosDBDelete(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/cosmosdb/myaccount/dbs/mydb/colls/users/docs"

	doRequest(t, "POST", base, `{"id":"u1","name":"Mo"}`).Body.Close()

	resp := doRequest(t, "DELETE", base+"/u1", "")
	resp.Body.Close()
	expectStatus(t, resp, 204)

	resp = doRequest(t, "GET", base+"/u1", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}
