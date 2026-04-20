package tests

import (
	"testing"
)

func TestContainerAppCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/containerApps"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "PUT", base+"/myapp"+av,
		`{"location":"eastus"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded")
	}
	if m["id"] != "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/containerApps/myapp" {
		t.Fatalf("unexpected id: %v", m["id"])
	}

	resp = doRequest(t, "PUT", base+"/myapp"+av, `{"location":"eastus"}`)
	resp.Body.Close()
	expectStatus(t, resp, 200)

	resp = doRequest(t, "GET", base+"/myapp"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
	m = decodeJSON(t, resp)
	if m["name"] != "myapp" {
		t.Fatalf("expected name=myapp")
	}

	doRequest(t, "DELETE", base+"/myapp"+av, "").Body.Close()
	resp = doRequest(t, "GET", base+"/myapp"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestContainerAppList(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/containerApps"
	av := "?api-version=2023-09-01"

	doRequest(t, "PUT", base+"/app1"+av, `{"location":"eastus"}`).Body.Close()
	doRequest(t, "PUT", base+"/app2"+av, `{"location":"eastus"}`).Body.Close()

	resp := doRequest(t, "GET", base+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	items := m["value"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestContainerAppStartStop(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/containerApps"
	av := "?api-version=2023-09-01"

	doRequest(t, "PUT", base+"/myapp"+av, `{"location":"eastus"}`).Body.Close()

	resp := doRequest(t, "POST", base+"/myapp/start"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	resp = doRequest(t, "POST", base+"/myapp/stop"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
}

func TestContainerAppRevisions(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/containerApps"
	av := "?api-version=2023-09-01"

	doRequest(t, "PUT", base+"/myapp"+av, `{"location":"eastus"}`).Body.Close()

	resp := doRequest(t, "GET", base+"/myapp/revisions"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	items := m["value"].([]interface{})
	if len(items) == 0 {
		t.Fatalf("expected at least 1 revision")
	}
}

func TestContainerAppGetAuthToken(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/containerApps"
	av := "?api-version=2023-09-01"

	doRequest(t, "PUT", base+"/myapp"+av, `{"location":"eastus"}`).Body.Close()

	resp := doRequest(t, "POST", base+"/myapp/getAuthToken"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["token"] == nil {
		t.Fatalf("expected token")
	}
}

func TestContainerAppAnalyzeCustomDomain(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/containerApps"
	av := "?api-version=2023-09-01"

	doRequest(t, "PUT", base+"/myapp"+av, `{"location":"eastus"}`).Body.Close()

	resp := doRequest(t, "POST", base+"/myapp/analyzeCustomDomain"+av, `{"hostname":"app.example.com"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["hostname"] != "app.example.com" {
		t.Fatalf("expected hostname=app.example.com")
	}
}

func TestContainerAppListBySubscription(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/providers/Microsoft.App/containerApps"
	av := "?api-version=2023-09-01"
	rgBase := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/containerApps"

	doRequest(t, "PUT", rgBase+"/app1"+av, `{"location":"eastus"}`).Body.Close()
	doRequest(t, "PUT", rgBase+"/app2"+av, `{"location":"eastus"}`).Body.Close()

	resp := doRequest(t, "GET", base+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	items := m["value"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestManagedEnvironmentCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/managedEnvironments"
	av := "?api-version=2023-09-01"

	resp := doRequest(t, "PUT", base+"/myenv"+av, `{"location":"eastus"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded")
	}
	if m["id"] != "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/managedEnvironments/myenv" {
		t.Fatalf("unexpected id: %v", m["id"])
	}

	resp = doRequest(t, "GET", base+"/myenv"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	doRequest(t, "DELETE", base+"/myenv"+av, "").Body.Close()
	resp = doRequest(t, "GET", base+"/myenv"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestContainerAppWithEnvironment(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	envBase := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/managedEnvironments"
	appBase := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/containerApps"
	av := "?api-version=2023-09-01"

	doRequest(t, "PUT", envBase+"/myenv"+av, `{"location":"eastus"}`).Body.Close()

	resp := doRequest(t, "PUT", appBase+"/myapp"+av, `{"location":"eastus","properties":{"environmentId":"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.App/managedEnvironments/myenv"}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	props := m["properties"].(map[string]interface{})
	envID, _ := props["managedEnvironmentId"].(string)
	if envID == "" {
		t.Fatalf("expected managedEnvironmentId in response")
	}
}
