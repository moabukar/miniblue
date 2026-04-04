package tests

import (
	"testing"
)

func TestResourceGroupCreate(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourcegroups"

	// Create
	resp := doRequest(t, "PUT", base+"/myRG?api-version=2023-07-01",
		`{"location":"eastus","tags":{"env":"dev"}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != "myRG" {
		t.Fatalf("expected name=myRG, got %v", m["name"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded, got %v", props["provisioningState"])
	}
}

func TestResourceGroupCreateDuplicate(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	url := ts.URL + "/subscriptions/sub1/resourcegroups/myRG?api-version=2023-07-01"

	// First create
	resp := doRequest(t, "PUT", url, `{"location":"eastus"}`)
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Second create - Azure returns 200 (update) not 409
	resp = doRequest(t, "PUT", url, `{"location":"eastus"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
}

func TestResourceGroupUpdate(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourcegroups/myRG?api-version=2023-07-01"

	// Create
	resp := doRequest(t, "PUT", base, `{"location":"eastus"}`)
	resp.Body.Close()

	// Patch tags
	resp = doRequest(t, "PATCH", base, `{"tags":{"env":"prod"}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	tags := m["tags"].(map[string]interface{})
	if tags["env"] != "prod" {
		t.Fatalf("expected tag env=prod, got %v", tags["env"])
	}
}

func TestResourceGroupNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/subscriptions/sub1/resourcegroups/nope?api-version=2023-07-01", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)

	e := decodeError(t, resp)
	if e.Error.Code != "ResourceNotFound" {
		t.Fatalf("expected ResourceNotFound, got %s", e.Error.Code)
	}
}

func TestResourceGroupCheckExistence(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourcegroups"

	// HEAD on nonexistent
	resp := doRequest(t, "HEAD", base+"/nope?api-version=2023-07-01", "")
	resp.Body.Close()
	expectStatus(t, resp, 404)

	// Create then HEAD
	resp = doRequest(t, "PUT", base+"/myRG?api-version=2023-07-01", `{"location":"eastus"}`)
	resp.Body.Close()
	resp = doRequest(t, "HEAD", base+"/myRG?api-version=2023-07-01", "")
	resp.Body.Close()
	expectStatus(t, resp, 204)
}

func TestResourceGroupDeleteCascade(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	av := "?api-version=2023-07-01"
	base := ts.URL + "/subscriptions/sub1"

	// Create RG
	resp := doRequest(t, "PUT", base+"/resourcegroups/myRG"+av, `{"location":"eastus"}`)
	resp.Body.Close()

	// Create VNet in RG
	resp = doRequest(t, "PUT", base+"/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/vnet1"+av,
		`{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`)
	resp.Body.Close()

	// Delete RG
	resp = doRequest(t, "DELETE", base+"/resourcegroups/myRG"+av, "")
	resp.Body.Close()
	expectStatus(t, resp, 202)

	// VNet should be gone
	resp = doRequest(t, "GET", base+"/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/vnet1"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestResourceGroupMissingLocation(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "PUT", ts.URL+"/subscriptions/sub1/resourcegroups/bad?api-version=2023-07-01", `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 400)

	e := decodeError(t, resp)
	if e.Error.Code != "InvalidRequestContent" {
		t.Fatalf("expected InvalidRequestContent, got %s", e.Error.Code)
	}
}
