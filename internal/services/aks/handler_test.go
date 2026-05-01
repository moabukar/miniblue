package aks

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/store"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	t.Setenv("AKS_BACKEND", "")
	t.Setenv("MINIBLUE_AKS_REAL", "")
	r := chi.NewRouter()
	NewHandler(store.New()).Register(r)
	return r
}

func do(t *testing.T, h http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestCreateGetDelete(t *testing.T) {
	h := newTestServer(t)
	base := "/subscriptions/sub1/resourcegroups/rg1/providers/Microsoft.ContainerService/managedClusters/k1"

	resp := do(t, h, "PUT", base, map[string]interface{}{
		"location": "westeurope",
		"properties": map[string]interface{}{
			"dnsPrefix":         "myprefix",
			"kubernetesVersion": "1.29.0",
		},
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("PUT new cluster: want 201, got %d – %s", resp.Code, resp.Body.String())
	}

	var got map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["name"] != "k1" {
		t.Errorf("name: want k1, got %v", got["name"])
	}
	if got["location"] != "westeurope" {
		t.Errorf("location preserved: got %v", got["location"])
	}
	props := got["properties"].(map[string]interface{})
	if props["kubernetesVersion"] != "1.29.0" {
		t.Errorf("kubernetesVersion: got %v", props["kubernetesVersion"])
	}
	if props["dnsPrefix"] != "myprefix" {
		t.Errorf("dnsPrefix: got %v", props["dnsPrefix"])
	}
	if !strings.HasSuffix(props["fqdn"].(string), ".azmk8s.io") {
		t.Errorf("fqdn should end in .azmk8s.io, got %v", props["fqdn"])
	}
	if _, ok := got["_miniblue_backend"]; ok {
		t.Errorf("internal _miniblue_backend leaked into PUT response")
	}

	// PUT again – must be 200 (update), not 201 (create).
	if r := do(t, h, "PUT", base, map[string]interface{}{"location": "westeurope"}); r.Code != http.StatusOK {
		t.Fatalf("PUT update: want 200, got %d", r.Code)
	}

	if r := do(t, h, "GET", base, nil); r.Code != http.StatusOK {
		t.Fatalf("GET: want 200, got %d", r.Code)
	}

	if r := do(t, h, "DELETE", base, nil); r.Code != http.StatusNoContent {
		t.Fatalf("DELETE: want 204, got %d", r.Code)
	}
	if r := do(t, h, "GET", base, nil); r.Code != http.StatusNotFound {
		t.Fatalf("GET after delete: want 404, got %d", r.Code)
	}
}

func TestListsAtBothScopes(t *testing.T) {
	h := newTestServer(t)
	for _, c := range []struct{ rg, name string }{{"rg1", "a"}, {"rg1", "b"}, {"rg2", "c"}} {
		do(t, h, "PUT",
			"/subscriptions/sub1/resourcegroups/"+c.rg+"/providers/Microsoft.ContainerService/managedClusters/"+c.name,
			map[string]interface{}{"location": "eastus"})
	}

	cases := []struct {
		path string
		want int
	}{
		{"/subscriptions/sub1/resourcegroups/rg1/providers/Microsoft.ContainerService/managedClusters", 2},
		{"/subscriptions/sub1/resourcegroups/rg2/providers/Microsoft.ContainerService/managedClusters", 1},
		{"/subscriptions/sub1/providers/Microsoft.ContainerService/managedClusters", 3},
	}
	for _, tc := range cases {
		r := do(t, h, "GET", tc.path, nil)
		if r.Code != http.StatusOK {
			t.Fatalf("GET %s: %d %s", tc.path, r.Code, r.Body.String())
		}
		var body struct {
			Value []map[string]interface{} `json:"value"`
		}
		if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s: %v", tc.path, err)
		}
		if len(body.Value) != tc.want {
			t.Errorf("list %s: want %d, got %d", tc.path, tc.want, len(body.Value))
		}
	}
}

func TestKubeconfigStubIsValid(t *testing.T) {
	h := newTestServer(t)
	base := "/subscriptions/sub1/resourcegroups/rg1/providers/Microsoft.ContainerService/managedClusters/k1"
	do(t, h, "PUT", base, map[string]interface{}{"location": "eastus"})

	r := do(t, h, "POST", base+"/listClusterAdminCredential", nil)
	if r.Code != http.StatusOK {
		t.Fatalf("listClusterAdminCredential: %d %s", r.Code, r.Body.String())
	}

	var body struct {
		Kubeconfigs []struct {
			Name  string `json:"name"`
			Value []byte `json:"value"`
		} `json:"kubeconfigs"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Kubeconfigs) != 1 {
		t.Fatalf("want 1 kubeconfig, got %d", len(body.Kubeconfigs))
	}
	if body.Kubeconfigs[0].Name != "clusterAdmin" {
		t.Errorf("name: got %q", body.Kubeconfigs[0].Name)
	}
	// Decoded value should look like a kubeconfig YAML.
	cfg := string(body.Kubeconfigs[0].Value)
	for _, want := range []string{"apiVersion: v1", "kind: Config", "clusters:", "users:", "k1"} {
		if !strings.Contains(cfg, want) {
			t.Errorf("kubeconfig missing %q. Full content:\n%s", want, cfg)
		}
	}

	// JSON roundtrip should preserve bytes (sanity check the Go []byte/base64 pact).
	encoded := base64.StdEncoding.EncodeToString(body.Kubeconfigs[0].Value)
	if encoded == "" {
		t.Error("kubeconfig encodes to empty base64")
	}
}

func TestAgentPools(t *testing.T) {
	h := newTestServer(t)
	base := "/subscriptions/sub1/resourcegroups/rg1/providers/Microsoft.ContainerService/managedClusters/k1"

	do(t, h, "PUT", base, map[string]interface{}{
		"location": "eastus",
		"properties": map[string]interface{}{
			"agentPoolProfiles": []map[string]interface{}{
				{"name": "system", "count": 2, "vmSize": "Standard_D2s_v5"},
				{"name": "user", "count": 3},
			},
		},
	})

	r := do(t, h, "GET", base+"/agentPools", nil)
	if r.Code != http.StatusOK {
		t.Fatalf("list pools: %d %s", r.Code, r.Body.String())
	}
	var list struct {
		Value []map[string]interface{} `json:"value"`
	}
	json.Unmarshal(r.Body.Bytes(), &list)
	if len(list.Value) != 2 {
		t.Fatalf("want 2 pools, got %d", len(list.Value))
	}

	if r := do(t, h, "GET", base+"/agentPools/user", nil); r.Code != http.StatusOK {
		t.Fatalf("get user pool: %d", r.Code)
	}
	if r := do(t, h, "GET", base+"/agentPools/missing", nil); r.Code != http.StatusNotFound {
		t.Fatalf("get missing pool: want 404, got %d", r.Code)
	}
}

func TestNotFound(t *testing.T) {
	h := newTestServer(t)
	if r := do(t, h, "GET", "/subscriptions/s/resourcegroups/r/providers/Microsoft.ContainerService/managedClusters/missing", nil); r.Code != http.StatusNotFound {
		t.Errorf("GET missing: want 404, got %d", r.Code)
	}
}

// TestPutResponseStripsInternalFields verifies the PUT handler does not
// leak miniblue-internal fields like _miniblue_backend (which holds the
// docker container handle for the real backend) to API consumers.
func TestPutResponseStripsInternalFields(t *testing.T) {
	h := newTestServer(t)
	r := do(t, h, "PUT",
		"/subscriptions/sub1/resourcegroups/rg1/providers/Microsoft.ContainerService/managedClusters/k1",
		map[string]interface{}{"location": "eastus"})
	if r.Code != http.StatusCreated {
		t.Fatalf("PUT: %d %s", r.Code, r.Body.String())
	}
	if strings.Contains(r.Body.String(), "_miniblue_backend") {
		t.Errorf("PUT response leaked internal field _miniblue_backend: %s", r.Body.String())
	}
	if strings.Contains(r.Body.String(), "_miniblue") {
		t.Errorf("PUT response leaked internal field starting with _miniblue: %s", r.Body.String())
	}
}

// TestContainerNameIsUniqueAcrossRGs is a regression for Bug H: two clusters
// with the same name in different resource groups must produce distinct
// docker container names so they do not clobber each other's backend.
func TestContainerNameIsUniqueAcrossRGs(t *testing.T) {
	b := newK3sBackend()
	a := b.containerName("sub1", "rgA", "k1")
	c := b.containerName("sub1", "rgB", "k1")
	if a == c {
		t.Fatalf("container names collide across RGs: %q == %q", a, c)
	}
	if !strings.HasPrefix(a, "miniblue-aks-k1-") || !strings.HasPrefix(c, "miniblue-aks-k1-") {
		t.Errorf("expected miniblue-aks-k1-<hash> form, got %q and %q", a, c)
	}
	// Same triple should be deterministic.
	if b.containerName("sub1", "rgA", "k1") != a {
		t.Errorf("containerName not deterministic")
	}
	// Sub also disambiguates.
	d := b.containerName("sub2", "rgA", "k1")
	if a == d {
		t.Fatalf("container names collide across subs: %q == %q", a, d)
	}
}

// TestSanitizeDockerName covers the helper that maps cluster names to a
// docker-safe form so a name like "My.Cluster_01" still produces a valid
// container name.
func TestSanitizeDockerName(t *testing.T) {
	cases := map[string]string{
		"k1":            "k1",
		"My-Cluster":    "my-cluster",
		"FOO_bar.baz":   "foo_bar.baz",
		"":              "x",
		"a b c":         "a_b_c",
		"-leading-dash": "x-leading-dash", // first char must be alphanumeric
		"ünîcødé":       "_n_c_d_",        // 7 runes, exotic chars get _
	}
	for in, want := range cases {
		if got := sanitizeDockerName(in); got != want {
			t.Errorf("sanitizeDockerName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCleanupClustersInRGIsSafeWithoutBackends verifies the public helper
// used by resourcegroups during cascade delete is a no-op for stub-mode
// clusters and does not panic on missing/empty/malformed handles.
func TestCleanupClustersInRGIsSafeWithoutBackends(t *testing.T) {
	t.Setenv("AKS_BACKEND", "")
	t.Setenv("MINIBLUE_AKS_REAL", "")
	s := store.New()
	r := chi.NewRouter()
	NewHandler(s).Register(r)

	for _, n := range []string{"a", "b", "c"} {
		req := httptest.NewRequest("PUT",
			"/subscriptions/sub1/resourcegroups/rg1/providers/Microsoft.ContainerService/managedClusters/"+n,
			strings.NewReader(`{"location":"eastus"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	// No backend handles set in stub mode; this should be a no-op (and not
	// shell out to docker for anything).
	CleanupClustersInRG(s, "sub1", "rg1")

	// The clusters themselves are still in the store; the helper only cleans
	// real backend containers, leaving the store delete to the caller.
	if got := len(s.ListByPrefix("aks:cluster:sub1:rg1:")); got != 3 {
		t.Errorf("CleanupClustersInRG should not delete store keys, want 3 left, got %d", got)
	}
}
