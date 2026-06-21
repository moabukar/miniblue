package vm

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Store keys, container naming, response shaping
// ---------------------------------------------------------------------------

func vmKey(sub, rg, name string) string {
	return "vm:" + sub + ":" + rg + ":" + name
}

func svcKey(sub, rg, vmName, svc string) string {
	return "vmsvc:" + sub + ":" + rg + ":" + vmName + ":" + svc
}

func secretKey(secret string) string {
	return "vmidsecret:" + secret
}

func vmContainerName(rg, name string) string {
	return "miniblue-vm-" + rg + "-" + name
}

func vmNetworkName(rg, name string) string {
	return "miniblue-vmnet-" + rg + "-" + name
}

func svcContainerName(rg, vmName, svc string) string {
	return "miniblue-vmsvc-" + rg + "-" + vmName + "-" + svc
}

func vmARMID(sub, rg, name string) string {
	return "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Compute/virtualMachines/" + name
}

// identityEnv builds the managed-identity environment variables injected into
// the VM container and every service container. The endpoint points back at
// the miniblue host port through the host gateway alias; MSI_* are the legacy
// names older Azure SDKs probe for.
func identityEnv(secret string) []string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4566"
	}
	endpoint := "http://host.docker.internal:" + port + "/metadata/identity/oauth2/token"
	return []string{
		"IDENTITY_ENDPOINT=" + endpoint,
		"IDENTITY_HEADER=" + secret,
		"MSI_ENDPOINT=" + endpoint,
		"MSI_SECRET=" + secret,
	}
}

// appendProvisionEvent records a timestamped lifecycle event on the VM record
// (the `_provisionLog` internal field backing the VM-level log view).
func appendProvisionEvent(rec map[string]interface{}, format string, args ...interface{}) {
	events, _ := rec["_provisionLog"].([]interface{})
	line := time.Now().UTC().Format(time.RFC3339) + " " + fmt.Sprintf(format, args...)
	rec["_provisionLog"] = append(events, line)
}

func provisionEvents(rec map[string]interface{}) []string {
	events, _ := rec["_provisionLog"].([]interface{})
	out := make([]string, 0, len(events))
	for _, e := range events {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// publicView returns a copy of a stored record with internal fields (those
// prefixed by "_") stripped, suitable for API responses.
func publicView(rec map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(rec))
	for k, v := range rec {
		if strings.HasPrefix(k, "_") {
			continue
		}
		out[k] = v
	}
	return out
}

func getProps(rec map[string]interface{}) map[string]interface{} {
	props, _ := rec["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
		rec["properties"] = props
	}
	return props
}

func getMinibluProps(rec map[string]interface{}) map[string]interface{} {
	props := getProps(rec)
	mb, _ := props["miniblue"].(map[string]interface{})
	if mb == nil {
		mb = map[string]interface{}{}
		props["miniblue"] = mb
	}
	return mb
}

func powerState(rec map[string]interface{}) string {
	mb := getMinibluProps(rec)
	s, _ := mb["powerState"].(string)
	return s
}

// buildVMRecord assembles the stored VM record from the request input,
// echoing the accepted subset of the ARM shape and adding miniblue fields.
func buildVMRecord(sub, rg, name string, input map[string]interface{}, image string) map[string]interface{} {
	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}
	inProps, _ := input["properties"].(map[string]interface{})
	if inProps == nil {
		inProps = map[string]interface{}{}
	}

	props := map[string]interface{}{
		"provisioningState": "Succeeded",
		"miniblue": map[string]interface{}{
			"image":         image,
			"powerState":    "running",
			"containerName": vmContainerName(rg, name),
		},
	}
	for _, k := range []string{"hardwareProfile", "storageProfile", "osProfile"} {
		if v, ok := inProps[k]; ok {
			props[k] = v
		}
	}
	if props["hardwareProfile"] == nil {
		props["hardwareProfile"] = map[string]interface{}{"vmSize": "Standard_B1s"}
	}

	return map[string]interface{}{
		"id":         vmARMID(sub, rg, name),
		"name":       name,
		"type":       "Microsoft.Compute/virtualMachines",
		"location":   location,
		"properties": props,
	}
}
