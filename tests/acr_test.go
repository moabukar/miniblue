package tests

import (
	"testing"
)

func TestACRRegistry(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ContainerRegistry/registries"
	av := "?api-version=2023-07-01"

	resp := doRequest(t, "PUT", base+"/myreg"+av,
		`{"location":"eastus","sku":{"name":"Premium","tier":"Premium"}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	props := m["properties"].(map[string]interface{})
	if props["loginServer"] != "myreg.azurecr.io" {
		t.Fatalf("expected loginServer=myreg.azurecr.io, got %v", props["loginServer"])
	}
	sku := m["sku"].(map[string]interface{})
	if sku["name"] != "Premium" {
		t.Fatalf("expected sku=Premium, got %v", sku["name"])
	}
}
