package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moabukar/local-azure/internal/server"
)

func setupServer() *httptest.Server {
	srv := server.New()
	return httptest.NewServer(srv.Handler())
}

// ---------- helpers ----------

type azError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func doRequest(t *testing.T, method, url string, body string) *http.Response {
	t.Helper()
	var req *http.Request
	if body != "" {
		req, _ = http.NewRequest(method, url, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func expectStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("expected status %d, got %d", want, resp.StatusCode)
	}
}

func decodeJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&m)
	return m
}

func decodeError(t *testing.T, resp *http.Response) azError {
	t.Helper()
	var e azError
	json.NewDecoder(resp.Body).Decode(&e)
	return e
}

// ---------- Health ----------

func TestHealthEndpoint(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/health", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["status"] != "running" {
		t.Fatalf("expected status=running, got %v", m["status"])
	}
	if m["version"] != "0.1.0" {
		t.Fatalf("expected version=0.1.0, got %v", m["version"])
	}
}

// ---------- Azure Response Headers ----------

func TestAzureResponseHeaders(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/health", "")
	defer resp.Body.Close()

	headers := []string{"x-ms-request-id", "x-ms-correlation-request-id", "x-ms-version"}
	for _, h := range headers {
		if resp.Header.Get(h) == "" {
			t.Fatalf("missing header: %s", h)
		}
	}
	if resp.Header.Get("x-ms-version") != "2023-11-03" {
		t.Fatalf("wrong x-ms-version: %s", resp.Header.Get("x-ms-version"))
	}
}

// ---------- API Versioning ----------

func TestAPIVersionRequired(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	// ARM endpoint without api-version should fail
	resp := doRequest(t, "GET", ts.URL+"/subscriptions/sub1/resourcegroups", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 400)

	e := decodeError(t, resp)
	if e.Error.Code != "MissingApiVersionParameter" {
		t.Fatalf("expected MissingApiVersionParameter, got %s", e.Error.Code)
	}
}

