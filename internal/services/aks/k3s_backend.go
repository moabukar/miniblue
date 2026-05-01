package aks

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// k3sImage is the upstream Rancher k3s image. Pinned to a major.minor channel
// so a release of miniblue uses a known-good k3s while still picking up
// security patches automatically.
const k3sImage = "rancher/k3s:v1.30-k3s1"

// k3sBackend launches one rancher/k3s container per AKS cluster.
//
// Each cluster gets a unique localhost port (chosen by asking the kernel for a
// free one), so multiple clusters can coexist. The kubeconfig is extracted
// from the running container and the server URL rewritten so external
// `kubectl` connects through the host-mapped port.
type k3sBackend struct {
	mu sync.Mutex // serializes container creates so port reservation can't race
}

func newK3sBackend() *k3sBackend { return &k3sBackend{} }

func (b *k3sBackend) containerName(clusterName string) string {
	return "miniblue-aks-" + clusterName
}

func (b *k3sBackend) Create(clusterName string) (*backendHandle, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	name := b.containerName(clusterName)

	// Idempotent PUT: tear down any prior container with the same name.
	_ = exec.Command("docker", "rm", "-f", name).Run()

	port, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("allocate port: %w", err)
	}

	args := []string{
		"run", "-d",
		"--name", name,
		"--privileged",
		"-p", fmt.Sprintf("%d:6443", port),
		"-e", "K3S_KUBECONFIG_OUTPUT=/output/kubeconfig.yaml",
		"-e", "K3S_KUBECONFIG_MODE=666",
		k3sImage,
		"server",
		"--tls-san=127.0.0.1",
		"--tls-san=localhost",
		"--disable=traefik", // smaller, faster boot; users can install ingress themselves
	}
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker run rancher/k3s: %v – %s", err, strings.TrimSpace(string(out)))
	}

	if err := waitForK3s(name, 60*time.Second); err != nil {
		// Don't leave a half-started container lying around.
		_ = exec.Command("docker", "rm", "-f", name).Run()
		return nil, fmt.Errorf("k3s never became ready: %w", err)
	}

	log.Printf("[aks] k3s cluster %q ready on https://localhost:%d", clusterName, port)
	return &backendHandle{ContainerName: name, HostPort: port}, nil
}

func (b *k3sBackend) Delete(h *backendHandle) error {
	if h == nil || h.ContainerName == "" {
		return nil
	}
	out, err := exec.Command("docker", "rm", "-f", h.ContainerName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm %s: %v – %s", h.ContainerName, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Kubeconfig reads /etc/rancher/k3s/k3s.yaml from inside the container and
// rewrites the server URL from the in-container address to the host-mapped
// localhost port, so the returned kubeconfig works from outside Docker.
func (b *k3sBackend) Kubeconfig(h *backendHandle, clusterName string) ([]byte, error) {
	if h == nil || h.ContainerName == "" {
		return nil, fmt.Errorf("nil backend handle")
	}
	out, err := exec.Command("docker", "exec", h.ContainerName, "cat", "/etc/rancher/k3s/k3s.yaml").Output()
	if err != nil {
		return nil, fmt.Errorf("docker exec cat k3s.yaml: %w", err)
	}
	cfg := string(out)
	// k3s writes server: https://127.0.0.1:6443 by default.
	cfg = strings.ReplaceAll(cfg, "https://127.0.0.1:6443", fmt.Sprintf("https://localhost:%d", h.HostPort))
	cfg = strings.ReplaceAll(cfg, "https://0.0.0.0:6443", fmt.Sprintf("https://localhost:%d", h.HostPort))
	// Friendlier names so `kubectl config current-context` shows the AKS name.
	cfg = strings.ReplaceAll(cfg, "name: default", "name: "+clusterName)
	cfg = strings.ReplaceAll(cfg, "cluster: default", "cluster: "+clusterName)
	cfg = strings.ReplaceAll(cfg, "user: default", "user: clusterAdmin_miniblue_"+clusterName)
	cfg = strings.ReplaceAll(cfg, "current-context: default", "current-context: "+clusterName)
	return []byte(cfg), nil
}

// freePort asks the kernel for an available TCP port. There's an unavoidable
// TOCTOU window between closing the listener and Docker binding the port; in
// practice the window is a few milliseconds and miniblue serializes Create
// calls via the backend mutex.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitForK3s polls until the k3s container's API server is responding.
func waitForK3s(containerName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command("docker", "exec", containerName,
			"kubectl", "--kubeconfig=/etc/rancher/k3s/k3s.yaml", "get", "--raw=/healthz").CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) == "ok" {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout after %s", timeout)
}
