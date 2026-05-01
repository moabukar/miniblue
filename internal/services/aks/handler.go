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
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

// Azure AKS naming rule. Reject early so PUTs that would fail at real Azure
// also fail here.
var validClusterName = regexp.MustCompile(`^[a-zA-Z0-9][-_a-zA-Z0-9]{0,62}$`).MatchString

// Handler serves the Microsoft.ContainerService ARM endpoints.
type Handler struct {
	store    *store.Store
	backend  backend
	realMode bool
}

// NewHandler probes for Docker once at startup. The real backend activates
// only when both Docker is reachable and the user opted in via env var.
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

// Register mounts all AKS routes. Both "resourceGroups" and "resourcegroups"
// are mounted because Azure ARM is case-insensitive on that segment.
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

// Shutdown deletes every real backend container before miniblue exits.
// Called from the server's SIGTERM/SIGINT path; no-op in stub mode.
func (h *Handler) Shutdown(ctx context.Context) {
	if !h.realMode {
		return
	}
	clusters := h.store.ListByPrefix("aks:cluster:")
	if len(clusters) == 0 {
		return
	}
	log.Printf("[aks] tearing down %d k3s cluster(s) before exit", len(clusters))
	var wg sync.WaitGroup
	for _, item := range clusters {
		handle := backendHandleFromCluster(item)
		if handle == nil {
			continue
		}
		wg.Add(1)
		go func(hh *backendHandle) {
			defer wg.Done()
			if err := h.backend.Delete(hh); err != nil {
				log.Printf("[aks] shutdown delete %s: %v", hh.ContainerName, err)
			}
		}(handle)
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		log.Printf("[aks] shutdown timed out: %v", ctx.Err())
	}
}

// ListAdminCredential and ListUserCredential return a base64-encoded
// kubeconfig: real one from the k3s container in real mode, stub otherwise.
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
