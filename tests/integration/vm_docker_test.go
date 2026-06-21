package integration

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Docker-backed e2e for the VM service: create → deploy two services → logs →
// runCommand → power → fail-fast → delete. Runs against the package's shared
// miniblue httptest server; Docker is guaranteed by this package's TestMain
// (testcontainers).
//
// The in-container token e2e is NOT exercised here: the httptest server binds
// 127.0.0.1 on a random port, which is unreachable via host.docker.internal.
// The token flow is covered at the HTTP level in tests/vmidentity_test.go and
// end-to-end (real daemon, port 4566) by the feature quickstart.

const vmAPIVersion = "?api-version=2024-07-01"

func vmDoJSON(t *testing.T, method, url, body string) (*http.Response, map[string]interface{}) {
	t.Helper()
	var req *http.Request
	if body != "" {
		req, _ = http.NewRequest(method, url, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var m map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&m)
	return resp, m
}

func vmDoText(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return resp, sb.String()
}

func TestVMDockerLifecycle(t *testing.T) {
	rgURL := serverURL + "/subscriptions/sub1/resourcegroups/vmrg" + vmAPIVersion
	resp, _ := vmDoJSON(t, "PUT", rgURL, `{"location":"eastus"}`)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		t.Fatalf("resource group create failed: %d", resp.StatusCode)
	}

	base := serverURL + "/subscriptions/sub1/resourceGroups/vmrg/providers/Microsoft.Compute/virtualMachines"
	vmURL := base + "/web" + vmAPIVersion

	// --- create: VM running, backed by a real container ---
	resp, m := vmDoJSON(t, "PUT", vmURL, `{"location":"eastus","properties":{"miniblue":{"image":"alpine:3"}}}`)
	if resp.StatusCode != 201 {
		t.Fatalf("vm create failed: %d %v", resp.StatusCode, m)
	}
	mb := m["properties"].(map[string]interface{})["miniblue"].(map[string]interface{})
	if mb["powerState"] != "running" {
		t.Fatalf("expected running after create, got %v", mb["powerState"])
	}

	// --- deploy two services, each its own container on the VM network ---
	svcBase := base + "/web/services"
	resp, m = vmDoJSON(t, "PUT", svcBase+"/api"+vmAPIVersion,
		`{"properties":{"image":"busybox:latest","command":["sh","-c","echo api-started; httpd -f -p 18099"],"ports":[18099]}}`)
	if resp.StatusCode != 201 {
		t.Fatalf("deploy api failed: %d %v", resp.StatusCode, m)
	}
	resp, m = vmDoJSON(t, "PUT", svcBase+"/worker"+vmAPIVersion,
		`{"properties":{"image":"busybox:latest","command":["sh","-c","while true; do echo tick; sleep 1; done"]}}`)
	if resp.StatusCode != 201 {
		t.Fatalf("deploy worker failed: %d %v", resp.StatusCode, m)
	}

	resp, m = vmDoJSON(t, "GET", svcBase+vmAPIVersion, "")
	if resp.StatusCode != 200 || len(m["value"].([]interface{})) != 2 {
		t.Fatalf("expected 2 services, got %d %v", resp.StatusCode, m)
	}

	// Published port reachable from the host.
	deadline := time.Now().Add(15 * time.Second)
	var conn net.Conn
	var err error
	for time.Now().Before(deadline) {
		conn, err = net.DialTimeout("tcp", "localhost:18099", time.Second)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("service port 18099 never became reachable: %v", err)
	}

	// --- logs: per-service tail and combined labeled view (SC-003) ---
	time.Sleep(2500 * time.Millisecond) // let worker emit a few ticks
	resp, text := vmDoText(t, base+"/web/logs"+vmAPIVersion+"&service=worker&tail=3")
	if resp.StatusCode != 200 || !strings.Contains(text, "tick") {
		t.Fatalf("worker logs missing tick: %d %q", resp.StatusCode, text)
	}
	if strings.Contains(text, "api-started") {
		t.Fatalf("per-service logs leaked another service's output: %q", text)
	}
	resp, text = vmDoText(t, base+"/web/logs"+vmAPIVersion)
	if resp.StatusCode != 200 ||
		!strings.Contains(text, "==> service: api <==") ||
		!strings.Contains(text, "virtual machine created") {
		t.Fatalf("combined logs missing labels or provisioning events: %q", text)
	}

	// --- port conflict rejected without touching the running service ---
	resp, m = vmDoJSON(t, "PUT", svcBase+"/clash"+vmAPIVersion,
		`{"properties":{"image":"busybox:latest","ports":[18099]}}`)
	if resp.StatusCode != 409 {
		t.Fatalf("expected 409 PortConflict, got %d %v", resp.StatusCode, m)
	}
	if e, _ := m["error"].(map[string]interface{}); e == nil || e["code"] != "PortConflict" {
		t.Fatalf("expected PortConflict code, got %v", m)
	}

	// --- replace one service; the other keeps running ---
	resp, _ = vmDoJSON(t, "PUT", svcBase+"/worker"+vmAPIVersion,
		`{"properties":{"image":"busybox:latest","command":["sh","-c","while true; do echo tock; sleep 1; done"]}}`)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 on replace, got %d", resp.StatusCode)
	}
	resp, m = vmDoJSON(t, "GET", svcBase+"/api"+vmAPIVersion, "")
	if resp.StatusCode != 200 || m["properties"].(map[string]interface{})["status"] != "running" {
		t.Fatalf("api disturbed by worker replace: %d %v", resp.StatusCode, m)
	}

	// --- runCommand proves in-VM execution and identity env injection ---
	resp, m = vmDoJSON(t, "POST", base+"/web/runCommand"+vmAPIVersion,
		`{"commandId":"RunShellScript","script":["echo endpoint=$IDENTITY_ENDPOINT; echo header=${IDENTITY_HEADER:+set}"]}`)
	if resp.StatusCode != 200 {
		t.Fatalf("runCommand failed: %d %v", resp.StatusCode, m)
	}
	msg := m["value"].([]interface{})[0].(map[string]interface{})["message"].(string)
	if !strings.Contains(msg, "endpoint=http://host.docker.internal:") || !strings.Contains(msg, "header=set") {
		t.Fatalf("identity env vars not injected: %q", msg)
	}

	// --- fail-fast: unpullable image leaves no record (spec edge case) ---
	resp, m = vmDoJSON(t, "PUT", base+"/broken"+vmAPIVersion,
		`{"location":"eastus","properties":{"miniblue":{"image":"miniblue-definitely-not-an-image:nope"}}}`)
	if resp.StatusCode != 409 {
		t.Fatalf("expected 409 ContainerStartFailed, got %d %v", resp.StatusCode, m)
	}
	resp, _ = vmDoJSON(t, "GET", base+"/broken"+vmAPIVersion, "")
	if resp.StatusCode != 404 {
		t.Fatalf("phantom VM record left behind: %d", resp.StatusCode)
	}

	// --- power: stop blocks deploys, start recovers ---
	resp, m = vmDoJSON(t, "POST", base+"/web/powerOff"+vmAPIVersion, "")
	if resp.StatusCode != 200 {
		t.Fatalf("powerOff failed: %d %v", resp.StatusCode, m)
	}
	mb = m["properties"].(map[string]interface{})["miniblue"].(map[string]interface{})
	if mb["powerState"] != "stopped" {
		t.Fatalf("expected stopped, got %v", mb["powerState"])
	}
	resp, m = vmDoJSON(t, "PUT", svcBase+"/late"+vmAPIVersion, `{"properties":{"image":"busybox:latest"}}`)
	if resp.StatusCode != 409 {
		t.Fatalf("expected 409 VMNotRunning on stopped VM, got %d %v", resp.StatusCode, m)
	}
	resp, m = vmDoJSON(t, "POST", base+"/web/start"+vmAPIVersion, "")
	if resp.StatusCode != 200 {
		t.Fatalf("start failed: %d %v", resp.StatusCode, m)
	}

	// --- delete: containers and network removed, listings clean ---
	resp, _ = vmDoJSON(t, "DELETE", vmURL, "")
	if resp.StatusCode != 204 {
		t.Fatalf("vm delete failed: %d", resp.StatusCode)
	}
	resp, _ = vmDoJSON(t, "GET", vmURL, "")
	if resp.StatusCode != 404 {
		t.Fatalf("vm still present after delete: %d", resp.StatusCode)
	}
	if conn, err := net.DialTimeout("tcp", "localhost:18099", time.Second); err == nil {
		conn.Close()
		t.Fatal("service port still open after VM delete")
	}
	fmt.Println("vm docker lifecycle complete")
}
