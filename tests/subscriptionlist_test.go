package tests

import (
	"testing"
)

func TestListVNetsInSubscription(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	av := "?api-version=2023-09-01"
	base := ts.URL + "/subscriptions/sub1"

	doRequest(t, "PUT", base+"/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks/vnet-a"+av,
		`{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`).Body.Close()
	doRequest(t, "PUT", base+"/resourceGroups/rg2/providers/Microsoft.Network/virtualNetworks/vnet-b"+av,
		`{"location":"westus","properties":{"addressSpace":{"addressPrefixes":["10.1.0.0/16"]}}}`).Body.Close()

	resp := doRequest(t, "GET", base+"/providers/Microsoft.Network/virtualNetworks"+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	resp.Body.Close()

	items := list["value"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 VNets across RGs, got %d", len(items))
	}
}

func TestListNSGsInSubscription(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	av := "?api-version=2023-09-01"
	base := ts.URL + "/subscriptions/sub1"

	doRequest(t, "PUT", base+"/resourceGroups/rg1/providers/Microsoft.Network/networkSecurityGroups/nsg-a"+av,
		`{"location":"eastus"}`).Body.Close()
	doRequest(t, "PUT", base+"/resourceGroups/rg2/providers/Microsoft.Network/networkSecurityGroups/nsg-b"+av,
		`{"location":"westus"}`).Body.Close()

	resp := doRequest(t, "GET", base+"/providers/Microsoft.Network/networkSecurityGroups"+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	resp.Body.Close()

	items := list["value"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 NSGs across RGs, got %d", len(items))
	}
}

func TestListPublicIPsInSubscription(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	av := "?api-version=2023-09-01"
	base := ts.URL + "/subscriptions/sub1"

	doRequest(t, "PUT", base+"/resourceGroups/rg1/providers/Microsoft.Network/publicIPAddresses/pip-a"+av,
		`{"location":"eastus"}`).Body.Close()
	doRequest(t, "PUT", base+"/resourceGroups/rg2/providers/Microsoft.Network/publicIPAddresses/pip-b"+av,
		`{"location":"westus"}`).Body.Close()

	resp := doRequest(t, "GET", base+"/providers/Microsoft.Network/publicIPAddresses"+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	resp.Body.Close()

	items := list["value"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 PIPs across RGs, got %d", len(items))
	}
}

func TestListLoadBalancersInSubscription(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	av := "?api-version=2023-09-01"
	base := ts.URL + "/subscriptions/sub1"

	doRequest(t, "PUT", base+"/resourceGroups/rg1/providers/Microsoft.Network/loadBalancers/lb-a"+av,
		`{"location":"eastus"}`).Body.Close()
	doRequest(t, "PUT", base+"/resourceGroups/rg2/providers/Microsoft.Network/loadBalancers/lb-b"+av,
		`{"location":"westus"}`).Body.Close()

	resp := doRequest(t, "GET", base+"/providers/Microsoft.Network/loadBalancers"+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	resp.Body.Close()

	items := list["value"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 LBs across RGs, got %d", len(items))
	}
}
