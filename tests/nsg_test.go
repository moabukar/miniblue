package tests

import (
	"testing"
)

func TestNSGLifecycle(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/networkSecurityGroups"
	av := "?api-version=2023-09-01"

	// Create NSG
	resp := doRequest(t, "PUT", base+"/nsg1"+av,
		`{"location":"eastus"}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	if m["name"] != "nsg1" {
		t.Fatalf("expected name=nsg1, got %v", m["name"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded")
	}
	defaultRules := props["defaultSecurityRules"].([]interface{})
	if len(defaultRules) < 3 {
		t.Fatalf("expected at least 3 default security rules, got %d", len(defaultRules))
	}
	resp.Body.Close()

	// Get NSG
	resp = doRequest(t, "GET", base+"/nsg1"+av, "")
	expectStatus(t, resp, 200)
	resp.Body.Close()

	// Create security rule
	resp = doRequest(t, "PUT", base+"/nsg1/securityRules/allow-http"+av,
		`{"properties":{"priority":100,"direction":"Inbound","access":"Allow","protocol":"Tcp","sourcePortRange":"*","destinationPortRange":"80","sourceAddressPrefix":"*","destinationAddressPrefix":"*"}}`)
	expectStatus(t, resp, 201)
	rule := decodeJSON(t, resp)
	if rule["name"] != "allow-http" {
		t.Fatalf("expected rule name=allow-http, got %v", rule["name"])
	}
	ruleProps := rule["properties"].(map[string]interface{})
	if ruleProps["priority"] != float64(100) {
		t.Fatalf("expected priority=100, got %v", ruleProps["priority"])
	}
	resp.Body.Close()

	// List security rules
	resp = doRequest(t, "GET", base+"/nsg1/securityRules"+av, "")
	expectStatus(t, resp, 200)
	list := decodeJSON(t, resp)
	rules := list["value"].([]interface{})
	if len(rules) != 1 {
		t.Fatalf("expected 1 security rule, got %d", len(rules))
	}
	resp.Body.Close()

	// Delete security rule
	resp = doRequest(t, "DELETE", base+"/nsg1/securityRules/allow-http"+av, "")
	expectStatus(t, resp, 202)
	resp.Body.Close()

	// Verify rule deleted
	resp = doRequest(t, "GET", base+"/nsg1/securityRules/allow-http"+av, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()

	// Delete NSG
	resp = doRequest(t, "DELETE", base+"/nsg1"+av, "")
	expectStatus(t, resp, 202)
	resp.Body.Close()

	// Get deleted NSG
	resp = doRequest(t, "GET", base+"/nsg1"+av, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
}

func TestNSGCascadeDeleteRules(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/networkSecurityGroups"
	av := "?api-version=2023-09-01"

	// Create NSG with a rule
	doRequest(t, "PUT", base+"/nsg2"+av, `{"location":"eastus"}`).Body.Close()
	doRequest(t, "PUT", base+"/nsg2/securityRules/rule1"+av,
		`{"properties":{"priority":200,"direction":"Inbound","access":"Deny","protocol":"*","sourcePortRange":"*","destinationPortRange":"*","sourceAddressPrefix":"*","destinationAddressPrefix":"*"}}`).Body.Close()

	// Delete NSG should cascade delete rules
	doRequest(t, "DELETE", base+"/nsg2"+av, "").Body.Close()

	resp := doRequest(t, "GET", base+"/nsg2/securityRules/rule1"+av, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
}

func TestRuleOnNonexistentNSG(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/networkSecurityGroups"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "PUT", base+"/nope/securityRules/rule1"+av,
		`{"properties":{"priority":100,"direction":"Inbound","access":"Allow","protocol":"Tcp","sourcePortRange":"*","destinationPortRange":"443","sourceAddressPrefix":"*","destinationAddressPrefix":"*"}}`)
	expectStatus(t, resp, 404)
	resp.Body.Close()
}