func TestAPIVersionAccepted(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	// With api-version should work
	resp := doRequest(t, "GET", ts.URL+"/subscriptions/sub1/resourcegroups?api-version=2023-07-01", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
}

// ---------- Resource Groups CRUD ----------

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

// ---------- Blob Storage CRUD ----------

func TestBlobStorageCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/blob/myaccount"

	// Create container
	resp := doRequest(t, "PUT", base+"/mycontainer", "")
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Upload blob
	resp = doRequest(t, "PUT", base+"/mycontainer/hello.txt", "Hello World!")
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Download
	resp = doRequest(t, "GET", base+"/mycontainer/hello.txt", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	if buf.String() != "Hello World!" {
		t.Fatalf("expected 'Hello World!', got '%s'", buf.String())
	}

	// Verify content-length header
	if resp.Header.Get("Content-Length") != "12" {
		t.Fatalf("expected Content-Length=12, got %s", resp.Header.Get("Content-Length"))
	}
}

func TestBlobListContentLength(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/blob/myaccount/mycontainer"

	doRequest(t, "PUT", base, "").Body.Close()
	doRequest(t, "PUT", base+"/test.txt", "abcdef").Body.Close()

	resp := doRequest(t, "GET", base, "")
	defer resp.Body.Close()

	var result struct {
		Blobs []struct {
			Name       string `json:"name"`
			Properties struct {
				ContentLength string `json:"contentLength"`
			} `json:"properties"`
		} `json:"blobs"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Blobs) != 1 {
		t.Fatalf("expected 1 blob, got %d", len(result.Blobs))
	}
	if result.Blobs[0].Properties.ContentLength != "6" {
		t.Fatalf("expected contentLength=6, got %s", result.Blobs[0].Properties.ContentLength)
	}
}

func TestBlobNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/blob/acct/container/nope.txt", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestBlobDelete(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/blob/myaccount/mycontainer"

	doRequest(t, "PUT", base, "").Body.Close()
	doRequest(t, "PUT", base+"/file.txt", "data").Body.Close()

	resp := doRequest(t, "DELETE", base+"/file.txt", "")
	resp.Body.Close()
	expectStatus(t, resp, 202)

	resp = doRequest(t, "GET", base+"/file.txt", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

// ---------- Key Vault CRUD ----------

func TestKeyVaultCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/keyvault/myvault/secrets"

	// Set secret
	resp := doRequest(t, "PUT", base+"/db-pass", `{"value":"supersecret"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["value"] != "supersecret" {
		t.Fatalf("expected value=supersecret, got %v", m["value"])
	}

	// Get
	resp = doRequest(t, "GET", base+"/db-pass", "")
	m = decodeJSON(t, resp)
	if m["value"] != "supersecret" {
		t.Fatalf("get: expected value=supersecret, got %v", m["value"])
	}

	// List
	resp = doRequest(t, "GET", base, "")
	list := decodeJSON(t, resp)
	if list["value"] == nil {
		t.Fatal("expected value array in list response")
	}

	// Delete
	resp = doRequest(t, "DELETE", base+"/db-pass", "")
	resp.Body.Close()

	// Should 404 now
	resp = doRequest(t, "GET", base+"/db-pass", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

// ---------- Cosmos DB ----------

func TestCosmosDBFullDocument(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/cosmosdb/myaccount/dbs/mydb/colls/users/docs"

	resp := doRequest(t, "POST", base, `{"id":"u1","name":"Mo","role":"CTO","email":"mo@test.com"}`)
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Get - verify all fields
	resp = doRequest(t, "GET", base+"/u1", "")
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	if m["name"] != "Mo" {
		t.Fatalf("expected name=Mo, got %v", m["name"])
	}
	if m["role"] != "CTO" {
		t.Fatalf("expected role=CTO, got %v", m["role"])
	}
}

func TestCosmosDBReplace(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/cosmosdb/myaccount/dbs/mydb/colls/users/docs"

	doRequest(t, "POST", base, `{"id":"u1","name":"Mo"}`).Body.Close()

	// Replace
	resp := doRequest(t, "PUT", base+"/u1", `{"name":"Mo Updated","role":"Founder"}`)
	resp.Body.Close()
	expectStatus(t, resp, 200)

	// Verify update
	resp = doRequest(t, "GET", base+"/u1", "")
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	if m["name"] != "Mo Updated" {
		t.Fatalf("expected name='Mo Updated', got %v", m["name"])
	}
}

func TestCosmosDBDelete(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/cosmosdb/myaccount/dbs/mydb/colls/users/docs"

	doRequest(t, "POST", base, `{"id":"u1","name":"Mo"}`).Body.Close()

	resp := doRequest(t, "DELETE", base+"/u1", "")
	resp.Body.Close()
	expectStatus(t, resp, 204)

	resp = doRequest(t, "GET", base+"/u1", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

// ---------- Subscriptions + Tenants ----------

func TestSubscriptionsAndTenants(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/subscriptions?api-version=2022-12-01", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	subs := m["value"].([]interface{})
	if len(subs) == 0 {
		t.Fatal("expected at least 1 subscription")
	}

	resp = doRequest(t, "GET", ts.URL+"/tenants?api-version=2022-12-01", "")
	m = decodeJSON(t, resp)
	tenants := m["value"].([]interface{})
	if len(tenants) == 0 {
		t.Fatal("expected at least 1 tenant")
	}
}

// ---------- Managed Identity ----------

func TestManagedIdentityToken(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/metadata/identity/oauth2/token?resource=https://management.azure.com/", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["token_type"] != "Bearer" {
		t.Fatalf("expected token_type=Bearer, got %v", m["token_type"])
	}
	if m["access_token"] == nil || m["access_token"] == "" {
		t.Fatal("expected non-empty access_token")
	}
}

// ---------- Metadata ----------

func TestMetadataEndpoints(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/metadata/endpoints", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
}

// ---------- Service Bus ----------

func TestServiceBusQueueConflict(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	url := ts.URL + "/servicebus/myns/queues/orders"

	// First create
	resp := doRequest(t, "PUT", url, "")
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Second create - should 409
	resp = doRequest(t, "PUT", url, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 409)

	e := decodeError(t, resp)
	if e.Error.Code != "Conflict" {
		t.Fatalf("expected Conflict, got %s", e.Error.Code)
	}
}

func TestServiceBusSendToNonexistentQueue(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "POST", ts.URL+"/servicebus/myns/queues/nope/messages", `{"body":"test"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestServiceBusSendAndReceive(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/servicebus/myns/queues/orders"

	doRequest(t, "PUT", base, "").Body.Close()

	// Send 2 messages
	doRequest(t, "POST", base+"/messages", `{"body":"order-001"}`).Body.Close()
	doRequest(t, "POST", base+"/messages", `{"body":"order-002"}`).Body.Close()

	// Receive head
	resp := doRequest(t, "GET", base+"/messages/head", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["body"] == nil {
		t.Fatal("expected message body")
	}
}

func TestServiceBusDeleteAndSend(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/servicebus/myns/queues/temp"

	doRequest(t, "PUT", base, "").Body.Close()

	// Delete
	resp := doRequest(t, "DELETE", base, "")
	resp.Body.Close()
	expectStatus(t, resp, 200)

	// Send to deleted queue should 404
	resp = doRequest(t, "POST", base+"/messages", `{"body":"test"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

// ---------- Storage Queue ----------

func TestStorageQueueConflict(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	url := ts.URL + "/queue/myaccount/myqueue"

	resp := doRequest(t, "PUT", url, "")
	resp.Body.Close()
	expectStatus(t, resp, 201)

	resp = doRequest(t, "PUT", url, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 409)
}

func TestStorageQueueMessageLifecycle(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/queue/myaccount/q1"

	doRequest(t, "PUT", base, "").Body.Close()

	// Send
	resp := doRequest(t, "POST", base+"/messages", `{"messageText":"hello"}`)
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Receive
	resp = doRequest(t, "GET", base+"/messages", "")
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	msgs := m["messages"].([]interface{})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	// Clear
	resp = doRequest(t, "DELETE", base+"/messages", "")
	resp.Body.Close()
	expectStatus(t, resp, 204)

	// Receive again - empty
	resp = doRequest(t, "GET", base+"/messages", "")
	m = decodeJSON(t, resp)
	msgs2, _ := m["messages"].([]interface{})
	if len(msgs2) != 0 {
		t.Fatalf("expected 0 messages after clear, got %d", len(msgs2))
	}
}

// ---------- Event Grid ----------

func TestEventGridCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.EventGrid/topics"
	av := "?api-version=2023-12-15-preview"

	// Create
	resp := doRequest(t, "PUT", base+"/myevents"+av, `{"location":"eastus"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded, got %v", props["provisioningState"])
	}

	// Update (same URL) should return 200
	resp = doRequest(t, "PUT", base+"/myevents"+av, `{"location":"eastus"}`)
	resp.Body.Close()
	expectStatus(t, resp, 200)

	// Publish
	resp = doRequest(t, "POST", ts.URL+"/eventgrid/myevents/events",
		`[{"id":"1","eventType":"test","subject":"test","data":{}}]`)
	resp.Body.Close()
	expectStatus(t, resp, 200)

	// Delete
	resp = doRequest(t, "DELETE", base+"/myevents"+av, "")
	resp.Body.Close()
	expectStatus(t, resp, 202)

	// Get deleted - 404
	resp = doRequest(t, "GET", base+"/myevents"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

// ---------- VNet + Subnet ----------

func TestVNetSubnetLifecycle(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks"
	av := "?api-version=2023-09-01"

	// Create VNet
	resp := doRequest(t, "PUT", base+"/vnet1"+av,
		`{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`)
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Create Subnet
	resp = doRequest(t, "PUT", base+"/vnet1/subnets/web"+av,
		`{"properties":{"addressPrefix":"10.0.1.0/24"}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != "web" {
		t.Fatalf("expected subnet name=web, got %v", m["name"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded")
	}

	// List subnets
	resp = doRequest(t, "GET", base+"/vnet1/subnets"+av, "")
	list := decodeJSON(t, resp)
	subnets := list["value"].([]interface{})
	if len(subnets) != 1 {
		t.Fatalf("expected 1 subnet, got %d", len(subnets))
	}

	// Delete VNet cascades subnets
	doRequest(t, "DELETE", base+"/vnet1"+av, "").Body.Close()
	resp = doRequest(t, "GET", base+"/vnet1/subnets/web"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestSubnetOnNonexistentVNet(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "PUT", base+"/nope/subnets/web"+av,
		`{"properties":{"addressPrefix":"10.0.1.0/24"}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

// ---------- DNS ----------

func TestDNSZoneAndRecord(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/dnsZones"
	av := "?api-version=2023-07-01-preview"

	// Create zone
	resp := doRequest(t, "PUT", base+"/example.com"+av, `{"location":"global"}`)
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Create A record
	resp = doRequest(t, "PUT", base+"/example.com/A/www"+av,
		`{"properties":{"TTL":300,"ARecords":[{"ipv4Address":"1.2.3.4"}]}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	// Delete zone cascades records
	doRequest(t, "DELETE", base+"/example.com"+av, "").Body.Close()
	resp = doRequest(t, "GET", base+"/example.com/A/www"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestDNSRecordOnNonexistentZone(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/dnsZones"

	resp := doRequest(t, "PUT", base+"/nope.com/A/www?api-version=2023-07-01",
		`{"properties":{"TTL":60}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

// ---------- ACR ----------

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

// ---------- App Config ----------

func TestAppConfigCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/appconfig/mystore/kv"

	// Set
	resp := doRequest(t, "PUT", base+"/db-host", `{"value":"localhost:5432"}`)
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	if m["value"] != "localhost:5432" {
		t.Fatalf("expected value=localhost:5432, got %v", m["value"])
	}
	if m["etag"] == nil {
		t.Fatal("expected etag")
	}

	// Delete then get = 404
	doRequest(t, "DELETE", base+"/db-host", "").Body.Close()
	resp = doRequest(t, "GET", base+"/db-host", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

// ---------- Functions ----------

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

// ---------- Table Storage ----------

func TestTableStorageCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/table/myaccount/users"

	// Create table
	resp := doRequest(t, "POST", base, "")
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Insert entity
	resp = doRequest(t, "PUT", base+"/partition1/row1",
		`{"properties":{"name":"Mo","age":30}}`)
	resp.Body.Close()
	expectStatus(t, resp, 204)

	// Get entity
	resp = doRequest(t, "GET", base+"/partition1/row1", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["PartitionKey"] != "partition1" {
		t.Fatalf("expected PartitionKey=partition1, got %v", m["PartitionKey"])
	}

	// Delete + 404
	doRequest(t, "DELETE", base+"/partition1/row1", "").Body.Close()
	resp = doRequest(t, "GET", base+"/partition1/row1", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}
