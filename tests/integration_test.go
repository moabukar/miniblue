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
