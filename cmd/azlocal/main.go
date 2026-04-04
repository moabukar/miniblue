package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

var baseURL = "http://localhost:4566"

func init() {
	if u := os.Getenv("LOCAL_AZURE_ENDPOINT"); u != "" {
		baseURL = u
	}
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(0)
	}

	// Parse the command
	cmd := args[0]
	switch cmd {
	case "group":
		handleGroup(args[1:])
	case "keyvault":
		handleKeyVault(args[1:])
	case "storage":
		handleStorage(args[1:])
	case "network":
		handleNetwork(args[1:])
	case "cosmosdb":
		handleCosmosDB(args[1:])
	case "servicebus":
		handleServiceBus(args[1:])
	case "appconfig":
		handleAppConfig(args[1:])
	case "functionapp":
		handleFunctions(args[1:])
	case "health":
		doGet("/health")
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`azlocal - CLI for local-azure (like awslocal for LocalStack)

Usage:
  azlocal <command> <subcommand> [flags]

Commands:
  group        Resource group operations
  keyvault     Key Vault secret operations
  storage      Blob storage operations
  network      Virtual network operations
  cosmosdb     Cosmos DB operations
  servicebus   Service Bus operations
  appconfig    App Configuration operations
  functionapp  Azure Functions operations
  health       Check local-azure health

Examples:
  azlocal group create --name myRG --location eastus
  azlocal group list --subscription sub1
  azlocal group show --name myRG --subscription sub1
  azlocal group delete --name myRG --subscription sub1

  azlocal keyvault secret set --vault myvault --name dbpass --value secret123
  azlocal keyvault secret show --vault myvault --name dbpass
  azlocal keyvault secret list --vault myvault

  azlocal storage container create --account myaccount --name mycontainer
  azlocal storage blob upload --account myaccount --container mycontainer --name hello.txt --data "Hello!"
  azlocal storage blob download --account myaccount --container mycontainer --name hello.txt
  azlocal storage blob list --account myaccount --container mycontainer

  azlocal health

Environment:
  LOCAL_AZURE_ENDPOINT  Override endpoint (default: http://localhost:4566)`)
}

// --- Helpers ---

func getFlag(args []string, name string) string {
	for i, a := range args {
		if a == "--"+name && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, "--"+name+"=") {
			return strings.TrimPrefix(a, "--"+name+"=")
		}
	}
	return ""
}

func requireFlag(args []string, name string) string {
	v := getFlag(args, name)
	if v == "" {
		fmt.Fprintf(os.Stderr, "Error: --%s is required\n", name)
		os.Exit(1)
	}
	return v
}

func sub(args []string) string {
	s := getFlag(args, "subscription")
	if s == "" {
		s = "default"
	}
	return s
}

func doGet(path string) {
	resp, err := http.Get(baseURL + path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	printResponse(resp)
}

func doPut(path string, body interface{}) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", baseURL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	printResponse(resp)
}

func doPutRaw(path string, contentType string, data []byte) {
	req, _ := http.NewRequest("PUT", baseURL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("OK")
	} else {
		printResponse(resp)
	}
}

func doDelete(path string) {
	req, _ := http.NewRequest("DELETE", baseURL+path, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("Deleted")
	} else {
		printResponse(resp)
	}
}

func doPost(path string, body interface{}) {
	data, _ := json.Marshal(body)
	resp, err := http.Post(baseURL+path, "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	printResponse(resp)
}

func printResponse(resp *http.Response) {
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		return
	}
	// Pretty print JSON
	var out bytes.Buffer
	if json.Indent(&out, body, "", "  ") == nil {
		fmt.Println(out.String())
	} else {
		fmt.Println(string(body))
	}
}

// --- Resource Groups ---

func handleGroup(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal group <create|list|show|delete> [flags]")
		return
	}
	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		s := sub(args)
		doPut("/subscriptions/"+s+"/resourcegroups/"+name, map[string]interface{}{
			"location": location,
			"tags":     map[string]string{},
		})
	case "list":
		s := sub(args)
		doGet("/subscriptions/" + s + "/resourcegroups")
	case "show":
		name := requireFlag(args, "name")
		s := sub(args)
		doGet("/subscriptions/" + s + "/resourcegroups/" + name)
	case "delete":
		name := requireFlag(args, "name")
		s := sub(args)
		doDelete("/subscriptions/" + s + "/resourcegroups/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: group %s\n", args[0])
	}
}

// --- Key Vault ---

func handleKeyVault(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: azlocal keyvault secret <set|show|list|delete> [flags]")
		return
	}
	if args[0] != "secret" {
		fmt.Fprintf(os.Stderr, "Unknown subcommand: keyvault %s\n", args[0])
		return
	}
	switch args[1] {
	case "set":
		vault := requireFlag(args, "vault")
		name := requireFlag(args, "name")
		value := requireFlag(args, "value")
		doPut("/keyvault/"+vault+"/secrets/"+name, map[string]string{"value": value})
	case "show":
		vault := requireFlag(args, "vault")
		name := requireFlag(args, "name")
		doGet("/keyvault/" + vault + "/secrets/" + name)
	case "list":
		vault := requireFlag(args, "vault")
		doGet("/keyvault/" + vault + "/secrets")
	case "delete":
		vault := requireFlag(args, "vault")
		name := requireFlag(args, "name")
		doDelete("/keyvault/" + vault + "/secrets/" + name)
	}
}

// --- Storage ---

func handleStorage(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: azlocal storage <container|blob> <subcommand> [flags]")
		return
	}
	switch args[0] {
	case "container":
		handleStorageContainer(args[1:])
	case "blob":
		handleStorageBlob(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: storage %s\n", args[0])
	}
}

