package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// azlocal vm / azlocal identity
// ---------------------------------------------------------------------------

func vmBase(s, rg string) string {
	return "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.Compute/virtualMachines"
}

func identityARMID(s, rg, name string) string {
	if strings.HasPrefix(name, "/") {
		return name
	}
	return "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/" + name
}

// getFlagAll returns every value of a repeatable flag (e.g. --env K=V --env X=Y).
func getFlagAll(args []string, name string) []string {
	var out []string
	for i, a := range args {
		if a == "--"+name && i+1 < len(args) {
			out = append(out, args[i+1])
		}
		if strings.HasPrefix(a, "--"+name+"=") {
			out = append(out, strings.TrimPrefix(a, "--"+name+"="))
		}
	}
	return out
}

// fetchJSONMap GETs a path and decodes the JSON body, returning the HTTP status.
func fetchJSONMap(path string) (map[string]interface{}, int) {
	resp, err := http.Get(baseURL + armPath(path))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var m map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&m)
	return m, resp.StatusCode
}

func vmFail(m map[string]interface{}, status int) {
	if e, ok := m["error"].(map[string]interface{}); ok {
		fmt.Fprintf(os.Stderr, "Error: %v: %v\n", e["code"], e["message"])
	} else {
		fmt.Fprintf(os.Stderr, "Error: unexpected status %d\n", status)
	}
	os.Exit(1)
}

func handleVM(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal vm <create|list|show|delete|start|stop|restart|deploy|services|service-delete|logs|ssh|run-command|identity-assign|identity-remove> [flags]")
		return
	}
	s := sub(args)

	switch args[0] {
	case "create":
		rg := requireFlag(args, "resource-group")
		name := requireFlag(args, "name")
		props := map[string]interface{}{}
		if image := getFlag(args, "image"); image != "" {
			props["miniblue"] = map[string]interface{}{"image": image}
		}
		if size := getFlag(args, "vm-size"); size != "" {
			props["hardwareProfile"] = map[string]interface{}{"vmSize": size}
		}
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		body := map[string]interface{}{
			"location":   location,
			"properties": props,
		}
		if ids := getFlagAll(args, "identity"); len(ids) > 0 {
			assignments := map[string]interface{}{}
			for _, id := range ids {
				assignments[identityARMID(s, rg, id)] = map[string]interface{}{}
			}
			body["identity"] = map[string]interface{}{
				"type":                   "UserAssigned",
				"userAssignedIdentities": assignments,
			}
		}
		doPut(vmBase(s, rg)+"/"+name, body)
	case "list":
		if rg := getFlag(args, "resource-group"); rg != "" {
			doGet(vmBase(s, rg))
		} else {
			doGet("/subscriptions/" + s + "/providers/Microsoft.Compute/virtualMachines")
		}
	case "show":
		rg := requireFlag(args, "resource-group")
		doGet(vmBase(s, rg) + "/" + requireFlag(args, "name"))
	case "delete":
		rg := requireFlag(args, "resource-group")
		doDelete(vmBase(s, rg) + "/" + requireFlag(args, "name"))
	case "start", "restart":
		rg := requireFlag(args, "resource-group")
		doPost(vmBase(s, rg)+"/"+requireFlag(args, "name")+"/"+args[0], map[string]interface{}{})
	case "stop":
		rg := requireFlag(args, "resource-group")
		doPost(vmBase(s, rg)+"/"+requireFlag(args, "name")+"/powerOff", map[string]interface{}{})
	case "deploy":
		rg := requireFlag(args, "resource-group")
		name := requireFlag(args, "name")
		svc := requireFlag(args, "service")
		image := requireFlag(args, "image")
		props := map[string]interface{}{"image": image}
		if cmd := getFlag(args, "command"); cmd != "" {
			props["command"] = []interface{}{"sh", "-c", cmd}
		}
		if ports := getFlag(args, "ports"); ports != "" {
			var pp []interface{}
			for _, p := range strings.Split(ports, ",") {
				n, err := strconv.Atoi(strings.TrimSpace(p))
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: invalid port %q\n", p)
					os.Exit(1)
				}
				pp = append(pp, n)
			}
			props["ports"] = pp
		}
		if envs := getFlagAll(args, "env"); len(envs) > 0 {
			var ee []interface{}
			for _, e := range envs {
				k, v, _ := strings.Cut(e, "=")
				ee = append(ee, map[string]interface{}{"name": k, "value": v})
			}
			props["environmentVariables"] = ee
		}
		doPut(vmBase(s, rg)+"/"+name+"/services/"+svc, map[string]interface{}{"properties": props})
	case "services":
		rg := requireFlag(args, "resource-group")
		doGet(vmBase(s, rg) + "/" + requireFlag(args, "name") + "/services")
	case "service-delete":
		rg := requireFlag(args, "resource-group")
		doDelete(vmBase(s, rg) + "/" + requireFlag(args, "name") + "/services/" + requireFlag(args, "service"))
	case "logs":
		rg := requireFlag(args, "resource-group")
		name := requireFlag(args, "name")
		vmLogs(s, rg, name, args)
	case "ssh":
		rg := requireFlag(args, "resource-group")
		name := requireFlag(args, "name")
		vmSSH(s, rg, name)
	case "run-command":
		rg := requireFlag(args, "resource-group")
		name := requireFlag(args, "name")
		command := requireFlag(args, "command")
		vmRunCommand(s, rg, name, command)
	case "identity-assign":
		rg := requireFlag(args, "resource-group")
		name := requireFlag(args, "name")
		id := requireFlag(args, "identity")
		vmIdentityUpdate(s, rg, name, identityARMID(s, rg, id), true)
	case "identity-remove":
		rg := requireFlag(args, "resource-group")
		name := requireFlag(args, "name")
		id := requireFlag(args, "identity")
		vmIdentityUpdate(s, rg, name, identityARMID(s, rg, id), false)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: vm %s\n", args[0])
		os.Exit(1)
	}
}

