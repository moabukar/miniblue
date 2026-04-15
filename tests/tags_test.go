package tests

import (
	"testing"
)

func TestVNetTagsPreserved(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "PUT", base+"/tagged-vnet"+av,
		`{"location":"eastus","tags":{"env":"prod","team":"platform"},"properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()

	tags, ok := m["tags"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tags in response, got %T", m["tags"])
	}
	if tags["env"] != "prod" {
		t.Fatalf("expected tag env=prod, got %v", tags["env"])
	}
	if tags["team"] != "platform" {
		t.Fatalf("expected tag team=platform, got %v", tags["team"])
	}

	resp = doRequest(t, "GET", base+"/tagged-vnet"+av, "")
	expectStatus(t, resp, 200)
	got := decodeJSON(t, resp)
	resp.Body.Close()

	gotTags, ok := got["tags"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tags on GET, got %T", got["tags"])
	}
	if gotTags["env"] != "prod" {
		t.Fatalf("GET: expected tag env=prod, got %v", gotTags["env"])
	}

	doRequest(t, "DELETE", base+"/tagged-vnet"+av, "").Body.Close()
}

func TestNSGTagsPreserved(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/networkSecurityGroups"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "PUT", base+"/tagged-nsg"+av,
		`{"location":"eastus","tags":{"env":"staging"}}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()

	tags := m["tags"].(map[string]interface{})
	if tags["env"] != "staging" {
		t.Fatalf("expected tag env=staging, got %v", tags["env"])
	}

	doRequest(t, "DELETE", base+"/tagged-nsg"+av, "").Body.Close()
}

func TestPublicIPTagsPreserved(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/publicIPAddresses"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "PUT", base+"/tagged-pip"+av,
		`{"location":"eastus","tags":{"cost-center":"12345"}}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()

	tags := m["tags"].(map[string]interface{})
	if tags["cost-center"] != "12345" {
		t.Fatalf("expected tag cost-center=12345, got %v", tags["cost-center"])
	}

	doRequest(t, "DELETE", base+"/tagged-pip"+av, "").Body.Close()
}

func TestLoadBalancerTagsPreserved(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/loadBalancers"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "PUT", base+"/tagged-lb"+av,
		`{"location":"eastus","tags":{"managed-by":"terraform"}}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()

	tags := m["tags"].(map[string]interface{})
	if tags["managed-by"] != "terraform" {
		t.Fatalf("expected tag managed-by=terraform, got %v", tags["managed-by"])
	}

	doRequest(t, "DELETE", base+"/tagged-lb"+av, "").Body.Close()
}

func TestAppGatewayTagsPreserved(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/applicationGateways"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "PUT", base+"/tagged-appgw"+av,
		`{"location":"eastus","tags":{"owner":"devops"},"properties":{}}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()

	tags := m["tags"].(map[string]interface{})
	if tags["owner"] != "devops" {
		t.Fatalf("expected tag owner=devops, got %v", tags["owner"])
	}

	doRequest(t, "DELETE", base+"/tagged-appgw"+av, "").Body.Close()
}

func TestDNSZoneTagsPreserved(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/dnsZones"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "PUT", base+"/tagged.example.com"+av,
		`{"location":"global","tags":{"zone-type":"public"}}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()

	tags, ok := m["tags"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tags on DNS zone, got %T", m["tags"])
	}
	if tags["zone-type"] != "public" {
		t.Fatalf("expected tag zone-type=public, got %v", tags["zone-type"])
	}

	doRequest(t, "DELETE", base+"/tagged.example.com"+av, "").Body.Close()
}

func TestEmptyTagsReturnsEmptyMap(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "PUT", base+"/no-tags-vnet"+av,
		`{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()

	tags, ok := m["tags"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tags to be empty map, got %T", m["tags"])
	}
	if len(tags) != 0 {
		t.Fatalf("expected 0 tags, got %d", len(tags))
	}

	doRequest(t, "DELETE", base+"/no-tags-vnet"+av, "").Body.Close()
}
