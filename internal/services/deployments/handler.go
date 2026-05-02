// Package deployments emulates Microsoft.Resources/deployments so Bicep
// (`az deployment group create`) and ARM template applies work end to end.
//
// Phase 1 scope: PUT/GET/DELETE/List a deployment, walk template.resources in
// order, resolve [parameters('x')] and [variables('x')] references, then PUT
// each resource against the embedded chi router. No copy loops, no
// conditions, no template expression functions beyond parameters/variables,
// no nested templates, no outputs evaluation.
package deployments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

// Dispatcher serves an internal API call (used to apply each templated
// resource). The server provides one wrapping its own chi router.
type Dispatcher func(method, path string, body []byte) (status int, respBody []byte)

type Handler struct {
	store      *store.Store
	dispatch   Dispatcher
	apiVersion string
}

func NewHandler(s *store.Store, d Dispatcher) *Handler {
	return &Handler{store: s, dispatch: d, apiVersion: "2021-04-01"}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/Microsoft.Resources/deployments", func(r chi.Router) {
		r.Get("/", h.List)
		r.Route("/{deploymentName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdate)
			r.Get("/", h.Get)
			r.Delete("/", h.Delete)
		})
	})
	// Case-insensitive duplicate per Azure ARM convention.
	r.Get("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Resources/deployments", h.List)
	r.Put("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Resources/deployments/{deploymentName}", h.CreateOrUpdate)
	r.Get("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Resources/deployments/{deploymentName}", h.Get)
	r.Delete("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Resources/deployments/{deploymentName}", h.Delete)
}

func key(sub, rg, name string) string {
	return "deployment:" + sub + ":" + rg + ":" + name
}

func (h *Handler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "deploymentName")

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		azerr.BadRequest(w, "could not parse request body")
		return
	}

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		azerr.BadRequest(w, "properties is required")
		return
	}
	template, _ := props["template"].(map[string]interface{})
	if template == nil {
		azerr.BadRequest(w, "properties.template is required")
		return
	}

	paramValues := flattenParameters(props["parameters"])
	vars, _ := template["variables"].(map[string]interface{})
	tplParams, _ := template["parameters"].(map[string]interface{})
	mergedParams := mergeParameterDefaults(tplParams, paramValues)

	resources, _ := template["resources"].([]interface{})
	outputResources := make([]map[string]interface{}, 0, len(resources))
	state := "Succeeded"
	var failureMsg string

	for i, raw := range resources {
		res, _ := raw.(map[string]interface{})
		if res == nil {
			continue
		}
		evaluated := resolveExpressions(res, mergedParams, vars).(map[string]interface{})

		path, body, err := buildResourceRequest(sub, rg, evaluated)
		if err != nil {
			state = "Failed"
			failureMsg = fmt.Sprintf("resource[%d]: %v", i, err)
			break
		}
		status, respBody := h.dispatch("PUT", path, body)
		if status >= 300 {
			state = "Failed"
			failureMsg = fmt.Sprintf("resource[%d] %s: %d %s", i, path, status, strings.TrimSpace(string(respBody)))
			break
		}
		outputResources = append(outputResources, map[string]interface{}{
			"id": "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/" + evaluated["type"].(string) + "/" + evaluated["name"].(string),
		})
	}

	resp := map[string]interface{}{
		"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Resources/deployments/" + name,
		"name": name,
		"type": "Microsoft.Resources/deployments",
		"properties": map[string]interface{}{
			"provisioningState": state,
			"mode":              valueOrDefault(props["mode"], "Incremental"),
			"timestamp":         time.Now().UTC().Format(time.RFC3339),
			"outputResources":   outputResources,
			"outputs":           map[string]interface{}{},
			"parameters":        paramValues,
		},
	}
	if state == "Failed" {
		resp["properties"].(map[string]interface{})["error"] = map[string]interface{}{
			"code":    "DeploymentFailed",
			"message": failureMsg,
		}
	}

	_, exists := h.store.Get(key(sub, rg, name))
	h.store.Set(key(sub, rg, name), resp)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "deploymentName")
	v, ok := h.store.Get(key(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Resources/deployments", name)
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "deploymentName")
	if !h.store.Delete(key(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Resources/deployments", name)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("deployment:" + sub + ":" + rg + ":")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// buildResourceRequest converts a (already-evaluated) ARM resource object
// into the PUT path and JSON body to apply it against the internal router.
// Nested type segments (e.g. Microsoft.Storage/storageAccounts/blobServices)
// are split into provider plus instance segments alternating type/name.
func buildResourceRequest(sub, rg string, res map[string]interface{}) (string, []byte, error) {
	rType, _ := res["type"].(string)
	name, _ := res["name"].(string)
	if rType == "" || name == "" {
		return "", nil, fmt.Errorf("resource missing type or name")
	}
	apiVersion, _ := res["apiVersion"].(string)
	if apiVersion == "" {
		apiVersion = "2023-01-01"
	}

	parts := strings.SplitN(rType, "/", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid type %q (want Provider/resourceType[/sub/...])", rType)
	}
	path := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/" + parts[0] + "/" + parts[1] + "/" + name + "?api-version=" + apiVersion

	body, err := json.Marshal(res)
	if err != nil {
		return "", nil, err
	}
	return path, body, nil
}

func valueOrDefault(v interface{}, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}

// NewRouterDispatcher wraps a chi.Router as a Dispatcher by spinning a
// httptest.ResponseRecorder per call. Lets the deployments handler dispatch
// templated resources back into the same server without going over TCP.
func NewRouterDispatcher(router http.Handler) Dispatcher {
	return func(method, path string, body []byte) (int, []byte) {
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code, rec.Body.Bytes()
	}
}
