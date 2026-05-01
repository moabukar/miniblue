// Package aks emulates the Azure Kubernetes Service (AKS) ARM management API.
//
// Two backends are supported:
//
//   - stub (default): cluster create/get/list/delete only updates miniblue's
//     in-memory store; listClusterAdminCredential returns a mock kubeconfig
//     pointing at a sentinel host. Sufficient for `terraform plan/apply`,
//     `az aks list`, and IaC iteration.
//
//   - real (k3s in Docker): when Docker is available and `AKS_BACKEND=k3s`
//     (or `MINIBLUE_AKS_REAL=1`), cluster create launches a `rancher/k3s`
//     container, exposes the API server on a dynamic localhost port, and
//     listClusterAdminCredential returns the real kubeconfig so `kubectl`
//     actually works.
//
// Both modes implement the same ARM contract; the backend choice is invisible
// to Terraform / Bicep / `az`.
package aks

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

// validClusterName matches Azure's AKS cluster name rule: 1 to 63 characters,
// starting with an alphanumeric, then alphanumerics, hyphens, or underscores.
// Validating up front prevents PUTs that would silently work in miniblue but
// fail when the same Terraform/Bicep is later applied to real Azure.
var validClusterName = regexp.MustCompile(`^[a-zA-Z0-9][-_a-zA-Z0-9]{0,62}$`).MatchString

// Handler serves the Microsoft.ContainerService ARM endpoints.
type Handler struct {
	store    *store.Store
	backend  backend
	realMode bool
}

// NewHandler returns a Handler. It probes for Docker once at startup; the
// real backend only activates when both Docker is reachable AND the user
// opted in via env var, to preserve miniblue's "fast & cheap by default"
// behavior.
func NewHandler(s *store.Store) *Handler {
	h := &Handler{store: s}

	wantReal := os.Getenv("AKS_BACKEND") == "k3s" || os.Getenv("MINIBLUE_AKS_REAL") == "1"
	if wantReal {
		if dockerAvailable() {
			b := newK3sBackend()
			b.reapOrphans(s)
			h.backend = b
			h.realMode = true
			log.Println("[aks] real backend enabled (rancher/k3s in Docker) – kubectl will work against created clusters")
		} else {
			log.Println("[aks] AKS_BACKEND=k3s requested but Docker is not available – falling back to stub")
		}
	}
	if h.backend == nil {
		h.backend = stubBackend{}
		log.Println("[aks] stub backend – clusters are ARM-only; set AKS_BACKEND=k3s with Docker running to enable real Kubernetes")
	}
	return h
}

// Register mounts all AKS routes on the given router.
//
// Azure ARM is case-insensitive on segments like "resourceGroups" but go-chi
// is case-sensitive, so both spellings are mounted.
func (h *Handler) Register(r chi.Router) {
	for _, rgSeg := range []string{"resourcegroups", "resourceGroups"} {
		r.Route("/subscriptions/{subscriptionId}/"+rgSeg+"/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters", func(r chi.Router) {
			r.Get("/", h.ListClustersInRG)
			r.Route("/{clusterName}", func(r chi.Router) {
				r.Put("/", h.CreateOrUpdateCluster)
				r.Get("/", h.GetCluster)
				r.Delete("/", h.DeleteCluster)
				r.Post("/listClusterAdminCredential", h.ListAdminCredential)
				r.Post("/listClusterUserCredential", h.ListUserCredential)
				r.Route("/agentPools", func(r chi.Router) {
					r.Get("/", h.ListAgentPools)
					r.Get("/{poolName}", h.GetAgentPool)
				})
			})
		})
	}
	// Subscription-scoped list (used by `az aks list`).
	r.Get("/subscriptions/{subscriptionId}/providers/Microsoft.ContainerService/managedClusters", h.ListClustersInSubscription)
}

// ---------------------------------------------------------------------------
// Store keys
// ---------------------------------------------------------------------------

func clusterKey(sub, rg, name string) string {
	return "aks:cluster:" + sub + ":" + rg + ":" + name
}

func clusterPrefixRG(sub, rg string) string {
	return "aks:cluster:" + sub + ":" + rg + ":"
}