func handleStorageContainer(args []string) {
	if len(args) == 0 {
		return
	}
	switch args[0] {
	case "create":
		account := requireFlag(args, "account")
		name := requireFlag(args, "name")
		doPutRaw("/blob/"+account+"/"+name, "application/json", nil)
	case "delete":
		account := requireFlag(args, "account")
		name := requireFlag(args, "name")
		doDelete("/blob/" + account + "/" + name)
	}
}

func handleStorageBlob(args []string) {
	if len(args) == 0 {
		return
	}
	switch args[0] {
	case "upload":
		account := requireFlag(args, "account")
		container := requireFlag(args, "container")
		name := requireFlag(args, "name")
		data := getFlag(args, "data")
		file := getFlag(args, "file")
		var content []byte
		if file != "" {
			var err error
			content, err = os.ReadFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
				os.Exit(1)
			}
		} else {
			content = []byte(data)
		}
		doPutRaw("/blob/"+account+"/"+container+"/"+name, "application/octet-stream", content)
	case "download":
		account := requireFlag(args, "account")
		container := requireFlag(args, "container")
		name := requireFlag(args, "name")
		doGet("/blob/" + account + "/" + container + "/" + name)
	case "list":
		account := requireFlag(args, "account")
		container := requireFlag(args, "container")
		doGet("/blob/" + account + "/" + container)
	case "delete":
		account := requireFlag(args, "account")
		container := requireFlag(args, "container")
		name := requireFlag(args, "name")
		doDelete("/blob/" + account + "/" + container + "/" + name)
	}
}

// --- Network ---

func handleNetwork(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: azlocal network vnet <create|show|list|delete> [flags]")
		return
	}
	if args[0] != "vnet" {
		fmt.Fprintf(os.Stderr, "Unknown subcommand: network %s\n", args[0])
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.Network/virtualNetworks"

	switch args[1] {
	case "create":
		name := requireFlag(args, "name")
		prefix := getFlag(args, "address-prefix")
		if prefix == "" {
			prefix = "10.0.0.0/16"
		}
		doPut(base+"/"+name, map[string]interface{}{
			"location": getFlag(args, "location"),
			"properties": map[string]interface{}{
				"addressSpace": map[string]interface{}{
					"addressPrefixes": []string{prefix},
				},
			},
		})
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "list":
		doGet(base)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	}
}

// --- Cosmos DB ---

func handleCosmosDB(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: azlocal cosmosdb doc <create|show|list|delete> [flags]")
		return
	}
	if args[0] != "doc" {
		fmt.Fprintf(os.Stderr, "Unknown subcommand: cosmosdb %s\n", args[0])
		return
	}
	account := requireFlag(args, "account")
	db := requireFlag(args, "database")
	coll := requireFlag(args, "collection")
	base := "/cosmosdb/" + account + "/dbs/" + db + "/colls/" + coll + "/docs"

	switch args[1] {
	case "create":
		id := requireFlag(args, "id")
		data := getFlag(args, "data")
		body := map[string]interface{}{"id": id}
		if data != "" {
			json.Unmarshal([]byte(data), &body)
			body["id"] = id
		}
		doPost(base, body)
	case "show":
		id := requireFlag(args, "id")
		doGet(base + "/" + id)
	case "list":
		doGet(base)
	case "delete":
		id := requireFlag(args, "id")
		doDelete(base + "/" + id)
	}
}

// --- Service Bus ---

func handleServiceBus(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: azlocal servicebus queue <create|send|receive|delete> [flags]")
		return
	}
	ns := requireFlag(args, "namespace")

	switch args[0] {
	case "queue":
		switch args[1] {
		case "create":
			name := requireFlag(args, "name")
			doPutRaw("/servicebus/"+ns+"/queues/"+name, "application/json", nil)
		case "send":
			name := requireFlag(args, "name")
			body := requireFlag(args, "body")
			doPost("/servicebus/"+ns+"/queues/"+name+"/messages", map[string]string{"body": body})
		case "receive":
			name := requireFlag(args, "name")
			doGet("/servicebus/" + ns + "/queues/" + name + "/messages/head")
		case "delete":
			name := requireFlag(args, "name")
			doDelete("/servicebus/" + ns + "/queues/" + name)
		}
	case "topic":
		switch args[1] {
		case "create":
			name := requireFlag(args, "name")
			doPutRaw("/servicebus/"+ns+"/topics/"+name, "application/json", nil)
		case "send":
			name := requireFlag(args, "name")
			body := requireFlag(args, "body")
			doPost("/servicebus/"+ns+"/topics/"+name+"/messages", map[string]string{"body": body})
		case "delete":
			name := requireFlag(args, "name")
			doDelete("/servicebus/" + ns + "/topics/" + name)
		}
	}
}

// --- App Config ---

func handleAppConfig(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: azlocal appconfig kv <set|show|list|delete> [flags]")
		return
	}
	if args[0] != "kv" {
		fmt.Fprintf(os.Stderr, "Unknown subcommand: appconfig %s\n", args[0])
		return
	}
	store := requireFlag(args, "store")
	base := "/appconfig/" + store + "/kv"

	switch args[1] {
	case "set":
		key := requireFlag(args, "key")
		value := requireFlag(args, "value")
		doPut(base+"/"+key, map[string]string{"value": value})
	case "show":
		key := requireFlag(args, "key")
		doGet(base + "/" + key)
	case "list":
		doGet(base)
	case "delete":
		key := requireFlag(args, "key")
		doDelete(base + "/" + key)
	}
}

// --- Functions ---

func handleFunctions(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal functionapp <create|show|list|delete> [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		doPut(base+"/"+name, map[string]interface{}{
			"location":   location,
			"properties": map[string]string{},
		})
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "list":
		doGet(base)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	}
}
