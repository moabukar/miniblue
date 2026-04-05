package tests

import (
	"testing"
)

const appConfigSub = "sub1"
const appConfigRG = "rg1"
const appConfigStore = "myappconfig"

func appConfigARMBase(ts interface{ URL string }) string {
	return ts.URL + "/subscriptions/" + appConfigSub + "/resourceGroups/" + appConfigRG + "/providers/Microsoft.AppConfiguration/configurationStores"
}

func TestAppConfigARMStoreCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := appConfigARMBase(ts) + "/" + appConfigStore

	// Create store
	resp := doRequest(t, "PUT", base, `{"location":"eastus","sku":{"name":"Standard"}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != appConfigStore {
		t.Fatalf("expected name=%s, got %v", appConfigStore, m["name"])
	}
	if m["type"] != "Microsoft.AppConfiguration/configurationStores" {
		t.Fatalf("expected correct type, got %v", m["type"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded, got %v", props["provisioningState"])
	}
	if props["endpoint"] == nil {
		t.Fatal("expected endpoint in properties")
	}

	// Get store
	resp2 := doRequest(t, "GET", base, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 200)

	// Update store (idempotent)
	resp3 := doRequest(t, "PUT", base, `{"location":"eastus"}`)
	defer resp3.Body.Close()
	expectStatus(t, resp3, 200)
}

func TestAppConfigARMStoreList(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := appConfigARMBase(ts)

	doRequest(t, "PUT", base+"/store-a", `{}`).Body.Close()
	doRequest(t, "PUT", base+"/store-b", `{}`).Body.Close()

	resp := doRequest(t, "GET", base, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	items := m["value"].([]interface{})
	if len(items) < 2 {
		t.Fatalf("expected at least 2 stores, got %d", len(items))
	}
}

func TestAppConfigARMStoreNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	resp := doRequest(t, "GET", appConfigARMBase(ts)+"/nonexistent", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestAppConfigARMStoreDelete(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := appConfigARMBase(ts) + "/store-del"

	doRequest(t, "PUT", base, `{}`).Body.Close()

	resp := doRequest(t, "DELETE", base, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 202)

	resp2 := doRequest(t, "GET", base, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 404)
}

func TestAppConfigARMResponseHasID(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := appConfigARMBase(ts) + "/mystore"

	resp := doRequest(t, "PUT", base, `{}`)
	defer resp.Body.Close()
	m := decodeJSON(t, resp)

	expectedID := "/subscriptions/" + appConfigSub + "/resourceGroups/" + appConfigRG + "/providers/Microsoft.AppConfiguration/configurationStores/mystore"
	if m["id"] != expectedID {
		t.Fatalf("expected id=%s, got %v", expectedID, m["id"])
	}
}

func TestAppConfigARMEndpointFormat(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	storeName := "teststoreendpt"
	base := appConfigARMBase(ts) + "/" + storeName

	resp := doRequest(t, "PUT", base, `{}`)
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	props := m["properties"].(map[string]interface{})

	expectedEndpoint := "https://" + storeName + ".azconfig.io"
	if props["endpoint"] != expectedEndpoint {
		t.Fatalf("expected endpoint=%s, got %v", expectedEndpoint, props["endpoint"])
	}
}

func TestAppConfigARMDoesNotBreakDataPlane(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/appconfig/mystore/kv"

	// Data-plane operations should still work
	resp := doRequest(t, "PUT", base+"/mykey", `{"value":"myvalue"}`)
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	if m["value"] != "myvalue" {
		t.Fatalf("expected value=myvalue, got %v", m["value"])
	}
}
