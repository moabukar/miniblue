package vm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
)

// ---------------------------------------------------------------------------
// Service deployments (miniblue extension sub-resource)
//
// A VM hosts zero or more named services, each backed by its own container
// joined to the VM's Docker network. Deploying to an existing name replaces
// only that service.
// ---------------------------------------------------------------------------

var serviceNameRe = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)

type serviceInput struct {
	image   string
	command []string
	ports   []int
	env     []string
}

func parseServiceInput(input map[string]interface{}) (*serviceInput, error) {
	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		return nil, fmt.Errorf("the request body must contain a properties object")
	}
	si := &serviceInput{}
	si.image, _ = props["image"].(string)
	if si.image == "" {
		return nil, fmt.Errorf("properties.image is required")
	}
	if cmdArr, ok := props["command"].([]interface{}); ok {
		for _, c := range cmdArr {
			if s, ok := c.(string); ok {
				si.command = append(si.command, s)
			}
		}
	}
	if pp, ok := props["ports"].([]interface{}); ok {
		for _, p := range pp {
			if f, ok := p.(float64); ok {
				si.ports = append(si.ports, int(f))
			}
		}
	}
	if envs, ok := props["environmentVariables"].([]interface{}); ok {
		for _, e := range envs {
			ev, _ := e.(map[string]interface{})
			n, _ := ev["name"].(string)
			v, _ := ev["value"].(string)
			if n != "" {
				si.env = append(si.env, n+"="+v)
			}
		}
	}
	return si, nil
}

// refreshService reconciles a service record's status with its container:
// running, stopped (exit 0), or failed (non-zero exit, with the reason).
func (h *Handler) refreshService(rec map[string]interface{}) {
	if !h.dockerAvail {
		return
	}
	containerName, _ := rec["_containerName"].(string)
	if containerName == "" {
		return
	}
	props := getProps(rec)
	if dockerInspectRunning(containerName) {
		props["status"] = "running"
		delete(props, "failureReason")
		return
	}
	code, errMsg, ok := dockerInspectExit(containerName)
	switch {
	case !ok:
		props["status"] = "stopped"
	case code != 0 || errMsg != "":
		props["status"] = "failed"
		props["failureReason"] = fmt.Sprintf("container exited with code %d %s", code, errMsg)
	default:
		props["status"] = "stopped"
	}
}

