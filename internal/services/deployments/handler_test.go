package deployments

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/store"
)

// fakeDispatcher records every call and returns 201.
type call struct{ method, path string }

func newFakeDispatcher() (Dispatcher, *[]call) {
	calls := &[]call{}
	d := func(method, path string, body []byte) (int, []byte) {
		*calls = append(*calls, call{method: method, path: path})
		return 201, []byte(`{"name":"ok"}`)
	}
	return d, calls
}

func do(t *testing.T, h http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSimpleDeploymentDispatchesEachResource(t *testing.T) {
	dispatch, calls := newFakeDispatcher()
	r := chi.NewRouter()
	NewHandler(store.New(), dispatch).Register(r)

	body := map[string]interface{}{
		"properties": map[string]interface{}{
			"mode": "Incremental",
			"template": map[string]interface{}{
				"resources": []interface{}{
					map[string]interface{}{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2023-01-01",
						"name":       "sa1",
						"location":   "eastus",
					},
					map[string]interface{}{
						"type":       "Microsoft.KeyVault/vaults",
						"apiVersion": "2023-07-01",
						"name":       "kv1",
						"location":   "eastus",
					},
				},
			},
		},
	}
	resp := do(t, r, "PUT",
		"/subscriptions/sub1/resourcegroups/rg1/providers/Microsoft.Resources/deployments/d1",
		body)
	if resp.Code != http.StatusCreated {
		t.Fatalf("PUT: %d %s", resp.Code, resp.Body.String())
	}
	var out map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &out)
	if state := out["properties"].(map[string]interface{})["provisioningState"]; state != "Succeeded" {
		t.Errorf("provisioningState: got %v", state)
	}
	if len(*calls) != 2 {
		t.Fatalf("dispatcher called %d times, want 2", len(*calls))
	}
	if (*calls)[0].path != "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Storage/storageAccounts/sa1?api-version=2023-01-01" {
		t.Errorf("first dispatch path wrong: %s", (*calls)[0].path)
	}
}

func TestParameterAndVariableSubstitution(t *testing.T) {
	dispatch, calls := newFakeDispatcher()
	r := chi.NewRouter()
	NewHandler(store.New(), dispatch).Register(r)

	body := map[string]interface{}{
		"properties": map[string]interface{}{
			"template": map[string]interface{}{
				"parameters": map[string]interface{}{
					"location":    map[string]interface{}{"type": "string", "defaultValue": "westus"},
					"storageName": map[string]interface{}{"type": "string"},
				},
				"variables": map[string]interface{}{
					"sku": "Standard_LRS",
				},
				"resources": []interface{}{
					map[string]interface{}{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2023-01-01",
						"name":       "[parameters('storageName')]",
						"location":   "[parameters('location')]",
						"sku":        map[string]interface{}{"name": "[variables('sku')]"},
					},
				},
			},
			"parameters": map[string]interface{}{
				"storageName": map[string]interface{}{"value": "actualname"},
			},
		},
	}
	if r := do(t, r, "PUT",
		"/subscriptions/s/resourcegroups/r/providers/Microsoft.Resources/deployments/d",
		body); r.Code != http.StatusCreated {
		t.Fatalf("%d %s", r.Code, r.Body.String())
	}
	if len(*calls) != 1 {
		t.Fatalf("calls: %+v", *calls)
	}
	want := "/subscriptions/s/resourceGroups/r/providers/Microsoft.Storage/storageAccounts/actualname?api-version=2023-01-01"
	if (*calls)[0].path != want {
		t.Errorf("path: got %s", (*calls)[0].path)
	}
}

func TestDispatcherFailureMarksDeploymentFailed(t *testing.T) {
	r := chi.NewRouter()
	failing := func(method, path string, body []byte) (int, []byte) {
		return 500, []byte(`{"error":"boom"}`)
	}
	NewHandler(store.New(), failing).Register(r)

	body := map[string]interface{}{
		"properties": map[string]interface{}{
			"template": map[string]interface{}{
				"resources": []interface{}{
					map[string]interface{}{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2023-01-01",
						"name":       "sa",
					},
				},
			},
		},
	}
	resp := do(t, r, "PUT",
		"/subscriptions/s/resourcegroups/r/providers/Microsoft.Resources/deployments/d",
		body)
	// Deployment record itself is 201; provisioningState reflects the failure.
	if resp.Code != http.StatusCreated {
		t.Fatalf("%d %s", resp.Code, resp.Body.String())
	}
	var out map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &out)
	props := out["properties"].(map[string]interface{})
	if props["provisioningState"] != "Failed" {
		t.Errorf("state: %v", props["provisioningState"])
	}
	if _, ok := props["error"]; !ok {
		t.Errorf("expected error field, got %+v", props)
	}
}

func TestGetDelete(t *testing.T) {
	dispatch, _ := newFakeDispatcher()
	r := chi.NewRouter()
	NewHandler(store.New(), dispatch).Register(r)

	body := map[string]interface{}{
		"properties": map[string]interface{}{
			"template": map[string]interface{}{
				"resources": []interface{}{},
			},
		},
	}
	do(t, r, "PUT",
		"/subscriptions/s/resourcegroups/r/providers/Microsoft.Resources/deployments/d",
		body)
	if rr := do(t, r, "GET",
		"/subscriptions/s/resourcegroups/r/providers/Microsoft.Resources/deployments/d", nil); rr.Code != http.StatusOK {
		t.Errorf("GET: %d", rr.Code)
	}
	if rr := do(t, r, "DELETE",
		"/subscriptions/s/resourcegroups/r/providers/Microsoft.Resources/deployments/d", nil); rr.Code != http.StatusNoContent {
		t.Errorf("DELETE: %d", rr.Code)
	}
	if rr := do(t, r, "GET",
		"/subscriptions/s/resourcegroups/r/providers/Microsoft.Resources/deployments/d", nil); rr.Code != http.StatusNotFound {
		t.Errorf("GET after delete: %d", rr.Code)
	}
}
