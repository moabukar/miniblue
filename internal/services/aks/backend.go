package aks

import (
	"log"
	"os"
	"os/exec"
)

// backend abstracts the per-cluster lifecycle. The stub backend is a no-op;
// the k3s backend launches and tears down rancher/k3s containers.
//
// Create takes (sub, rg, name) because two clusters can share a name across
// different resource groups (or subscriptions); the backend uses the triple
// to produce a unique container identifier so they do not collide.
type backend interface {
	Create(sub, rg, clusterName string) (*backendHandle, error)
	Delete(handle *backendHandle) error
	Kubeconfig(handle *backendHandle, clusterName string) ([]byte, error)
}

// backendHandle is a serializable identifier for a real cluster's storage
// container, so a restarted miniblue can still find/clean it up.
type backendHandle struct {
	ContainerName string `json:"containerName"`
	HostPort      int    `json:"hostPort"`
}

func (h *backendHandle) serialize() map[string]interface{} {
	return map[string]interface{}{
		"containerName": h.ContainerName,
		"hostPort":      h.HostPort,
	}
}

// backendHandleFromCluster recovers a handle stashed in the cluster response
// (under the _miniblue_backend key), or returns nil if absent / malformed.
func backendHandleFromCluster(v interface{}) *backendHandle {
	cm, _ := v.(map[string]interface{})
	if cm == nil {
		return nil
	}
	raw, ok := cm["_miniblue_backend"].(map[string]interface{})
	if !ok {
		return nil
	}
	name, _ := raw["containerName"].(string)
	if name == "" {
		return nil
	}
	port := 0
	switch p := raw["hostPort"].(type) {
	case int:
		port = p
	case float64:
		port = int(p)
	}
	return &backendHandle{ContainerName: name, HostPort: port}
}

// dockerAvailable mirrors the ACI probe so AKS picks the same backend on the
// same host.
func dockerAvailable() bool {
	if _, err := os.Stat("/var/run/docker.sock"); err != nil {
		return false
	}
	out, err := exec.Command("docker", "info").CombinedOutput()
	if err != nil {
		log.Printf("[aks] docker info failed: %v – %s", err, string(out))
		return false
	}
	return true
}

// stubBackend is the default. It records nothing real – the in-memory store
// is the source of truth.
type stubBackend struct{}

func (stubBackend) Create(string, string, string) (*backendHandle, error) { return nil, nil }
func (stubBackend) Delete(*backendHandle) error                           { return nil }
func (stubBackend) Kubeconfig(*backendHandle, string) ([]byte, error)     { return nil, nil }