func (h *Handler) CreateOrUpdateService(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vmName := chi.URLParam(r, "vmName")
	svcName := chi.URLParam(r, "serviceName")

	vmRec, ok := h.getVM(sub, rg, vmName)
	if !ok {
		azerr.NotFound(w, "Microsoft.Compute/virtualMachines", vmName)
		return
	}
	if !serviceNameRe.MatchString(svcName) {
		azerr.BadRequest(w, "Service name must match [a-z0-9-]{1,40}.")
		return
	}

	input, err := decodeBody(r)
	if err != nil {
		azerr.BadRequest(w, "Could not parse request body: "+err.Error())
		return
	}
	si, err := parseServiceInput(input)
	if err != nil {
		azerr.BadRequest(w, err.Error())
		return
	}

	if !h.dockerAvail {
		dockerUnavailable(w)
		return
	}

	h.refreshVM(vmRec)
	if powerState(vmRec) != "running" {
		azerr.WriteError(w, http.StatusConflict, "VMNotRunning",
			"The virtual machine '"+vmName+"' must be running to deploy a service. Start it with the start action first.")
		return
	}

	// Reject port conflicts with sibling services BEFORE touching anything,
	// so a bad re-deploy never disturbs what is already running.
	for _, sibling := range h.serviceRecords(sub, rg, vmName) {
		if n, _ := sibling["name"].(string); n == svcName {
			continue
		}
		sp, _ := sibling["properties"].(map[string]interface{})
		sPorts, _ := sp["ports"].([]interface{})
		for _, spv := range sPorts {
			f, _ := spv.(float64)
			for _, p := range si.ports {
				if int(f) == p {
					azerr.WriteError(w, http.StatusConflict, "PortConflict",
						fmt.Sprintf("Port %d is already exposed by service '%s' on this virtual machine.", p, sibling["name"]))
					return
				}
			}
		}
	}

	key := svcKey(sub, rg, vmName, svcName)
	containerName := svcContainerName(rg, vmName, svcName)
	_, replacing := h.store.Get(key)
	if replacing {
		dockerRemove(containerName)
	}

	secret, _ := vmRec["_identitySecret"].(string)
	_, err = dockerRun(runOpts{
		name:    containerName,
		image:   si.image,
		env:     append(append([]string{}, si.env...), identityEnv(secret)...),
		ports:   si.ports,
		network: vmNetworkName(rg, vmName),
		cmd:     si.command,
	})
	if err != nil {
		// The replaced container (if any) is already gone; remove its record
		// too rather than keep an entry whose backing container vanished.
		h.store.Delete(key)
		azerr.WriteError(w, http.StatusConflict, "ContainerStartFailed", err.Error())
		return
	}

	ports := make([]interface{}, 0, len(si.ports))
	endpoints := make([]interface{}, 0, len(si.ports))
	for _, p := range si.ports {
		ports = append(ports, float64(p))
		endpoints = append(endpoints, fmt.Sprintf("localhost:%d", p))
	}
	props := map[string]interface{}{
		"image":      si.image,
		"status":     "running",
		"deployedAt": time.Now().UTC().Format(time.RFC3339),
		"ports":      ports,
		"endpoints":  endpoints,
	}
	if len(si.command) > 0 {
		cmd := make([]interface{}, len(si.command))
		for i, c := range si.command {
			cmd[i] = c
		}
		props["command"] = cmd
	}
	if envs, ok := input["properties"].(map[string]interface{})["environmentVariables"]; ok {
		props["environmentVariables"] = envs
	}
	rec := map[string]interface{}{
		"id":             vmARMID(sub, rg, vmName) + "/services/" + svcName,
		"name":           svcName,
		"type":           "Microsoft.Compute/virtualMachines/services",
		"properties":     props,
		"_containerName": containerName,
	}
	h.store.Set(key, rec)

	appendProvisionEvent(vmRec, "service %s deployed (image %s)", svcName, si.image)
	h.store.Set(vmKey(sub, rg, vmName), vmRec)

	if replacing {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(publicView(rec))
}

func (h *Handler) GetService(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vmName := chi.URLParam(r, "vmName")
	svcName := chi.URLParam(r, "serviceName")

	v, ok := h.store.Get(svcKey(sub, rg, vmName, svcName))
	if !ok {
		azerr.NotFound(w, "Microsoft.Compute/virtualMachines/services", svcName)
		return
	}
	rec, _ := v.(map[string]interface{})
	h.refreshService(rec)
	h.store.Set(svcKey(sub, rg, vmName, svcName), rec)
	json.NewEncoder(w).Encode(publicView(rec))
}

func (h *Handler) ListServices(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vmName := chi.URLParam(r, "vmName")

	if _, ok := h.getVM(sub, rg, vmName); !ok {
		azerr.NotFound(w, "Microsoft.Compute/virtualMachines", vmName)
		return
	}
	value := []interface{}{}
	for _, rec := range h.serviceRecords(sub, rg, vmName) {
		h.refreshService(rec)
		value = append(value, publicView(rec))
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"value": value})
}

func (h *Handler) DeleteService(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vmName := chi.URLParam(r, "vmName")
	svcName := chi.URLParam(r, "serviceName")

	key := svcKey(sub, rg, vmName, svcName)
	v, ok := h.store.Get(key)
	if !ok {
		azerr.NotFound(w, "Microsoft.Compute/virtualMachines/services", svcName)
		return
	}
	if h.dockerAvail {
		rec, _ := v.(map[string]interface{})
		if cn, _ := rec["_containerName"].(string); cn != "" {
			dockerRemove(cn)
		}
	}
	h.store.Delete(key)

	if vmRec, ok := h.getVM(sub, rg, vmName); ok {
		appendProvisionEvent(vmRec, "service %s removed", svcName)
		h.store.Set(vmKey(sub, rg, vmName), vmRec)
	}
	w.WriteHeader(http.StatusNoContent)
}
