package aks

import (
	"log"
	"os/exec"
	"strings"

	"github.com/moabukar/miniblue/internal/store"
)

// CleanupClustersInRG removes any real backend containers (k3s) attached to
// AKS clusters in the given subscription/resource group. Safe to call when
// AKS is in stub mode (no handles will be present) or when no AKS clusters
// exist in the RG. Intended to be invoked by the resourcegroups handler
// during cascade delete, BEFORE it removes the AKS keys from the store.
//
// Errors from `docker rm` are logged but not returned: the cascade delete
// should always succeed at the ARM level even if container cleanup fails
// (the user can prune leftovers manually with `docker ps --filter
// name=miniblue-aks-`).
func CleanupClustersInRG(s *store.Store, sub, rg string) {
	prefix := clusterPrefixRG(sub, rg)
	for _, item := range s.ListByPrefix(prefix) {
		handle := backendHandleFromCluster(item)
		if handle == nil || handle.ContainerName == "" {
			continue
		}
		out, err := exec.Command("docker", "rm", "-f", handle.ContainerName).CombinedOutput()
		if err != nil {
			log.Printf("[aks] cascade cleanup: docker rm %s: %v – %s", handle.ContainerName, err, strings.TrimSpace(string(out)))
		}
	}
}
