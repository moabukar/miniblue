package aci

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

// Handler serves Azure Container Instances (ACI) endpoints.
type Handler struct {
	store       *store.Store
	dockerAvail bool
}

// NewHandler creates a new ACI handler. It probes for Docker availability once
// at startup so every subsequent request knows whether to use real containers.
func NewHandler(s *store.Store) *Handler {
	h := &Handler{store: s}
	h.dockerAvail = h.checkDocker()
	if h.dockerAvail {
		log.Println("[aci] Docker is available – container groups will use real containers")
	} else {
		log.Println("[aci] Docker is NOT available – container groups will be stub-only")
	}
	return h
}

// Register mounts all ACI ARM routes on the given router.
func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerInstance/containerGroups", func(r chi.Router) {
		r.Get("/", h.ListContainerGroups)
		r.Route("/{containerGroupName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateContainerGroup)
			r.Get("/", h.GetContainerGroup)
			r.Delete("/", h.DeleteContainerGroup)
		})
	})
}

// ---------------------------------------------------------------------------
// Docker helpers
// ---------------------------------------------------------------------------

// checkDocker returns true when the Docker CLI is reachable and the daemon is
// responsive. It looks for the socket first, then runs `docker info`.
func (h *Handler) checkDocker() bool {
	// Quick pre-check: on Linux/macOS the socket should exist.
	if _, err := os.Stat("/var/run/docker.sock"); err != nil {
		return false
	}
	out, err := exec.Command("docker", "info").CombinedOutput()
	if err != nil {
		log.Printf("[aci] docker info failed: %v – %s", err, string(out))
		return false
	}
	return true
}

// containerName returns the Docker container name for a given group.
func containerName(groupName string) string {
	return "miniblue-" + groupName
}

