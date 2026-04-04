package tests

import (
	"testing"
)

func TestFunctionAppCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Web/sites"
	av := "?api-version=2023-01-01"

	// Create
	resp := doRequest(t, "PUT", base+"/myfunc"+av,
		`{"location":"eastus"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	props := m["properties"].(map[string]interface{})
	if props["defaultHostName"] != "myfunc.azurewebsites.net" {
		t.Fatalf("expected defaultHostName=myfunc.azurewebsites.net, got %v", props["defaultHostName"])
	}
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded")
	}

	// Update returns 200
	resp = doRequest(t, "PUT", base+"/myfunc"+av, `{"location":"eastus"}`)
	resp.Body.Close()
	expectStatus(t, resp, 200)

	// Delete then get = 404
	doRequest(t, "DELETE", base+"/myfunc"+av, "").Body.Close()
	resp = doRequest(t, "GET", base+"/myfunc"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}
