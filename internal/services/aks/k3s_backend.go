package aks

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// k3sImage is the upstream Rancher k3s image. Pinned to a known-good patch
// version (rancher/k3s does not publish floating tags like v1.30-k3s1).
const k3sImage = "rancher/k3s:v1.30.14-k3s1"

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
// rewrites it so that:
//
//   - the API server URL points at the host-mapped localhost port instead of
//     the in-container address, so external kubectl can reach it;
//   - the cluster, context, and user are renamed from k3s's "default" to the
//     AKS resource name (cluster + context) and clusterAdmin_miniblue_<name>
//     (user), matching what real `az aks get-credentials` produces and so
//     multiple cluster kubeconfigs do not collide when merged into ~/.kube/config.
//
// Done with a YAML round-trip rather than text replacement because k3s emits
// "name: default" three times (cluster, context, user) and string-level
// replacement cannot tell them apart.
func (b *k3sBackend) Kubeconfig(h *backendHandle, clusterName string) ([]byte, error) {
	if h == nil || h.ContainerName == "" {
		return nil, fmt.Errorf("nil backend handle")
	}
	raw, err := exec.Command("docker", "exec", h.ContainerName, "cat", "/etc/rancher/k3s/k3s.yaml").Output()
	if err != nil {
		return nil, fmt.Errorf("docker exec cat k3s.yaml: %w", err)
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse k3s kubeconfig: %w", err)
	}

	userName := "clusterAdmin_miniblue_" + clusterName
	hostServer := fmt.Sprintf("https://localhost:%d", h.HostPort)

	if clusters, ok := cfg["clusters"].([]interface{}); ok {
		for _, c := range clusters {
			cm, _ := c.(map[string]interface{})
			if cm == nil {
				continue
			}
			cm["name"] = clusterName
			if inner, ok := cm["cluster"].(map[string]interface{}); ok {
				if s, ok := inner["server"].(string); ok {
					inner["server"] = strings.NewReplacer(
						"https://127.0.0.1:6443", hostServer,
						"https://0.0.0.0:6443", hostServer,
					).Replace(s)
				}
			}
		}
	}
	if contexts, ok := cfg["contexts"].([]interface{}); ok {
		for _, c := range contexts {
			cm, _ := c.(map[string]interface{})
			if cm == nil {
				continue
			}
			cm["name"] = clusterName
			if inner, ok := cm["context"].(map[string]interface{}); ok {
				inner["cluster"] = clusterName
				inner["user"] = userName
			}
		}
	}
	if users, ok := cfg["users"].([]interface{}); ok {
		for _, u := range users {
			um, _ := u.(map[string]interface{})
			if um == nil {
				continue
			}
			um["name"] = userName
		}
	}
	cfg["current-context"] = clusterName

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("re-marshal kubeconfig: %w", err)
	}
	return out, nil
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
