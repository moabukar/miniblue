package aks

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/moabukar/miniblue/internal/store"
)

// k3sImage is the upstream Rancher k3s image. Pinned to a known-good patch
// version (rancher/k3s does not publish floating tags like v1.30-k3s1).
const k3sImage = "rancher/k3s:v1.30.14-k3s1"

// k3sBackend launches one rancher/k3s container per AKS cluster on its own
// localhost port.
type k3sBackend struct {
	mu sync.Mutex // serializes Create so freePort + docker run cannot race
}

func newK3sBackend() *k3sBackend {
	b := &k3sBackend{}
	go b.warmPull()
	return b
}

// warmPull fetches the k3s image in the background so the first cluster
// create is not blocked on the ~200MB pull. Best-effort; lazy pull on
// Create otherwise.
func (b *k3sBackend) warmPull() {
	log.Printf("[aks] warming up: docker pull %s (200MB on first run)", k3sImage)
	if out, err := exec.Command("docker", "pull", k3sImage).CombinedOutput(); err != nil {
		log.Printf("[aks] background image pull failed: %v – %s (first cluster create will retry)", err, strings.TrimSpace(string(out)))
		return
	}
	log.Printf("[aks] image %s ready", k3sImage)
}

// Hash suffix disambiguates clusters that share a name across RGs/subscriptions
// while keeping the readable cluster name visible in `docker ps`.
func (b *k3sBackend) containerName(sub, rg, clusterName string) string {
	h := sha256.Sum256([]byte(sub + ":" + rg + ":" + clusterName))
	return "miniblue-aks-" + sanitizeDockerName(clusterName) + "-" + hex.EncodeToString(h[:4])
}

// Docker container names match [a-zA-Z0-9][a-zA-Z0-9_.-]*; map anything
// else to '_'.
func sanitizeDockerName(s string) string {
	if s == "" {
		return "x"
	}
	var b strings.Builder
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
		case r == '-' || r == '_' || r == '.':
			if i == 0 {
				b.WriteRune('x') // first char must be alphanumeric
			}
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func (b *k3sBackend) Create(sub, rg, clusterName string) (*backendHandle, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	name := b.containerName(sub, rg, clusterName)

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

// Kubeconfig reads k3s.yaml from inside the container and rewrites the
// server URL to the host-mapped port and the cluster/context/user names to
// the AKS resource name so multiple cluster kubeconfigs do not collide when
// merged into ~/.kube/config. YAML round-trip (not string replace) because
// k3s emits "name: default" in three places that look identical to a regex.
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

// reapOrphans removes miniblue-aks-* containers not referenced by any cluster
// in the store. Empty store = stateless restart, all containers are orphans.
// Persistent store = only out-of-band containers (none should normally
// exist) are removed.
func (b *k3sBackend) reapOrphans(s *store.Store) {
	expected := map[string]struct{}{}
	for _, item := range s.ListByPrefix("aks:cluster:") {
		if h := backendHandleFromCluster(item); h != nil && h.ContainerName != "" {
			expected[h.ContainerName] = struct{}{}
		}
	}
	out, err := exec.Command("docker", "ps", "-a",
		"--filter", "name=^miniblue-aks-",
		"--format", "{{.Names}}").Output()
	if err != nil {
		log.Printf("[aks] startup orphan reap: docker ps failed: %v", err)
		return
	}
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if name == "" {
			continue
		}
		if _, ok := expected[name]; ok {
			continue
		}
		log.Printf("[aks] reaping orphan k3s container %s (no matching AKS resource in store)", name)
		_ = exec.Command("docker", "rm", "-f", name).Run()
	}
}

// Kernel-allocated free port. TOCTOU window between close and `docker run`
// is a few ms, mitigated by Create's mutex.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// Two-phase wait: API healthz, then a node Ready=True. Without phase 2,
// `kubectl apply` right after Create succeeds but pods sit Pending until
// the node registers.
func waitForK3s(containerName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		out, err := exec.Command("docker", "exec", containerName,
			"kubectl", "--kubeconfig=/etc/rancher/k3s/k3s.yaml", "get", "--raw=/healthz").CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) == "ok" {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if !time.Now().Before(deadline) {
		return fmt.Errorf("API server /healthz never returned ok within %s", timeout)
	}

	for time.Now().Before(deadline) {
		out, err := exec.Command("docker", "exec", containerName,
			"kubectl", "--kubeconfig=/etc/rancher/k3s/k3s.yaml", "get", "nodes",
			"-o", `jsonpath={.items[*].status.conditions[?(@.type=="Ready")].status}`).CombinedOutput()
		if err == nil && strings.Contains(string(out), "True") {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("no node reached Ready=True within %s", timeout)
}
