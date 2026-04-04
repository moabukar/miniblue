package tests

import (
	"testing"
)

func TestEventGridCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.EventGrid/topics"
	av := "?api-version=2023-12-15-preview"

	// Create
	resp := doRequest(t, "PUT", base+"/myevents"+av, `{"location":"eastus"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded, got %v", props["provisioningState"])
	}

	// Update (same URL) should return 200
	resp = doRequest(t, "PUT", base+"/myevents"+av, `{"location":"eastus"}`)
	resp.Body.Close()
	expectStatus(t, resp, 200)

	// Publish
	resp = doRequest(t, "POST", ts.URL+"/eventgrid/myevents/events",
		`[{"id":"1","eventType":"test","subject":"test","data":{}}]`)
	resp.Body.Close()
	expectStatus(t, resp, 200)

	// Delete
	resp = doRequest(t, "DELETE", base+"/myevents"+av, "")
	resp.Body.Close()
	expectStatus(t, resp, 202)

	// Get deleted - 404
	resp = doRequest(t, "GET", base+"/myevents"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}
