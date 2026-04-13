package tests

import (
	"testing"
)

func TestPublicIPLifecycle(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/publicIPAddresses"
	av := "?api-version=2023-09-01"

	// Create Public IP
	resp := doRequest(t, "PUT", base+"/pip1"+av,
		`{"location":"eastus","sku":{"name":"Standard","tier":"Regional"},"properties":{"publicIPAllocationMethod":"Static","publicIPAddressVersion":"IPv4"}}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	if m["name"] != "pip1" {
		t.Fatalf("expected name=pip1, got %v", m["name"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded")
	}
	if props["ipAddress"] == nil || props["ipAddress"] == "" {
		t.Fatalf("expected ipAddress to be set")
	}
	resp.Body.Close()

	// Get Public IP
	resp = doRequest(t, "GET", base+"/pip1"+av, "")
	expectStatus(t, resp, 200)
	resp.Body.Close()

	// List Public IPs
	resp = doRequest(t, "GET", base+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	items := list["value"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 public IP, got %d", len(items))
	}
	resp.Body.Close()

	// Update Public IP (PUT again)
	resp = doRequest(t, "PUT", base+"/pip1"+av,
		`{"location":"westus","sku":{"name":"Standard","tier":"Regional"},"properties":{"publicIPAllocationMethod":"Dynamic"}}`)
	expectStatus(t, resp, 200)
	resp.Body.Close()

	// Delete Public IP
	resp = doRequest(t, "DELETE", base+"/pip1"+av, "")
	expectStatus(t, resp, 202)
	resp.Body.Close()

	// Get deleted
	resp = doRequest(t, "GET", base+"/pip1"+av, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
}
