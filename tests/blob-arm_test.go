package tests

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/moabukar/miniblue/internal/server"
	"github.com/moabukar/miniblue/internal/storageauth"
)

const blobSub = "sub1"
const blobRG = "rg1"
const blobAcct = "mystorageacct"

func blobARMBase(ts *httptest.Server) string {
	return ts.URL + "/subscriptions/" + blobSub + "/resourceGroups/" + blobRG + "/providers/Microsoft.Storage/storageAccounts"
}

func TestBlobARMStorageAccountCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := blobARMBase(ts) + "/" + blobAcct

	// Create account (200 OK matches Azure ARM / azurerm SDK expectations)
	resp := doRequest(t, "PUT", base, `{"location":"eastus","sku":{"name":"Standard_LRS"},"kind":"StorageV2"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

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

func TestBlobARMStorageAccountListKeys(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := blobARMBase(ts) + "/keyacct"
	doRequest(t, "PUT", base, `{}`).Body.Close()

	lk := base + "/listKeys"
	resp := doRequest(t, "POST", lk, `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	keys, ok := m["keys"].([]interface{})
	if !ok || len(keys) != 2 {
		t.Fatalf("expected keys array of length 2, got %v", m["keys"])
	}
	k0 := keys[0].(map[string]interface{})
	if k0["keyName"] != "key1" {
		t.Fatalf("expected key1, got %v", k0["keyName"])
	}
	if k0["permissions"] != "Full" {
		t.Fatalf("expected Full permissions, got %v", k0["permissions"])
	}
	if k0["value"] == nil || k0["value"] == "" {
		t.Fatal("expected non-empty key value")
	}
}

func TestBlobARMStorageAccountListKeysNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	lk := blobARMBase(ts) + "/nosuchacct/listKeys"
	resp := doRequest(t, "POST", lk, "{}")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestBlobARMStorageAccountListInSubscription(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	rgBase := ts.URL + "/subscriptions/" + blobSub + "/resourceGroups/" + blobRG + "/providers/Microsoft.Storage/storageAccounts"
	doRequest(t, "PUT", rgBase+"/sublistacct", `{}`).Body.Close()

	subListURL := ts.URL + "/subscriptions/" + blobSub + "/providers/Microsoft.Storage/storageAccounts"
	resp := doRequest(t, "GET", subListURL, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	items, ok := m["value"].([]interface{})
	if !ok || len(items) < 1 {
		t.Fatalf("expected at least 1 account in subscription-scoped list, got %v", m["value"])
	}
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
	expectStatus(t, resp, 200)

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

	// Data-plane operations should still work (no ARM account => no shared-key requirement)
	doRequest(t, "PUT", base, "").Body.Close()

	resp := doRequest(t, "PUT", base+"/hello.txt", "data")
	resp.Body.Close()
	expectStatus(t, resp, 201)

	resp2 := doRequest(t, "GET", base+"/hello.txt", "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 200)
}

func TestBlobDataPlaneSharedKeyServiceProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acct := "armblobacct"
	doRequest(t, "PUT", blobARMBase(ts)+"/"+acct, `{}`).Body.Close()

	keyB64 := storageauth.DeterministicAccountKey(blobSub, blobRG, acct, "1")
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		t.Fatal(err)
	}
	u := ts.URL + "/blob/" + acct + "?comp=properties&restype=service"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("x-ms-date", time.Now().UTC().Format(http.TimeFormat))
	req.Header.Set("x-ms-version", "2020-10-02")
	if err := storageauth.SignBlobSharedKey(req, acct, key, false); err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<StorageServiceProperties>") {
		t.Fatalf("expected blob service properties XML, got %s", body)
	}
}

