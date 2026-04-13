package tests

import (
	"testing"
)

func TestLoadBalancerLifecycle(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/loadBalancers"
	av := "?api-version=2023-09-01"

	// Create Load Balancer
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
	if m["name"] != "lb1" {
		t.Fatalf("expected name=lb1, got %v", m["name"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded")
	}
	frontends := props["frontendIPConfigurations"].([]interface{})
	if len(frontends) != 1 {
		t.Fatalf("expected 1 frontend IP config, got %d", len(frontends))
	}
	backends := props["backendAddressPools"].([]interface{})
	if len(backends) != 1 {
		t.Fatalf("expected 1 backend pool, got %d", len(backends))
	}
	resp.Body.Close()

	// Get Load Balancer
	resp = doRequest(t, "GET", base+"/lb1"+av, "")
	expectStatus(t, resp, 200)
	resp.Body.Close()

	// List Load Balancers
	resp = doRequest(t, "GET", base+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	items := list["value"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 load balancer, got %d", len(items))
	}
	resp.Body.Close()

	// Update Load Balancer
	resp = doRequest(t, "PUT", base+"/lb1"+av, `{
		"location":"eastus",
		"sku":{"name":"Standard","tier":"Regional"},
		"properties":{
			"frontendIPConfigurations":[{"name":"frontend1"},{"name":"frontend2"}],
			"backendAddressPools":[{"name":"backend1"}]
		}
	}`)
	expectStatus(t, resp, 200)
	resp.Body.Close()

	// Delete Load Balancer
	resp = doRequest(t, "DELETE", base+"/lb1"+av, "")
	expectStatus(t, resp, 202)
	resp.Body.Close()

	// Get deleted
	resp = doRequest(t, "GET", base+"/lb1"+av, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
}
