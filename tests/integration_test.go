package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moabukar/local-azure/internal/server"
)

func setupServer() *httptest.Server {
	srv := server.New()
	return httptest.NewServer(srv.Handler())
}

func TestHealthEndpoint(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["status"] != "running" {
		t.Fatalf("expected status=running, got %v", result["status"])
	}

	services, ok := result["services"].([]interface{})
	if !ok || len(services) == 0 {
		t.Fatal("expected services list")
	}
}

func TestResourceGroupCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	base := ts.URL + "/subscriptions/test-sub/resourcegroups"

	// Create
	body := `{"location": "eastus", "tags": {"env": "test"}}`
	req, _ := http.NewRequest("PUT", base+"/myRG", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Fatalf("create: expected 201, got %d", resp.StatusCode)
	}

	// Get
	resp, err = http.Get(base + "/myRG")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("get: expected 200, got %d", resp.StatusCode)
	}

	var rg map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&rg)

	if rg["name"] != "myRG" {
		t.Fatalf("expected name=myRG, got %v", rg["name"])
	}
	if rg["location"] != "eastus" {
		t.Fatalf("expected location=eastus, got %v", rg["location"])
	}
	if rg["type"] != "Microsoft.Resources/resourceGroups" {
		t.Fatalf("expected correct type, got %v", rg["type"])
	}

	// List
	resp, err = http.Get(base)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var list map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&list)
	items := list["value"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("list: expected 1 item, got %d", len(items))
	}

	// Delete
	req, _ = http.NewRequest("DELETE", base+"/myRG", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("delete: expected 200, got %d", resp.StatusCode)
	}

	// Get after delete - should 404
	resp, err = http.Get(base + "/myRG")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("get after delete: expected 404, got %d", resp.StatusCode)
	}
}

func TestKeyVaultSecretsCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	base := ts.URL + "/keyvault/testvault/secrets"

	// Set
	body := `{"value": "supersecret"}`
	req, _ := http.NewRequest("PUT", base+"/mykey", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var secret map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&secret)

	if secret["value"] != "supersecret" {
		t.Fatalf("expected value=supersecret, got %v", secret["value"])
	}
	if secret["id"] != "https://testvault.vault.azure.net/secrets/mykey" {
		t.Fatalf("expected correct id, got %v", secret["id"])
	}

	// Get
	resp, err = http.Get(base + "/mykey")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&secret)
	if secret["value"] != "supersecret" {
		t.Fatalf("get: expected value=supersecret, got %v", secret["value"])
	}

	// List
	resp, err = http.Get(base)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var list map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&list)
	items := list["value"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("list: expected 1, got %d", len(items))
	}

	// Delete
	req, _ = http.NewRequest("DELETE", base+"/mykey", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Verify deleted
	resp, err = http.Get(base + "/mykey")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

func TestBlobStorageCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	// Create container
	req, _ := http.NewRequest("PUT", ts.URL+"/blob/testaccount/testcontainer", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("create container: expected 201, got %d", resp.StatusCode)
	}

	// Upload blob
	data := []byte("hello world from tests")
	req, _ = http.NewRequest("PUT", ts.URL+"/blob/testaccount/testcontainer/test.txt", bytes.NewReader(data))
	req.Header.Set("Content-Type", "text/plain")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("upload blob: expected 201, got %d", resp.StatusCode)
	}

	// Download blob
	resp, err = http.Get(ts.URL + "/blob/testaccount/testcontainer/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	downloaded, _ := io.ReadAll(resp.Body)
	if string(downloaded) != "hello world from tests" {
		t.Fatalf("download: expected 'hello world from tests', got '%s'", string(downloaded))
	}

	// List blobs
	resp, err = http.Get(ts.URL + "/blob/testaccount/testcontainer")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var list map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&list)
	blobs := list["blobs"].([]interface{})
	if len(blobs) != 1 {
		t.Fatalf("list: expected 1 blob, got %d", len(blobs))
	}
}

func TestSubscriptionsAndTenants(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	// List subscriptions
	resp, err := http.Get(ts.URL + "/subscriptions")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	subs := result["value"].([]interface{})
	if len(subs) == 0 {
		t.Fatal("expected at least one subscription")
	}

	// List tenants
	resp, err = http.Get(ts.URL + "/tenants")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&result)
	tenants := result["value"].([]interface{})
	if len(tenants) == 0 {
		t.Fatal("expected at least one tenant")
	}
}

func TestManagedIdentityToken(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/metadata/identity/oauth2/token?resource=https://management.azure.com/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var token map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&token)

	if token["token_type"] != "Bearer" {
		t.Fatalf("expected Bearer token, got %v", token["token_type"])
	}
	if token["access_token"] == nil || token["access_token"] == "" {
		t.Fatal("expected non-empty access_token")
	}
}

func TestMetadataEndpoints(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/metadata/endpoints")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var endpoints map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&endpoints)

	if endpoints["resourceManagerEndpoint"] == nil {
		t.Fatal("expected resourceManagerEndpoint")
	}
	if endpoints["activeDirectoryEndpoint"] == nil {
		t.Fatal("expected activeDirectoryEndpoint")
	}
}

