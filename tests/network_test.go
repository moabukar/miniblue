package tests

import (
	"testing"
)

func TestVNetSubnetLifecycle(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks"
	av := "?api-version=2023-09-01"

	// Create VNet
	resp := doRequest(t, "PUT", base+"/vnet1"+av,
		`{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`)
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Create Subnet
	resp = doRequest(t, "PUT", base+"/vnet1/subnets/web"+av,
		`{"properties":{"addressPrefix":"10.0.1.0/24"}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != "web" {
		t.Fatalf("expected subnet name=web, got %v", m["name"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded")
	}

	// List subnets
	resp = doRequest(t, "GET", base+"/vnet1/subnets"+av, "")
	list := decodeJSON(t, resp)
	subnets := list["value"].([]interface{})
	if len(subnets) != 1 {
		t.Fatalf("expected 1 subnet, got %d", len(subnets))
	}

	// Delete VNet cascades subnets
	doRequest(t, "DELETE", base+"/vnet1"+av, "").Body.Close()
	resp = doRequest(t, "GET", base+"/vnet1/subnets/web"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestSubnetOnNonexistentVNet(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "PUT", base+"/nope/subnets/web"+av,
		`{"properties":{"addressPrefix":"10.0.1.0/24"}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}