func vmLogs(s, rg, name string, args []string) {
	q := []string{}
	if svc := getFlag(args, "service"); svc != "" {
		q = append(q, "service="+svc)
	}
	if tail := getFlag(args, "tail"); tail != "" {
		q = append(q, "tail="+tail)
	}
	if hasFlag(args, "follow") {
		q = append(q, "follow=true")
	}
	path := vmBase(s, rg) + "/" + name + "/logs"
	if len(q) > 0 {
		path += "?" + strings.Join(q, "&")
	}
	resp, err := http.Get(baseURL + armPath(path))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", strings.TrimSpace(string(body)))
		os.Exit(1)
	}
	// Stream straight to stdout so --follow renders lines as they arrive.
	io.Copy(os.Stdout, resp.Body)
}

func vmSSH(s, rg, name string) {
	rec, status := fetchJSONMap(vmBase(s, rg) + "/" + name)
	if status != http.StatusOK {
		vmFail(rec, status)
	}
	props, _ := rec["properties"].(map[string]interface{})
	mb, _ := props["miniblue"].(map[string]interface{})
	state, _ := mb["powerState"].(string)
	containerName, _ := mb["containerName"].(string)
	if state != "running" {
		fmt.Fprintf(os.Stderr, "Error: VM '%s' is not running (state: %s). Start it with: azlocal vm start --resource-group %s --name %s\n", name, state, rg, name)
		os.Exit(1)
	}
	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: the docker CLI is required for ssh and was not found on PATH.")
		os.Exit(1)
	}
	for _, shell := range []string{"/bin/bash", "/bin/sh"} {
		cmd := exec.Command("docker", "exec", "-it", containerName, shell)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err == nil {
			return
		}
		// 126/127 = shell missing in the image; try the next one.
		if ee, ok := err.(*exec.ExitError); ok && (ee.ExitCode() == 126 || ee.ExitCode() == 127) {
			continue
		}
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "Error: no usable shell found in the VM image.")
	os.Exit(1)
}

func vmRunCommand(s, rg, name, command string) {
	body := map[string]interface{}{
		"commandId": "RunShellScript",
		"script":    []interface{}{command},
	}
	data, _ := json.Marshal(body)
	resp, err := http.Post(baseURL+armPath(vmBase(s, rg)+"/"+name+"/runCommand"), "application/json", strings.NewReader(string(data)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != http.StatusOK {
		vmFail(out, resp.StatusCode)
	}
	values, _ := out["value"].([]interface{})
	if len(values) == 0 {
		return
	}
	v0, _ := values[0].(map[string]interface{})
	msg, _ := v0["message"].(string)
	fmt.Print(msg)
	if !strings.HasSuffix(msg, "\n") {
		fmt.Println()
	}
	if code, _ := v0["code"].(string); strings.Contains(code, "failed") {
		os.Exit(1)
	}
}

func vmIdentityUpdate(s, rg, name, armID string, assign bool) {
	rec, status := fetchJSONMap(vmBase(s, rg) + "/" + name)
	if status != http.StatusOK {
		vmFail(rec, status)
	}
	block, _ := rec["identity"].(map[string]interface{})
	assignments, _ := block["userAssignedIdentities"].(map[string]interface{})
	if assignments == nil {
		assignments = map[string]interface{}{}
	}
	if assign {
		assignments[armID] = map[string]interface{}{}
	} else {
		found := false
		for k := range assignments {
			if strings.EqualFold(k, armID) {
				delete(assignments, k)
				found = true
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "Error: identity '%s' is not assigned to VM '%s'\n", armID, name)
			os.Exit(1)
		}
	}
	if len(assignments) == 0 {
		rec["identity"] = map[string]interface{}{"type": "None"}
	} else {
		rec["identity"] = map[string]interface{}{
			"type":                   "UserAssigned",
			"userAssignedIdentities": assignments,
		}
	}
	doPut(vmBase(s, rg)+"/"+name, rec)
}

func handleIdentity(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal identity <create|list|show|delete> [flags]")
		return
	}
	s := sub(args)
	rg := requireFlag(args, "resource-group")
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		doPut(base+"/"+name, map[string]interface{}{"location": location})
	case "list":
		doGet(base)
	case "show":
		doGet(base + "/" + requireFlag(args, "name"))
	case "delete":
		doDelete(base + "/" + requireFlag(args, "name"))
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: identity %s\n", args[0])
		os.Exit(1)
	}
}
