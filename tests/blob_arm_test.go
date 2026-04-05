package tests

import (
	"testing"
)

const blobSub = "sub1"
const blobRG = "rg1"
const blobAcct = "mystorageacct"

func blobARMBase(ts interface{ URL string }) string {
	return ts.URL + "/subscriptions/" + blobSub + "/resourceGroups/" + blobRG + "/providers/Microsoft.Storage/storageAccounts"
}

func TestBlobARMStorageAccountCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := blobARMBase(ts) + "/" + blobAcct

	// Create account
	resp := doRequest(t, "PUT", base, `{"location":"eastus","sku":{"name":"Standard_LRS"},"kind":"StorageV2"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != blobAcct {
		t.Fatalf("expected name=%s, got %v", blobAcct, m["name"])
	}
	if m["type"] != "Microsoft.Storage/storageAccounts" {
		t.Fatalf("expected correct type, got %v", m["type"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded, got %v", props["provisioningState"])
	}
	endpoints := props["primaryEndpoints"].(map[string]interface{})
	if endpoints["blob"] == nil {
		t.Fatal("expected blob endpoint in primaryEndpoints")
	}

	// Get account
	resp2 := doRequest(t, "GET", base, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 200)

	// Update account (idempotent)
	resp3 := doRequest(t, "PUT", base, `{"location":"eastus"}`)
	defer resp3.Body.Close()
	expectStatus(t, resp3, 200)
}

func TestBlobARMStorageAccountList(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := blobARMBase(ts)

	doRequest(t, "PUT", base+"/acct-a", `{}`).Body.Close()
	doRequest(t, "PUT", base+"/acct-b", `{}`).Body.Close()

	resp := doRequest(t, "GET", base, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	items := m["value"].([]interface{})
	if len(items) < 2 {
		t.Fatalf("expected at least 2 accounts, got %d", len(items))
	}
}

func TestBlobARMStorageAccountNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	resp := doRequest(t, "GET", blobARMBase(ts)+"/nonexistent", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestBlobARMStorageAccountDelete(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := blobARMBase(ts) + "/acct-del"

	doRequest(t, "PUT", base, `{}`).Body.Close()

	resp := doRequest(t, "DELETE", base, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 202)

	resp2 := doRequest(t, "GET", base, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 404)
}

func TestBlobARMContainerCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctBase := blobARMBase(ts) + "/" + blobAcct
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	cBase := acctBase + "/blobServices/default/containers/mycontainer"

	// Create container
	resp := doRequest(t, "PUT", cBase, `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != "mycontainer" {
		t.Fatalf("expected name=mycontainer, got %v", m["name"])
	}
	if m["type"] != "Microsoft.Storage/storageAccounts/blobServices/containers" {
		t.Fatalf("expected correct type, got %v", m["type"])
	}

	// Get container
	resp2 := doRequest(t, "GET", cBase, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 200)

	// List containers
	resp3 := doRequest(t, "GET", acctBase+"/blobServices/default/containers", "")
	defer resp3.Body.Close()
	expectStatus(t, resp3, 200)
	ml := decodeJSON(t, resp3)
	if len(ml["value"].([]interface{})) == 0 {
		t.Fatal("expected at least 1 container in list")
	}

	// Delete container
	resp4 := doRequest(t, "DELETE", cBase, "")
	defer resp4.Body.Close()
	expectStatus(t, resp4, 202)

	resp5 := doRequest(t, "GET", cBase, "")
	defer resp5.Body.Close()
	expectStatus(t, resp5, 404)
}

func TestBlobARMResponseHasID(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := blobARMBase(ts) + "/myacct"

	resp := doRequest(t, "PUT", base, `{}`)
	defer resp.Body.Close()
	m := decodeJSON(t, resp)

	expectedID := "/subscriptions/" + blobSub + "/resourceGroups/" + blobRG + "/providers/Microsoft.Storage/storageAccounts/myacct"
	if m["id"] != expectedID {
		t.Fatalf("expected id=%s, got %v", expectedID, m["id"])
	}
}

func TestBlobARMPrimaryEndpointFormat(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "endpttestacct"
	base := blobARMBase(ts) + "/" + acctName

	resp := doRequest(t, "PUT", base, `{}`)
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	props := m["properties"].(map[string]interface{})
	endpoints := props["primaryEndpoints"].(map[string]interface{})

	expected := "https://" + acctName + ".blob.core.windows.net/"
	if endpoints["blob"] != expected {
		t.Fatalf("expected blob endpoint=%s, got %v", expected, endpoints["blob"])
	}
}

func TestBlobARMDoesNotBreakDataPlane(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/blob/myaccount/mycontainer"

	// Data-plane operations should still work
	doRequest(t, "PUT", base, "").Body.Close()

	resp := doRequest(t, "PUT", base+"/hello.txt", "data")
	resp.Body.Close()
	expectStatus(t, resp, 201)

	resp2 := doRequest(t, "GET", base+"/hello.txt", "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 200)
}