func clusterPrefixSub(sub string) string {
	return "aks:cluster:" + sub + ":"
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func (h *Handler) CreateOrUpdateCluster(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "clusterName")

	if !validClusterName(name) {
		azerr.BadRequest(w, `Cluster name must match ^[a-zA-Z0-9][-_a-zA-Z0-9]{0,62}$ (Azure AKS naming rule).`)
		return
	}

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		azerr.BadRequest(w, "Could not parse request body.")
		return
	}

	key := clusterKey(sub, rg, name)
	_, exists := h.store.Get(key)

	cluster := buildClusterResponse(sub, rg, name, input)

	// If real backend, provision a k3s container and stash its handle on the
	// resource so listClusterAdminCredential can return the real kubeconfig.
	// Surface the error rather than silently falling back to stub: a user who
	// opted in to AKS_BACKEND=k3s should not get back a "Succeeded" cluster
	// whose kubeconfig points at miniblue-aks.invalid.
	if h.realMode {
		handle, err := h.backend.Create(sub, rg, name)
		if err != nil {
			log.Printf("[aks] real backend Create failed for %s/%s: %v", rg, name, err)
			azerr.WriteError(w, http.StatusInternalServerError, "AksBackendUnavailable",
				"AKS_BACKEND=k3s is set but cluster provisioning failed: "+err.Error())
			return
		}
		if handle != nil {
			cluster["_miniblue_backend"] = handle.serialize()
		}
	}

	h.store.Set(key, cluster)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(stripInternalFields(cluster))
}

func (h *Handler) GetCluster(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "clusterName")

	v, ok := h.store.Get(clusterKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.ContainerService/managedClusters", name)
		return
	}
	_ = json.NewEncoder(w).Encode(stripInternalFields(v))
}

func (h *Handler) DeleteCluster(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "clusterName")

	key := clusterKey(sub, rg, name)
	v, ok := h.store.Get(key)
	if !ok {
		azerr.NotFound(w, "Microsoft.ContainerService/managedClusters", name)
		return
	}

	if h.realMode {
		if handle := backendHandleFromCluster(v); handle != nil {
			if err := h.backend.Delete(handle); err != nil {
				log.Printf("[aks] real backend Delete failed for %s/%s: %v", rg, name, err)
			}
		}
	}

	h.store.Delete(key)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListClustersInRG(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix(clusterPrefixRG(sub, rg))
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": stripInternalFieldsList(items)})
}

func (h *Handler) ListClustersInSubscription(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	items := h.store.ListByPrefix(clusterPrefixSub(sub))
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": stripInternalFieldsList(items)})
}

// ListAdminCredential and ListUserCredential are POST actions that return a
// base64-encoded kubeconfig. In real mode we extract it from the running k3s
// container and rewrite the server URL to the host-mapped port; in stub mode
// we return a syntactically-valid kubeconfig pointing at a sentinel host.
func (h *Handler) ListAdminCredential(w http.ResponseWriter, r *http.Request) {
	h.writeKubeconfigResponse(w, r, "clusterAdmin")
}

func (h *Handler) ListUserCredential(w http.ResponseWriter, r *http.Request) {
	h.writeKubeconfigResponse(w, r, "clusterUser")
}

func (h *Handler) writeKubeconfigResponse(w http.ResponseWriter, r *http.Request, kubeconfigName string) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "clusterName")

	v, ok := h.store.Get(clusterKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.ContainerService/managedClusters", name)
		return
	}

	var kubeconfig []byte
	if h.realMode {
		handle := backendHandleFromCluster(v)
		if handle == nil {
			// Cluster exists in store but has no real backend (e.g. created in
			// stub mode then miniblue restarted with AKS_BACKEND=k3s, or a
			// failed Create). Surface rather than silently downgrade.
			azerr.WriteError(w, http.StatusInternalServerError, "AksBackendUnavailable",
				"AKS_BACKEND=k3s is set but this cluster has no real backend; recreate it")
			return
		}
		cfg, err := h.backend.Kubeconfig(handle, name)
		if err != nil {
			log.Printf("[aks] real backend Kubeconfig failed for %s/%s: %v", rg, name, err)
			azerr.WriteError(w, http.StatusInternalServerError, "AksBackendUnavailable",
				"k3s container is unreachable: "+err.Error())
			return
		}
		kubeconfig = cfg
	}
	if kubeconfig == nil {
		kubeconfig = stubKubeconfig(name)
	}

	resp := map[string]interface{}{
		"kubeconfigs": []map[string]interface{}{
			{
				"name":  kubeconfigName,
				"value": kubeconfig, // json.Marshal base64-encodes []byte automatically
			},
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) ListAgentPools(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "clusterName")

	v, ok := h.store.Get(clusterKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.ContainerService/managedClusters", name)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"value": agentPoolsFromCluster(v, sub, rg, name),
	})
}

func (h *Handler) GetAgentPool(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "clusterName")
	pool := chi.URLParam(r, "poolName")

	v, ok := h.store.Get(clusterKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.ContainerService/managedClusters", name)
		return
	}
	for _, p := range agentPoolsFromCluster(v, sub, rg, name) {
		pm, _ := p.(map[string]interface{})
		if pm != nil && pm["name"] == pool {
			_ = json.NewEncoder(w).Encode(pm)
			return
		}
	}
	azerr.NotFound(w, "Microsoft.ContainerService/managedClusters/agentPools", pool)
}
