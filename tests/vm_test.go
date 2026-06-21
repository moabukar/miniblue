package tests

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// VM API tests run in forced stub mode (MINIBLUE_DISABLE_DOCKER) so they are
// deterministic on machines with and without Docker. Docker-backed behavior
// is covered by tests/integration/vm_docker_test.go.

const vmAV = "?api-version=2024-07-01"

func setupVMServer(t *testing.T) *httptest.Server {
	t.Setenv("MINIBLUE_DISABLE_DOCKER", "1")
	return setupServer()
}

func createRG(t *testing.T, ts *httptest.Server, sub, rg string) {
	t.Helper()
	resp := doRequest(t, "PUT", ts.URL+"/subscriptions/"+sub+"/resourcegroups/"+rg+vmAV,
		`{"location":"eastus"}`)
	resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		t.Fatalf("failed to create resource group: status %d", resp.StatusCode)
	}
}

func vmURL(ts *httptest.Server, sub, rg, name string) string {
	return ts.URL + "/subscriptions/" + sub + "/resourceGroups/" + rg +
		"/providers/Microsoft.Compute/virtualMachines/" + name
}

func TestVMCreateGetUpdate(t *testing.T) {
	ts := setupVMServer(t)
	defer ts.Close()
	createRG(t, ts, "sub1", "rg1")

	resp := doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "web")+vmAV,
		`{"location":"westus","properties":{"miniblue":{"image":"ubuntu:24.04"},"hardwareProfile":{"vmSize":"Standard_B2s"}}}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()

	if m["name"] != "web" || m["type"] != "Microsoft.Compute/virtualMachines" {
		t.Fatalf("unexpected resource identity: %v / %v", m["name"], m["type"])
	}
	if m["location"] != "westus" {
		t.Fatalf("expected location westus, got %v", m["location"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState Succeeded, got %v", props["provisioningState"])
	}
	mb := props["miniblue"].(map[string]interface{})
	if mb["powerState"] != "running" {
		t.Fatalf("expected powerState running, got %v", mb["powerState"])
	}
	if mb["containerName"] != "miniblue-vm-rg1-web" {
		t.Fatalf("unexpected containerName: %v", mb["containerName"])
	}
	for k := range m {
		if strings.HasPrefix(k, "_") {
			t.Fatalf("internal field %q leaked into the API response", k)
		}
	}

	// Idempotent PUT: update returns 200.
	resp = doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "web")+vmAV, `{"location":"westus"}`)
	expectStatus(t, resp, 200)
	resp.Body.Close()

	resp = doRequest(t, "GET", vmURL(ts, "sub1", "rg1", "web")+vmAV, "")
	expectStatus(t, resp, 200)
	resp.Body.Close()
}

func TestVMCreateMissingResourceGroup(t *testing.T) {
	ts := setupVMServer(t)
	defer ts.Close()

	resp := doRequest(t, "PUT", vmURL(ts, "sub1", "nope", "web")+vmAV, `{"location":"eastus"}`)
	expectStatus(t, resp, 404)
	e := decodeError(t, resp)
	resp.Body.Close()
	if e.Error.Code != "ResourceGroupNotFound" {
		t.Fatalf("expected ResourceGroupNotFound, got %s", e.Error.Code)
	}
}

func TestVMImageMapping(t *testing.T) {
	ts := setupVMServer(t)
	defer ts.Close()
	createRG(t, ts, "sub1", "rg1")

	// No image reference at all → documented default.
	resp := doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "defaulted")+vmAV, `{"location":"eastus"}`)
	expectStatus(t, resp, 201)
	m := decodeJSON(t, resp)
	resp.Body.Close()
	mb := m["properties"].(map[string]interface{})["miniblue"].(map[string]interface{})
	if mb["image"] != "ubuntu:24.04" {
		t.Fatalf("expected default image ubuntu:24.04, got %v", mb["image"])
	}

	// Mappable ARM imageReference.
	resp = doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "jammy")+vmAV,
		`{"location":"eastus","properties":{"storageProfile":{"imageReference":{"publisher":"Canonical","offer":"0001-com-ubuntu-server-jammy","sku":"22_04-lts"}}}}`)
	expectStatus(t, resp, 201)
	m = decodeJSON(t, resp)
	resp.Body.Close()
	mb = m["properties"].(map[string]interface{})["miniblue"].(map[string]interface{})
	if mb["image"] != "ubuntu:22.04" {
		t.Fatalf("expected ubuntu:22.04, got %v", mb["image"])
	}

	// Present but unmappable reference → 400.
	resp = doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "weird")+vmAV,
		`{"location":"eastus","properties":{"storageProfile":{"imageReference":{"publisher":"MicrosoftWindowsServer","offer":"WindowsServer","sku":"2022-datacenter"}}}}`)
	expectStatus(t, resp, 400)
	resp.Body.Close()
}

func TestVMListAndDelete(t *testing.T) {
	ts := setupVMServer(t)
	defer ts.Close()
	createRG(t, ts, "sub1", "rg1")
	createRG(t, ts, "sub1", "rg2")

	for _, name := range []string{"a", "b"} {
		resp := doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", name)+vmAV, `{"location":"eastus"}`)
		expectStatus(t, resp, 201)
		resp.Body.Close()
	}
	resp := doRequest(t, "PUT", vmURL(ts, "sub1", "rg2", "c")+vmAV, `{"location":"eastus"}`)
	expectStatus(t, resp, 201)
	resp.Body.Close()

	// Resource-group scoped list.
	resp = doRequest(t, "GET", ts.URL+"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines"+vmAV, "")
	expectStatus(t, resp, 200)
	m := decodeJSON(t, resp)
	resp.Body.Close()
	if got := len(m["value"].([]interface{})); got != 2 {
		t.Fatalf("expected 2 VMs in rg1, got %d", got)
	}

	// Subscription-wide list.
	resp = doRequest(t, "GET", ts.URL+"/subscriptions/sub1/providers/Microsoft.Compute/virtualMachines"+vmAV, "")
	expectStatus(t, resp, 200)
	m = decodeJSON(t, resp)
	resp.Body.Close()
	if got := len(m["value"].([]interface{})); got != 3 {
		t.Fatalf("expected 3 VMs in subscription, got %d", got)
	}

	resp = doRequest(t, "DELETE", vmURL(ts, "sub1", "rg1", "a")+vmAV, "")
	expectStatus(t, resp, 204)
	resp.Body.Close()
	resp = doRequest(t, "GET", vmURL(ts, "sub1", "rg1", "a")+vmAV, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
	resp = doRequest(t, "DELETE", vmURL(ts, "sub1", "rg1", "a")+vmAV, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
}

func TestVMRuntimeOperationsRequireDocker(t *testing.T) {
	ts := setupVMServer(t)
	defer ts.Close()
	createRG(t, ts, "sub1", "rg1")
	resp := doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "web")+vmAV, `{"location":"eastus"}`)
	expectStatus(t, resp, 201)
	resp.Body.Close()

	// Every Docker-dependent operation returns 409 DockerUnavailable in stub mode.
	checks := []struct {
		method, url, body string
	}{
		{"POST", vmURL(ts, "sub1", "rg1", "web") + "/start" + vmAV, ""},
		{"POST", vmURL(ts, "sub1", "rg1", "web") + "/powerOff" + vmAV, ""},
		{"POST", vmURL(ts, "sub1", "rg1", "web") + "/restart" + vmAV, ""},
		{"POST", vmURL(ts, "sub1", "rg1", "web") + "/runCommand" + vmAV, `{"commandId":"RunShellScript","script":["echo hi"]}`},
		{"GET", vmURL(ts, "sub1", "rg1", "web") + "/logs" + vmAV, ""},
		{"PUT", vmURL(ts, "sub1", "rg1", "web") + "/services/api" + vmAV, `{"properties":{"image":"nginx:alpine"}}`},
	}
	for _, c := range checks {
		resp := doRequest(t, c.method, c.url, c.body)
		if resp.StatusCode != 409 {
			t.Fatalf("%s %s: expected 409 in stub mode, got %d", c.method, c.url, resp.StatusCode)
		}
		e := decodeError(t, resp)
		resp.Body.Close()
		if e.Error.Code != "DockerUnavailable" {
			t.Fatalf("%s %s: expected DockerUnavailable, got %s", c.method, c.url, e.Error.Code)
		}
	}

	// Record-level operations still work without Docker (stub mode contract).
	resp = doRequest(t, "GET", vmURL(ts, "sub1", "rg1", "web")+"/services"+vmAV, "")
	expectStatus(t, resp, 200)
	resp.Body.Close()
}

func TestVMServiceValidation(t *testing.T) {
	ts := setupVMServer(t)
	defer ts.Close()
	createRG(t, ts, "sub1", "rg1")
	resp := doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "web")+vmAV, `{"location":"eastus"}`)
	expectStatus(t, resp, 201)
	resp.Body.Close()

	// Deploy onto a missing VM → 404 (checked before Docker availability).
	resp = doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "ghost")+"/services/api"+vmAV,
		`{"properties":{"image":"nginx:alpine"}}`)
	expectStatus(t, resp, 404)
	resp.Body.Close()

	// Invalid service name → 400 (checked before Docker availability).
	resp = doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "web")+"/services/Bad_Name"+vmAV,
		`{"properties":{"image":"nginx:alpine"}}`)
	expectStatus(t, resp, 400)
	resp.Body.Close()

	// Missing image → 400.
	resp = doRequest(t, "PUT", vmURL(ts, "sub1", "rg1", "web")+"/services/api"+vmAV,
		`{"properties":{}}`)
	expectStatus(t, resp, 400)
	resp.Body.Close()
}

func TestVMResourceGroupCascade(t *testing.T) {
	ts := setupVMServer(t)
	defer ts.Close()
	createRG(t, ts, "sub1", "doomed")
	resp := doRequest(t, "PUT", vmURL(ts, "sub1", "doomed", "web")+vmAV, `{"location":"eastus"}`)
	expectStatus(t, resp, 201)
	resp.Body.Close()

	resp = doRequest(t, "DELETE", ts.URL+"/subscriptions/sub1/resourcegroups/doomed"+vmAV, "")
	resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 202 && resp.StatusCode != 204 {
		t.Fatalf("resource group delete failed: %d", resp.StatusCode)
	}

	resp = doRequest(t, "GET", vmURL(ts, "sub1", "doomed", "web")+vmAV, "")
	expectStatus(t, resp, 404)
	resp.Body.Close()
}