// dockerRun creates and starts a real container via `docker run`.
func (h *Handler) dockerRun(groupName string, spec map[string]interface{}) (string, error) {
	name := containerName(groupName)

	// Extract container spec from properties.containers[0].properties
	image := "nginx:latest"
	var envVars []string
	var ports []string
	var cmd []string

	containers, _ := spec["containers"].([]interface{})
	if len(containers) > 0 {
		c0, _ := containers[0].(map[string]interface{})
		props, _ := c0["properties"].(map[string]interface{})
		if props != nil {
			if img, ok := props["image"].(string); ok && img != "" {
				image = img
			}
			if envs, ok := props["environmentVariables"].([]interface{}); ok {
				for _, e := range envs {
					ev, _ := e.(map[string]interface{})
					n, _ := ev["name"].(string)
					v, _ := ev["value"].(string)
					if n != "" {
						envVars = append(envVars, n+"="+v)
					}
				}
			}
			if pp, ok := props["ports"].([]interface{}); ok {
				for _, p := range pp {
					pm, _ := p.(map[string]interface{})
					if port, ok := pm["port"].(float64); ok {
						ports = append(ports, fmt.Sprintf("%d:%d", int(port), int(port)))
					}
				}
			}
			if cmdArr, ok := props["command"].([]interface{}); ok {
				for _, c := range cmdArr {
					if s, ok := c.(string); ok {
						cmd = append(cmd, s)
					}
				}
			}
		}
	}

	args := []string{"run", "-d", "--name", name}
	for _, e := range envVars {
		args = append(args, "-e", e)
	}
	for _, p := range ports {
		args = append(args, "-p", p)
	}
	args = append(args, image)
	args = append(args, cmd...)

	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker run failed: %v – %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// dockerInspectState returns "Running" or "Terminated" by inspecting a container.
func (h *Handler) dockerInspectState(groupName string) string {
	name := containerName(groupName)
	out, err := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", name).CombinedOutput()
	if err != nil {
		return "Terminated"
	}
	if strings.TrimSpace(string(out)) == "true" {
		return "Running"
	}
	return "Terminated"
}

// dockerRemove stops and removes the container.
func (h *Handler) dockerRemove(groupName string) {
	name := containerName(groupName)
	_ = exec.Command("docker", "stop", name).Run()
	_ = exec.Command("docker", "rm", name).Run()
}

// ---------------------------------------------------------------------------
// Store key
// ---------------------------------------------------------------------------

func (h *Handler) groupKey(sub, rg, name string) string {
	return "aci:containergroup:" + sub + ":" + rg + ":" + name
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func (h *Handler) CreateOrUpdateContainerGroup(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "containerGroupName")

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		azerr.BadRequest(w, "Could not parse request body.")
		return
	}

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	// If Docker is available, create a real container.
	var containerID string
	state := "Running"
	if h.dockerAvail {
		// Remove any previous container with same name (idempotent PUT).
		h.dockerRemove(name)

		var err error
		containerID, err = h.dockerRun(name, props)
		if err != nil {
			log.Printf("[aci] docker run error for %s: %v", name, err)
			// Fall back to stub behaviour so the API still works.
			state = "Waiting"
		} else {
			state = "Running"
		}
	}

	resp := h.buildResponse(sub, rg, name, input, state, containerID)
	key := h.groupKey(sub, rg, name)
	_, exists := h.store.Get(key)
	h.store.Set(key, resp)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetContainerGroup(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "containerGroupName")

	v, ok := h.store.Get(h.groupKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.ContainerInstance/containerGroups", name)
		return
	}

	resp, _ := v.(map[string]interface{})

	// If Docker is available, refresh the state from the real container.
	if h.dockerAvail && resp != nil {
		state := h.dockerInspectState(name)
		h.patchState(resp, state)
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) DeleteContainerGroup(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "containerGroupName")

	if !h.store.Delete(h.groupKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.ContainerInstance/containerGroups", name)
		return
	}

	if h.dockerAvail {
		h.dockerRemove(name)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListContainerGroups(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("aci:containergroup:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// ---------------------------------------------------------------------------
// Response builders
// ---------------------------------------------------------------------------

func (h *Handler) buildResponse(sub, rg, name string, input map[string]interface{}, state, containerID string) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.ContainerInstance/containerGroups/" + name

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	osType, _ := props["osType"].(string)
	if osType == "" {
		osType = "Linux"
	}

	// Build containers array for the response.
	respContainers := []interface{}{}
	if containers, ok := props["containers"].([]interface{}); ok {
		for _, c := range containers {
			cm, _ := c.(map[string]interface{})
			if cm == nil {
				continue
			}
			cName, _ := cm["name"].(string)
			cProps, _ := cm["properties"].(map[string]interface{})
			if cProps == nil {
				cProps = map[string]interface{}{}
			}
			image, _ := cProps["image"].(string)
			cPorts, _ := cProps["ports"].([]interface{})

			rc := map[string]interface{}{
				"name": cName,
				"properties": map[string]interface{}{
					"image": image,
					"ports": cPorts,
					"instanceView": map[string]interface{}{
						"currentState": map[string]interface{}{
							"state":     state,
							"startTime": time.Now().UTC().Format(time.RFC3339),
						},
					},
				},
			}
			respContainers = append(respContainers, rc)
		}
	}

	// Build ipAddress section.
	ipAddr := map[string]interface{}{
		"ip":   "127.0.0.1",
		"type": "Public",
	}
	if inputIP, ok := props["ipAddress"].(map[string]interface{}); ok {
		if t, ok := inputIP["type"].(string); ok {
			ipAddr["type"] = t
		}
		if p, ok := inputIP["ports"].([]interface{}); ok {
			ipAddr["ports"] = p
		}
	}
	if _, ok := ipAddr["ports"]; !ok {
		ipAddr["ports"] = []interface{}{}
	}

	resp := map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.ContainerInstance/containerGroups",
		"location": location,
		"properties": map[string]interface{}{
			"provisioningState": "Succeeded",
			"containers":        respContainers,
			"osType":            osType,
			"instanceView": map[string]interface{}{
				"state": state,
			},
			"ipAddress": ipAddr,
		},
	}

	if containerID != "" {
		resp["_containerID"] = containerID
	}

	return resp
}

// patchState updates the state fields inside an existing response map.
func (h *Handler) patchState(resp map[string]interface{}, state string) {
	props, _ := resp["properties"].(map[string]interface{})
	if props == nil {
		return
	}
	if iv, ok := props["instanceView"].(map[string]interface{}); ok {
		iv["state"] = state
	}
	if containers, ok := props["containers"].([]interface{}); ok {
		for _, c := range containers {
			cm, _ := c.(map[string]interface{})
			if cm == nil {
				continue
			}
			cp, _ := cm["properties"].(map[string]interface{})
			if cp == nil {
				continue
			}
			if iv, ok := cp["instanceView"].(map[string]interface{}); ok {
				if cs, ok := iv["currentState"].(map[string]interface{}); ok {
					cs["state"] = state
				}
			}
		}
	}
}