func TestBlobDataPlaneSharedKeyListContainers(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acct := "listcontacct"
	doRequest(t, "PUT", blobARMBase(ts)+"/"+acct, `{}`).Body.Close()

	keyB64 := storageauth.DeterministicAccountKey(blobSub, blobRG, acct, "1")
	key, _ := base64.StdEncoding.DecodeString(keyB64)
	putURL := ts.URL + "/blob/" + acct + "/c1"
	putReq, _ := http.NewRequest(http.MethodPut, putURL, nil)
	putReq.Header.Set("x-ms-date", time.Now().UTC().Format(http.TimeFormat))
	putReq.Header.Set("x-ms-version", "2020-10-02")
	if err := storageauth.SignBlobSharedKey(putReq, acct, key, false); err != nil {
		t.Fatal(err)
	}
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatal(err)
	}
	putResp.Body.Close()
	expectStatus(t, putResp, 201)
	u := ts.URL + "/blob/" + acct + "?comp=list"
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("x-ms-date", time.Now().UTC().Format(http.TimeFormat))
	req.Header.Set("x-ms-version", "2020-10-02")
	_ = storageauth.SignBlobSharedKey(req, acct, key, false)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<EnumerationResults") || !strings.Contains(string(body), "c1") {
		t.Fatalf("expected list containers XML with c1, got %s", body)
	}
}

func TestBlobDataPlaneSharedKeyRequiredAfterARM(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acct := "secureacct"
	doRequest(t, "PUT", blobARMBase(ts)+"/"+acct, `{}`).Body.Close()

	u := ts.URL + "/blob/" + acct + "?comp=properties&restype=service"
	resp, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	expectStatus(t, resp, 403)
}

