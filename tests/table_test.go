package tests

import (
	"testing"
)

func TestTableStorageCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/table/myaccount/users"

	// Create table
	resp := doRequest(t, "POST", base, "")
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Insert entity
	resp = doRequest(t, "PUT", base+"/partition1/row1",
		`{"properties":{"name":"Mo","age":30}}`)
	resp.Body.Close()
	expectStatus(t, resp, 204)

	// Get entity
	resp = doRequest(t, "GET", base+"/partition1/row1", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["PartitionKey"] != "partition1" {
		t.Fatalf("expected PartitionKey=partition1, got %v", m["PartitionKey"])
	}

	// Delete + 404
	doRequest(t, "DELETE", base+"/partition1/row1", "").Body.Close()
	resp = doRequest(t, "GET", base+"/partition1/row1", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}
