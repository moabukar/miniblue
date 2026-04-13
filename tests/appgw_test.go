package tests

import (
	"testing"
)

func TestAppGatewayLifecycle(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/applicationGateways"
	av := "?api-version=2023-09-01"

	body := `{
		"location":"eastus",
		"properties":{
			"sku":{"name":"Standard_v2","tier":"Standard_v2","capacity":2},
			"gatewayIPConfigurations":[{"name":"gw-ip","properties":{"subnet":{"id":"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks/vnet1/subnets/appgw-subnet"}}}],
			"frontendIPConfigurations":[{"name":"fe-ip","properties":{"publicIPAddress":{"id":"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/publicIPAddresses/appgw-pip"}}}],
			"frontendPorts":[{"name":"port80","properties":{"port":80}},{"name":"port443","properties":{"port":443}}],
			"backendAddressPools":[{"name":"backend","properties":{"backendAddresses":[{"ipAddress":"10.0.1.10"},{"ipAddress":"10.0.1.11"}]}}],
			"backendHttpSettingsCollection":[{"name":"http-settings","properties":{"port":80,"protocol":"Http","cookieBasedAffinity":"Disabled","requestTimeout":30}}],
			"httpListeners":[{"name":"http-listener","properties":{"frontendIPConfiguration":{"id":"fe-ip"},"frontendPort":{"id":"port80"},"protocol":"Http"}}],
			"requestRoutingRules":[{"name":"rule1","properties":{"ruleType":"Basic","priority":100,"httpListener":{"id":"http-listener"},"backendAddressPool":{"id":"backend"},"backendHttpSettings":{"id":"http-settings"}}}],
			"probes":[{"name":"health","properties":{"protocol":"Http","host":"localhost","path":"/health","interval":30,"timeout":30,"unhealthyThreshold":3}}]
		}
	}`

	// Create Application Gateway
	resp := doRequest(t, "PUT", base+"/appgw1"+av, body)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	if m["name"] != "appgw1" {
		t.Fatalf("expected name=appgw1, got %v", m["name"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded")
	}
	if props["operationalState"] != "Running" {
		t.Fatalf("expected operationalState=Running, got %v", props["operationalState"])
	}

	// Verify sub-configurations passed through
	sku := props["sku"].(map[string]interface{})
	if sku["name"] != "Standard_v2" {
		t.Fatalf("expected sku name=Standard_v2, got %v", sku["name"])
	}
	frontends := props["frontendIPConfigurations"].([]interface{})
	if len(frontends) != 1 {
		t.Fatalf("expected 1 frontend, got %d", len(frontends))
	}
	backends := props["backendAddressPools"].([]interface{})
	if len(backends) != 1 {
		t.Fatalf("expected 1 backend pool, got %d", len(backends))
	}
	listeners := props["httpListeners"].([]interface{})
	if len(listeners) != 1 {
		t.Fatalf("expected 1 listener, got %d", len(listeners))
	}
	rules := props["requestRoutingRules"].([]interface{})
	if len(rules) != 1 {
		t.Fatalf("expected 1 routing rule, got %d", len(rules))
	}
	ports := props["frontendPorts"].([]interface{})
	if len(ports) != 2 {
		t.Fatalf("expected 2 frontend ports, got %d", len(ports))
	}
	probes := props["probes"].([]interface{})
	if len(probes) != 1 {
		t.Fatalf("expected 1 probe, got %d", len(probes))
	}
	resp.Body.Close()

	// Get Application Gateway
	resp = doRequest(t, "GET", base+"/appgw1"+av, "")
	expectStatus(t, resp, 200)
	resp.Body.Close()

	// List Application Gateways
	resp = doRequest(t, "GET", base+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	items := list["value"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 app gateway, got %d", len(items))
	}
	resp.Body.Close()

	// Update Application Gateway
	resp = doRequest(t, "PUT", base+"/appgw1"+av, `{
		"location":"eastus",
		"properties":{
			"sku":{"name":"WAF_v2","tier":"WAF_v2","capacity":3},
			"frontendIPConfigurations":[{"name":"fe-ip"}],
			"backendAddressPools":[{"name":"backend1"},{"name":"backend2"}]
		}
	}`)
	expectStatus(t, resp, 200)
	updated := decodeJSON(t, resp)
	updatedProps := updated["properties"].(map[string]interface{})
	updatedSku := updatedProps["sku"].(map[string]interface{})
	if updatedSku["name"] != "WAF_v2" {
		t.Fatalf("expected updated sku name=WAF_v2, got %v", updatedSku["name"])
	}
	updatedBackends := updatedProps["backendAddressPools"].([]interface{})
	if len(updatedBackends) != 2 {
		t.Fatalf("expected 2 backend pools after update, got %d", len(updatedBackends))
	}
	resp.Body.Close()

	// Delete Application Gateway
	resp = doRequest(t, "DELETE", base+"/appgw1"+av, "")
	expectStatus(t, resp, 202)
	resp.Body.Close()

	// Get deleted
	resp = doRequest(t, "GET", base+"/appgw1"+av, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
}

func TestAppGatewayDefaultSku(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/applicationGateways"
	av := "?api-version=2023-09-01"

	// Create with minimal config (no sku specified)
	resp := doRequest(t, "PUT", base+"/appgw-minimal"+av,
		`{"location":"westus","properties":{}}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	props := m["properties"].(map[string]interface{})
	sku := props["sku"].(map[string]interface{})
	if sku["name"] != "Standard_v2" {
		t.Fatalf("expected default sku=Standard_v2, got %v", sku["name"])
	}
	if m["location"] != "westus" {
		t.Fatalf("expected location=westus, got %v", m["location"])
	}
	resp.Body.Close()

	doRequest(t, "DELETE", base+"/appgw-minimal"+av, "").Body.Close()
}
