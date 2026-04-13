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
	resp.Body.Close()
	if m["name"] != "pip1" {
		t.Fatalf("expected name=pip1, got %v", m["name"])
	}
	if m["type"] != "Microsoft.Network/publicIPAddresses" {
		t.Fatalf("expected correct type, got %v", m["type"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded")
	}
	if props["publicIPAllocationMethod"] != "Static" {
		t.Fatalf("expected allocationMethod=Static, got %v", props["publicIPAllocationMethod"])
	}
	if props["publicIPAddressVersion"] != "IPv4" {
		t.Fatalf("expected version=IPv4, got %v", props["publicIPAddressVersion"])
	}
	originalIP := props["ipAddress"].(string)
	if originalIP == "" {
		t.Fatalf("expected ipAddress to be set")
	}

	// Get Public IP
	resp = doRequest(t, "GET", base+"/pip1"+av, "")
	expectStatus(t, resp, 200)
	got := decodeJSON(t, resp)
	resp.Body.Close()
	if got["name"] != "pip1" {
		t.Fatalf("GET: expected name=pip1, got %v", got["name"])
	}

	// List Public IPs
	resp = doRequest(t, "GET", base+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	resp.Body.Close()
	items := list["value"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 public IP, got %d", len(items))
	}

	// Update Public IP (PUT again) - IP should be preserved
	resp = doRequest(t, "PUT", base+"/pip1"+av,
		`{"location":"westus","sku":{"name":"Standard","tier":"Regional"},"properties":{"publicIPAllocationMethod":"Dynamic"}}`)
	expectStatus(t, resp, 200)
	updated := decodeJSON(t, resp)
	resp.Body.Close()
	updatedProps := updated["properties"].(map[string]interface{})
	if updatedProps["ipAddress"] != originalIP {
		t.Fatalf("expected IP preserved on update: want %s, got %v", originalIP, updatedProps["ipAddress"])
	}
	if updatedProps["publicIPAllocationMethod"] != "Dynamic" {
		t.Fatalf("expected allocationMethod=Dynamic after update, got %v", updatedProps["publicIPAllocationMethod"])
	}
	if updated["location"] != "westus" {
		t.Fatalf("expected location=westus after update, got %v", updated["location"])
	}

	// Delete Public IP
	resp = doRequest(t, "DELETE", base+"/pip1"+av, "")
	expectStatus(t, resp, 202)
	resp.Body.Close()

	// Get deleted
	resp = doRequest(t, "GET", base+"/pip1"+av, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
}

func TestPublicIPNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/publicIPAddresses"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "GET", base+"/nonexistent"+av, "")
	expectStatus(t, resp, 404)
	e := decodeError(t, resp)
	resp.Body.Close()
	if e.Error.Code != "ResourceNotFound" {
		t.Fatalf("expected ResourceNotFound, got %s", e.Error.Code)
	}
}

func TestPublicIPDeleteNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/publicIPAddresses"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "DELETE", base+"/nonexistent"+av, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
}

func TestPublicIPEmptyList(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/publicIPAddresses"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "GET", base+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	resp.Body.Close()
	items := list["value"].([]interface{})
	if len(items) != 0 {
		t.Fatalf("expected 0 public IPs, got %d", len(items))
	}
}

func TestPublicIPDefaults(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/publicIPAddresses"
	av := "?api-version=2023-09-01"

	// Create with minimal body - should get defaults
	resp := doRequest(t, "PUT", base+"/pip-defaults"+av, `{}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()

	if m["location"] != "eastus" {
		t.Fatalf("expected default location=eastus, got %v", m["location"])
	}
	sku := m["sku"].(map[string]interface{})
	if sku["name"] != "Standard" {
		t.Fatalf("expected default sku=Standard, got %v", sku["name"])
	}
	props := m["properties"].(map[string]interface{})
	if props["publicIPAllocationMethod"] != "Static" {
		t.Fatalf("expected default allocationMethod=Static, got %v", props["publicIPAllocationMethod"])
	}
	if props["publicIPAddressVersion"] != "IPv4" {
		t.Fatalf("expected default version=IPv4, got %v", props["publicIPAddressVersion"])
	}

	doRequest(t, "DELETE", base+"/pip-defaults"+av, "").Body.Close()
}

func TestPublicIPMultipleResources(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/publicIPAddresses"
	av := "?api-version=2023-09-01"

	doRequest(t, "PUT", base+"/pip-a"+av, `{"location":"eastus"}`).Body.Close()
	doRequest(t, "PUT", base+"/pip-b"+av, `{"location":"westus"}`).Body.Close()
	doRequest(t, "PUT", base+"/pip-c"+av, `{"location":"northeurope"}`).Body.Close()

	resp := doRequest(t, "GET", base+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	resp.Body.Close()
	items := list["value"].([]interface{})
	if len(items) != 3 {
		t.Fatalf("expected 3 public IPs, got %d", len(items))
	}

	// Delete one and verify count
	doRequest(t, "DELETE", base+"/pip-b"+av, "").Body.Close()
	resp = doRequest(t, "GET", base+av, "")
	list = decodeJSON(t, resp)
	resp.Body.Close()
	items = list["value"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 public IPs after delete, got %d", len(items))
	}
}