func TestBlobDataPlaneSharedKeyRejectBadSignature(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acct := "badsigacct"
	doRequest(t, "PUT", blobARMBase(ts)+"/"+acct, `{}`).Body.Close()

	u := ts.URL + "/blob/" + acct + "?comp=properties&restype=service"
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("x-ms-date", time.Now().UTC().Format(http.TimeFormat))
	req.Header.Set("x-ms-version", "2020-10-02")
	req.Header.Set("Authorization", "SharedKey "+acct+":AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	expectStatus(t, resp, 403)
}

func TestBlobServiceFilterStillRegistersStorageAccountARM(t *testing.T) {
	t.Setenv("SERVICES", "blob")
	srv := server.New()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	base := ts.URL + "/subscriptions/" + blobSub + "/resourceGroups/" + blobRG + "/providers/Microsoft.Storage/storageAccounts/" + blobAcct
	resp := doRequest(t, "PUT", base, `{"location":"eastus","sku":{"name":"Standard_LRS"},"kind":"StorageV2"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	hr := doRequest(t, "GET", ts.URL+"/health", "")
	defer hr.Body.Close()
	h := decodeJSON(t, hr)
	for _, s := range h["services"].([]interface{}) {
		if s.(string) == "storageaccounts" {
			t.Fatal("with SERVICES=blob, health should not list storageaccounts (backward compatible service list)")
		}
	}
}

func TestStorageAccountsServiceWithoutBlobDataPlane(t *testing.T) {
	t.Setenv("SERVICES", "resourcegroups,storageaccounts")
	srv := server.New()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	base := ts.URL + "/subscriptions/" + blobSub + "/resourceGroups/" + blobRG + "/providers/Microsoft.Storage/storageAccounts/armonly"
	resp := doRequest(t, "PUT", base, `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	br := doRequest(t, "PUT", ts.URL+"/blob/x/y", "")
	defer br.Body.Close()
	expectStatus(t, br, 404)
}

func TestStorageLocalEndpointsDisabled(t *testing.T) {
	t.Setenv("MINIBLUE_STORAGE_ENDPOINTS", "")
	ts := setupServer()
	defer ts.Close()
	acctName := "azureendpointsacct"
	base := blobARMBase(ts) + "/" + acctName

	resp := doRequest(t, "PUT", base, `{}`)
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	props := m["properties"].(map[string]interface{})
	endpoints := props["primaryEndpoints"].(map[string]interface{})

	expectedBlob := "https://" + acctName + ".blob.core.windows.net/"
	if endpoints["blob"] != expectedBlob {
		t.Fatalf("expected blob endpoint=%s, got %v", expectedBlob, endpoints["blob"])
	}
	expectedQueue := "https://" + acctName + ".queue.core.windows.net/"
	if endpoints["queue"] != expectedQueue {
		t.Fatalf("expected queue endpoint=%s, got %v", expectedQueue, endpoints["queue"])
	}
	expectedTable := "https://" + acctName + ".table.core.windows.net/"
	if endpoints["table"] != expectedTable {
		t.Fatalf("expected table endpoint=%s, got %v", expectedTable, endpoints["table"])
	}
	expectedFile := "https://" + acctName + ".file.core.windows.net/"
	if endpoints["file"] != expectedFile {
		t.Fatalf("expected file endpoint=%s, got %v", expectedFile, endpoints["file"])
	}
}

func TestStorageLocalEndpointsEnabled(t *testing.T) {
	t.Setenv("MINIBLUE_STORAGE_ENDPOINT", "http://localhost:4566")
	ts := setupServer()
	defer ts.Close()
	acctName := "localendpointsacct"
	base := blobARMBase(ts) + "/" + acctName

	resp := doRequest(t, "PUT", base, `{}`)
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	props := m["properties"].(map[string]interface{})
	endpoints := props["primaryEndpoints"].(map[string]interface{})

	expectedBlob := "http://localhost:4566/blob/" + acctName + "/"
	if endpoints["blob"] != expectedBlob {
		t.Fatalf("expected blob endpoint=%s, got %v", expectedBlob, endpoints["blob"])
	}
	expectedQueue := "http://localhost:4566/queue/" + acctName + "/"
	if endpoints["queue"] != expectedQueue {
		t.Fatalf("expected queue endpoint=%s, got %v", expectedQueue, endpoints["queue"])
	}
	expectedTable := "http://localhost:4566/table/" + acctName + "/"
	if endpoints["table"] != expectedTable {
		t.Fatalf("expected table endpoint=%s, got %v", expectedTable, endpoints["table"])
	}
	expectedFile := "http://localhost:4566/file/" + acctName + "/"
	if endpoints["file"] != expectedFile {
		t.Fatalf("expected file endpoint=%s, got %v", expectedFile, endpoints["file"])
	}
}

func TestStorageFileServiceProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "fileacct"
	acctBase := blobARMBase(ts) + "/" + acctName
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	resp := doRequest(t, "GET", acctBase+"/fileServices/default", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["type"] != "Microsoft.Storage/storageAccounts/fileServices" {
		t.Fatalf("expected fileServices type, got %v", m["type"])
	}
	if m["name"] != "default" {
		t.Fatalf("expected name=default, got %v", m["name"])
	}
}

func TestStorageFileServicePropertiesNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	resp := doRequest(t, "GET", blobARMBase(ts)+"/nosuchacct/fileServices/default", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestStorageFileServiceSetProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "fileacctset"
	acctBase := blobARMBase(ts) + "/" + acctName
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	resp := doRequest(t, "PUT", acctBase+"/fileServices/default", `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["type"] != "Microsoft.Storage/storageAccounts/fileServices" {
		t.Fatalf("expected fileServices type, got %v", m["type"])
	}
}

func TestStorageFileServicePatchProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "fileacctpatch"
	acctBase := blobARMBase(ts) + "/" + acctName
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	resp := doRequest(t, "PATCH", acctBase+"/fileServices/default", `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["type"] != "Microsoft.Storage/storageAccounts/fileServices" {
		t.Fatalf("expected fileServices type, got %v", m["type"])
	}
}

func TestStorageQueueServiceProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "queueacct"
	acctBase := blobARMBase(ts) + "/" + acctName
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	resp := doRequest(t, "GET", acctBase+"/queueServices/default", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["type"] != "Microsoft.Storage/storageAccounts/queueServices" {
		t.Fatalf("expected queueServices type, got %v", m["type"])
	}
	if m["name"] != "default" {
		t.Fatalf("expected name=default, got %v", m["name"])
	}
}

func TestStorageQueueServicePropertiesNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	resp := doRequest(t, "GET", blobARMBase(ts)+"/nosuchacct/queueServices/default", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestStorageQueueServiceSetProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "queueacctset"
	acctBase := blobARMBase(ts) + "/" + acctName
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	resp := doRequest(t, "PUT", acctBase+"/queueServices/default", `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["type"] != "Microsoft.Storage/storageAccounts/queueServices" {
		t.Fatalf("expected queueServices type, got %v", m["type"])
	}
}

func TestStorageQueueServicePatchProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "queueacctpatch"
	acctBase := blobARMBase(ts) + "/" + acctName
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	resp := doRequest(t, "PATCH", acctBase+"/queueServices/default", `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["type"] != "Microsoft.Storage/storageAccounts/queueServices" {
		t.Fatalf("expected queueServices type, got %v", m["type"])
	}
}

func TestStorageTableServiceProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "tableacct"
	acctBase := blobARMBase(ts) + "/" + acctName
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	resp := doRequest(t, "GET", acctBase+"/tableServices/default", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["type"] != "Microsoft.Storage/storageAccounts/tableServices" {
		t.Fatalf("expected tableServices type, got %v", m["type"])
	}
	if m["name"] != "default" {
		t.Fatalf("expected name=default, got %v", m["name"])
	}
}

func TestStorageTableServicePropertiesNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	resp := doRequest(t, "GET", blobARMBase(ts)+"/nosuchacct/tableServices/default", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestStorageTableServiceSetProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "tableacctset"
	acctBase := blobARMBase(ts) + "/" + acctName
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	resp := doRequest(t, "PUT", acctBase+"/tableServices/default", `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["type"] != "Microsoft.Storage/storageAccounts/tableServices" {
		t.Fatalf("expected tableServices type, got %v", m["type"])
	}
}

func TestStorageTableServicePatchProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "tableacctpatch"
	acctBase := blobARMBase(ts) + "/" + acctName
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	resp := doRequest(t, "PATCH", acctBase+"/tableServices/default", `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["type"] != "Microsoft.Storage/storageAccounts/tableServices" {
		t.Fatalf("expected tableServices type, got %v", m["type"])
	}
}

func TestStorageBlobServiceProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "blobacct"
	acctBase := blobARMBase(ts) + "/" + acctName
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	resp := doRequest(t, "GET", acctBase+"/blobServices/default", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["type"] != "Microsoft.Storage/storageAccounts/blobServices" {
		t.Fatalf("expected blobServices type, got %v", m["type"])
	}
	if m["name"] != "default" {
		t.Fatalf("expected name=default, got %v", m["name"])
	}
}

func TestStorageBlobServicePropertiesNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	resp := doRequest(t, "GET", blobARMBase(ts)+"/nosuchacct/blobServices/default", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestStorageBlobServiceSetProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "blobacctset"
	acctBase := blobARMBase(ts) + "/" + acctName
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	resp := doRequest(t, "PUT", acctBase+"/blobServices/default", `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["type"] != "Microsoft.Storage/storageAccounts/blobServices" {
		t.Fatalf("expected blobServices type, got %v", m["type"])
	}
}

func TestStorageBlobServicePatchProperties(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctName := "blobacctpatch"
	acctBase := blobARMBase(ts) + "/" + acctName
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	resp := doRequest(t, "PATCH", acctBase+"/blobServices/default", `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["type"] != "Microsoft.Storage/storageAccounts/blobServices" {
		t.Fatalf("expected blobServices type, got %v", m["type"])
	}
}
