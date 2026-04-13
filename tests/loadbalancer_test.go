package tests

import (
	"testing"
)

func TestLoadBalancerLifecycle(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/loadBalancers"
	av := "?api-version=2023-09-01"

	// Create Load Balancer with full config
	resp := doRequest(t, "PUT", base+"/lb1"+av, `{
		"location":"eastus",
		"sku":{"name":"Standard","tier":"Regional"},
		"properties":{
			"frontendIPConfigurations":[{"name":"frontend1","properties":{"publicIPAddress":{"id":"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/publicIPAddresses/pip1"}}}],
			"backendAddressPools":[{"name":"backend1"}],
			"probes":[{"name":"probe1","properties":{"protocol":"Tcp","port":80,"intervalInSeconds":15,"numberOfProbes":2}}],
			"loadBalancingRules":[{"name":"rule1","properties":{"frontendIPConfiguration":{"id":"frontend1"},"backendAddressPool":{"id":"backend1"},"probe":{"id":"probe1"},"protocol":"Tcp","frontendPort":80,"backendPort":80}}]
		}
	}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()
	if m["name"] != "lb1" {
		t.Fatalf("expected name=lb1, got %v", m["name"])
	}
	if m["type"] != "Microsoft.Network/loadBalancers" {
		t.Fatalf("expected correct type, got %v", m["type"])
	}
	if m["location"] != "eastus" {
		t.Fatalf("expected location=eastus, got %v", m["location"])
	}
	sku := m["sku"].(map[string]interface{})
	if sku["name"] != "Standard" {
		t.Fatalf("expected sku=Standard, got %v", sku["name"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded")
	}
	frontends := props["frontendIPConfigurations"].([]interface{})
	if len(frontends) != 1 {
		t.Fatalf("expected 1 frontend, got %d", len(frontends))
	}
	backends := props["backendAddressPools"].([]interface{})
	if len(backends) != 1 {
		t.Fatalf("expected 1 backend pool, got %d", len(backends))
	}
	probes := props["probes"].([]interface{})
	if len(probes) != 1 {
		t.Fatalf("expected 1 probe, got %d", len(probes))
	}
	rules := props["loadBalancingRules"].([]interface{})
	if len(rules) != 1 {
		t.Fatalf("expected 1 LB rule, got %d", len(rules))
	}

	// Get Load Balancer
	resp = doRequest(t, "GET", base+"/lb1"+av, "")
	expectStatus(t, resp, 200)
	got := decodeJSON(t, resp)
	resp.Body.Close()
	if got["name"] != "lb1" {
		t.Fatalf("GET: expected name=lb1, got %v", got["name"])
	}

	// List Load Balancers
	resp = doRequest(t, "GET", base+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	resp.Body.Close()
	items := list["value"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 load balancer, got %d", len(items))
	}

	// Update Load Balancer - add second frontend, verify 200
	resp = doRequest(t, "PUT", base+"/lb1"+av, `{
		"location":"westus",
		"sku":{"name":"Standard","tier":"Regional"},
		"properties":{
			"frontendIPConfigurations":[{"name":"frontend1"},{"name":"frontend2"}],
			"backendAddressPools":[{"name":"backend1"}]
		}
	}`)
	expectStatus(t, resp, 200)
	updated := decodeJSON(t, resp)
	resp.Body.Close()
	updatedProps := updated["properties"].(map[string]interface{})
	updatedFrontends := updatedProps["frontendIPConfigurations"].([]interface{})
	if len(updatedFrontends) != 2 {
		t.Fatalf("expected 2 frontends after update, got %d", len(updatedFrontends))
	}
	if updated["location"] != "westus" {
		t.Fatalf("expected location=westus after update, got %v", updated["location"])
	}

	// Delete Load Balancer
	resp = doRequest(t, "DELETE", base+"/lb1"+av, "")
	expectStatus(t, resp, 202)
	resp.Body.Close()

	// Get deleted
	resp = doRequest(t, "GET", base+"/lb1"+av, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
}

func TestLoadBalancerNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/loadBalancers"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "GET", base+"/nonexistent"+av, "")
	expectStatus(t, resp, 404)
	e := decodeError(t, resp)
	resp.Body.Close()
	if e.Error.Code != "ResourceNotFound" {
		t.Fatalf("expected ResourceNotFound, got %s", e.Error.Code)
	}
}

func TestLoadBalancerDeleteNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/loadBalancers"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "DELETE", base+"/nonexistent"+av, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
}

func TestLoadBalancerEmptyList(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/loadBalancers"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "GET", base+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	resp.Body.Close()
	items := list["value"].([]interface{})
	if len(items) != 0 {
		t.Fatalf("expected 0 LBs, got %d", len(items))
	}
}

func TestLoadBalancerDefaults(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/loadBalancers"
	av := "?api-version=2023-09-01"

	// Create with minimal body
	resp := doRequest(t, "PUT", base+"/lb-minimal"+av, `{}`)
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

	// All property arrays should be empty, not nil
	props := m["properties"].(map[string]interface{})
	for _, field := range []string{"frontendIPConfigurations", "backendAddressPools", "loadBalancingRules", "probes", "inboundNatRules", "outboundRules"} {
		arr, ok := props[field].([]interface{})
		if !ok {
			t.Fatalf("expected %s to be an array, got %T", field, props[field])
		}
		if len(arr) != 0 {
			t.Fatalf("expected %s to be empty, got %d items", field, len(arr))
		}
	}

	doRequest(t, "DELETE", base+"/lb-minimal"+av, "").Body.Close()
}

func TestLoadBalancerMultiple(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/loadBalancers"
	av := "?api-version=2023-09-01"

	doRequest(t, "PUT", base+"/lb-a"+av, `{"location":"eastus"}`).Body.Close()
	doRequest(t, "PUT", base+"/lb-b"+av, `{"location":"westus"}`).Body.Close()

	resp := doRequest(t, "GET", base+av, "")
	list := decodeJSON(t, resp)
	resp.Body.Close()
	items := list["value"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 LBs, got %d", len(items))
	}

	doRequest(t, "DELETE", base+"/lb-a"+av, "").Body.Close()

	resp = doRequest(t, "GET", base+av, "")
	list = decodeJSON(t, resp)
	resp.Body.Close()
	items = list["value"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 LB after delete, got %d", len(items))
	}
}