func TestAzureErrorFormat(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	// Request a non-existent resource group
	resp, err := http.Get(ts.URL + "/subscriptions/sub1/resourcegroups/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var azErr struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&azErr)

	if azErr.Error.Code != "ResourceNotFound" {
		t.Fatalf("expected error code ResourceNotFound, got %v", azErr.Error.Code)
	}
	if azErr.Error.Message == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestAzureResponseHeaders(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Check Azure standard headers
	if resp.Header.Get("x-ms-request-id") == "" {
		t.Fatal("expected x-ms-request-id header")
	}
	if resp.Header.Get("x-ms-correlation-request-id") == "" {
		t.Fatal("expected x-ms-correlation-request-id header")
	}
	if resp.Header.Get("x-ms-version") != "2023-11-03" {
		t.Fatalf("expected x-ms-version=2023-11-03, got %v", resp.Header.Get("x-ms-version"))
	}
}

func TestCosmosDBFullDocument(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	base := ts.URL + "/cosmosdb/testaccount/dbs/testdb/colls/users/docs"

	// Create with full body
	body := `{"id":"user1","name":"Mo","role":"CTO","email":"mo@test.com"}`
	resp, err := http.Post(base, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Fatalf("create: expected 201, got %d", resp.StatusCode)
	}

	// Get and verify all fields preserved
	resp, err = http.Get(base + "/user1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var doc map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&doc)

	if doc["name"] != "Mo" {
		t.Fatalf("expected name=Mo, got %v", doc["name"])
	}
	if doc["role"] != "CTO" {
		t.Fatalf("expected role=CTO, got %v", doc["role"])
	}
	if doc["email"] != "mo@test.com" {
		t.Fatalf("expected email=mo@test.com, got %v", doc["email"])
	}
}

func TestServiceBusCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	// Create queue
	req, _ := http.NewRequest("PUT", ts.URL+"/servicebus/testns/queues/orders", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("create queue: expected 201, got %d", resp.StatusCode)
	}

	// Send message
	resp, err = http.Post(ts.URL+"/servicebus/testns/queues/orders/messages",
		"application/json", bytes.NewBufferString(`{"body":"test order"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Receive
	resp, err = http.Get(ts.URL + "/servicebus/testns/queues/orders/messages/head")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var msg map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&msg)
	if msg["body"] != "test order" {
		t.Fatalf("expected body='test order', got %v", msg["body"])
	}
}

func TestEventGridCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.EventGrid/topics"

	// Create topic
	req, _ := http.NewRequest("PUT", base+"/myevents",
		bytes.NewBufferString(`{"location":"eastus"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Fatalf("create topic: expected 201, got %d", resp.StatusCode)
	}

	var topic map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&topic)
	if topic["type"] != "Microsoft.EventGrid/topics" {
		t.Fatalf("expected type=Microsoft.EventGrid/topics, got %v", topic["type"])
	}

	// Publish event
	resp, err = http.Post(ts.URL+"/eventgrid/myevents/events",
		"application/json",
		bytes.NewBufferString(`[{"id":"1","eventType":"test","subject":"test","data":{}}]`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("publish: expected 200, got %d", resp.StatusCode)
	}
}

func TestVNetAndSubnet(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks"

	// Create VNet
	req, _ := http.NewRequest("PUT", base+"/myVNet",
		bytes.NewBufferString(`{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("create vnet: expected 201, got %d", resp.StatusCode)
	}

	// Create Subnet
	req, _ = http.NewRequest("PUT", base+"/myVNet/subnets/frontend",
		bytes.NewBufferString(`{"properties":{"addressPrefix":"10.0.1.0/24"}}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var subnet map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&subnet)
	if subnet["name"] != "frontend" {
		t.Fatalf("expected subnet name=frontend, got %v", subnet["name"])
	}
}

func TestDNSZoneAndRecord(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/dnsZones"

	// Create zone
	req, _ := http.NewRequest("PUT", base+"/example.com",
		bytes.NewBufferString(`{"location":"global"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("create zone: expected 201, got %d", resp.StatusCode)
	}

	// Create A record
	req, _ = http.NewRequest("PUT", base+"/example.com/A/www",
		bytes.NewBufferString(`{"properties":{"TTL":300,"ARecords":[{"ipv4Address":"1.2.3.4"}]}}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("create record: expected 201, got %d", resp.StatusCode)
	}
}

func TestACRRegistry(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ContainerRegistry/registries"

	// Create
	req, _ := http.NewRequest("PUT", base+"/myreg",
		bytes.NewBufferString(`{"location":"eastus"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var reg map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&reg)

	props := reg["properties"].(map[string]interface{})
	if props["loginServer"] != "myreg.azurecr.io" {
		t.Fatalf("expected loginServer=myreg.azurecr.io, got %v", props["loginServer"])
	}
}

func TestAppConfigCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	base := ts.URL + "/appconfig/mystore/kv"

	// Set
	req, _ := http.NewRequest("PUT", base+"/mykey",
		bytes.NewBufferString(`{"value":"myvalue"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var kv map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&kv)
	if kv["value"] != "myvalue" {
		t.Fatalf("expected value=myvalue, got %v", kv["value"])
	}
	if kv["etag"] == nil {
		t.Fatal("expected etag")
	}

	// Get after delete should 404
	req, _ = http.NewRequest("DELETE", base+"/mykey", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp, err = http.Get(base + "/mykey")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}
