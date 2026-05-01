package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
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
	case "dns":
		handleDNS(args[1:])
	case "eventgrid":
		handleEventGrid(args[1:])
	case "acr":
		handleACR(args[1:])
	case "postgres":
		handlePostgres(args[1:])
	case "redis":
		handleRedis(args[1:])
	case "sql":
		handleSQL(args[1:])
	case "mysql":
		handleMySQL(args[1:])
	case "aci":
		handleACI(args[1:])
	case "aks":
		handleAKS(args[1:])
	case "containerapp":
		handleContainerApp(args[1:])
	case "table":
		handleTable(args[1:])
	case "queue":
		handleQueue(args[1:])
	case "reset":
		doPost("/_miniblue/reset", nil)
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
	fmt.Println(`azlocal - CLI for miniblue (like awslocal for LocalStack)

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
  dns          DNS zone and record operations
  eventgrid    Event Grid topic operations
  acr          Azure Container Registry operations
  postgres     Azure Database for PostgreSQL operations
  redis        Azure Cache for Redis operations
  sql          Azure SQL Database operations
  mysql        Azure Database for MySQL operations
  aci          Azure Container Instances operations
  aks          Azure Kubernetes Service operations
  containerapp Azure Container Apps operations
  table        Azure Table Storage operations
  queue        Azure Queue Storage operations
  reset        Reset all miniblue state
  health       Check miniblue health

Examples:
  azlocal group create --name myRG --location eastus
  azlocal group list --subscription sub1
  azlocal group show --name myRG --subscription sub1
  azlocal group delete --name myRG --subscription sub1

  azlocal keyvault secret set --vault myvault --name dbpass --value secret123
  azlocal keyvault secret show --vault myvault --name dbpass
  azlocal keyvault secret list --vault myvault

  azlocal storage account create --resource-group myRG --name myaccount
  azlocal storage account list --resource-group myRG
  azlocal storage account show --resource-group myRG --name myaccount
  azlocal storage account list-keys --resource-group myRG --name myaccount
  azlocal storage account delete --resource-group myRG --name myaccount

  azlocal storage container create --account myaccount --name mycontainer
  azlocal storage blob upload --account myaccount --container mycontainer --name hello.txt --data "Hello!"
  azlocal storage blob download --account myaccount --container mycontainer --name hello.txt
  azlocal storage blob list --account myaccount --container mycontainer

  azlocal dns zone create --resource-group myRG --name example.com
  azlocal dns record create --resource-group myRG --zone example.com --type A --name www --data '{"properties":{"TTL":300,"ARecords":[{"ipv4Address":"1.2.3.4"}]}}'

  azlocal eventgrid topic create --resource-group myRG --name mytopic --location eastus
  azlocal eventgrid topic list --resource-group myRG

  azlocal acr create --resource-group myRG --name myregistry --location eastus
  azlocal acr list --resource-group myRG

  azlocal postgres server create --resource-group myRG --name mypg
  azlocal postgres database create --resource-group myRG --server mypg --name mydb

  azlocal redis create --resource-group myRG --name myredis
  azlocal redis list-keys --resource-group myRG --name myredis

  azlocal sql server create --resource-group myRG --name mysqlsrv --location eastus
  azlocal sql database create --resource-group myRG --server mysqlsrv --name mydb

  azlocal mysql server create --resource-group myRG --name mymysql --location eastus
  azlocal mysql database create --resource-group myRG --server mymysql --name mydb

  azlocal aci create --resource-group myRG --name mygroup --image nginx --location eastus
  azlocal aci list --resource-group myRG

  azlocal aks create --resource-group myRG --name mycluster --node-count 1
  azlocal aks list --resource-group myRG
  azlocal aks show --resource-group myRG --name mycluster
  azlocal aks get-credentials --resource-group myRG --name mycluster --file -
  azlocal aks delete --resource-group myRG --name mycluster

  azlocal containerapp env create --resource-group myRG --name myenv --location eastus
  azlocal containerapp env list --resource-group myRG
  azlocal containerapp env show --resource-group myRG --name myenv

  azlocal containerapp create --resource-group myRG --name myapp --image nginx --environment myenv
  azlocal containerapp list --resource-group myRG
  azlocal containerapp show --resource-group myRG --name myapp
  azlocal containerapp delete --resource-group myRG --name myapp

  azlocal table create --account myaccount --name mytable
  azlocal table entity put --account myaccount --table mytable --partition-key pk1 --row-key rk1 --data '{"foo":"bar"}'
  azlocal table entity get --account myaccount --table mytable --partition-key pk1 --row-key rk1

  azlocal queue create --account myaccount --name myqueue
  azlocal queue message send --account myaccount --queue myqueue --body "hello"
  azlocal queue message receive --account myaccount --queue myqueue
  azlocal queue message clear --account myaccount --queue myqueue

  azlocal reset
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
		s = "00000000-0000-0000-0000-000000000000"
	}
	return s
}

// armPath appends api-version for ARM endpoints
func armPath(path string) string {
	if strings.Contains(path, "/subscriptions/") {
		if strings.Contains(path, "?") {
			return path + "&api-version=2023-01-01"
		}
		return path + "?api-version=2023-01-01"
	}
	return path
}

func doGet(path string) {
	resp, err := http.Get(baseURL + armPath(path))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	printResponse(resp)
}

func doPut(path string, body interface{}) {
	data, err := json.Marshal(body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to encode request body: %v\n", err)
		os.Exit(1)
	}
	req, _ := http.NewRequest("PUT", baseURL+armPath(path), bytes.NewReader(data))
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
	req, _ := http.NewRequest("PUT", baseURL+armPath(path), bytes.NewReader(data))
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
	req, _ := http.NewRequest("DELETE", baseURL+armPath(path), nil)
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
	data, err := json.Marshal(body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to encode request body: %v\n", err)
		os.Exit(1)
	}
	resp, err := http.Post(baseURL+armPath(path), "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	printResponse(resp)
}

func printResponse(resp *http.Response) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read response: %v\n", err)
		return
	}
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
		fmt.Println(`Usage: azlocal storage <account|container|blob> <subcommand> [flags]`)
		return
	}
	switch args[0] {
	case "account":
		handleStorageAccount(args[1:])
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

func handleStorageAccount(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: azlocal storage account <create|list|show|delete|list-keys> [flags]`)
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.Storage/storageAccounts"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		sku := getFlag(args, "sku")
		if sku == "" {
			sku = "Standard_LRS"
		}
		doPut(base+"/"+name, map[string]interface{}{
			"location": location,
			"sku": map[string]string{
				"name": sku,
			},
			"kind": "StorageV2",
		})
	case "list":
		doGet(base)
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	case "list-keys":
		name := requireFlag(args, "name")
		doPost(base+"/"+name+"/listKeys", nil)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: storage account %s\n", args[0])
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

// --- DNS ---

func handleDNS(args []string) {
	if len(args) < 2 {
		fmt.Println(`Usage:
  azlocal dns zone <create|list|show|delete> [flags]
  azlocal dns record <create|show|delete> [flags]`)
		return
	}
	switch args[0] {
	case "zone":
		handleDNSZone(args[1:])
	case "record":
		handleDNSRecord(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: dns %s\n", args[0])
	}
}

func handleDNSZone(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal dns zone <create|list|show|delete> [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.Network/dnsZones"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		doPut(base+"/"+name, map[string]interface{}{
			"location": "global",
		})
	case "list":
		doGet(base)
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: dns zone %s\n", args[0])
	}
}

func handleDNSRecord(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal dns record <create|show|delete> --zone ZONE --type TYPE --name NAME [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	zone := requireFlag(args, "zone")
	recordType := requireFlag(args, "type")
	name := requireFlag(args, "name")
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.Network/dnsZones/" + zone + "/" + recordType + "/" + name

	switch args[0] {
	case "create":
		data := getFlag(args, "data")
		var body interface{}
		if data != "" {
			json.Unmarshal([]byte(data), &body)
		}
		if body == nil {
			body = map[string]interface{}{}
		}
		doPut(base, body)
	case "show":
		doGet(base)
	case "delete":
		doDelete(base)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: dns record %s\n", args[0])
	}
}

// --- Event Grid ---

func handleEventGrid(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: azlocal eventgrid topic <create|list|show|delete> [flags]")
		return
	}
	if args[0] != "topic" {
		fmt.Fprintf(os.Stderr, "Unknown subcommand: eventgrid %s\n", args[0])
		return
	}
	handleEventGridTopic(args[1:])
}

func handleEventGridTopic(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal eventgrid topic <create|list|show|delete> [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.EventGrid/topics"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		doPut(base+"/"+name, map[string]interface{}{
			"location": location,
		})
	case "list":
		doGet(base)
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: eventgrid topic %s\n", args[0])
	}
}

// --- ACR (Azure Container Registry) ---

func handleACR(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal acr <create|list|show|delete> [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.ContainerRegistry/registries"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		sku := getFlag(args, "sku")
		if sku == "" {
			sku = "Basic"
		}
		doPut(base+"/"+name, map[string]interface{}{
			"location": location,
			"sku": map[string]string{
				"name": sku,
			},
		})
	case "list":
		doGet(base)
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: acr %s\n", args[0])
	}
}

// --- PostgreSQL ---

func handlePostgres(args []string) {
	if len(args) < 2 {
		fmt.Println(`Usage:
  azlocal postgres server <create|list|show|delete> [flags]
  azlocal postgres database <create|list|delete> [flags]`)
		return
	}
	switch args[0] {
	case "server":
		handlePostgresServer(args[1:])
	case "database":
		handlePostgresDatabase(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: postgres %s\n", args[0])
	}
}

func handlePostgresServer(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal postgres server <create|list|show|delete> [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.DBforPostgreSQL/flexibleServers"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		doPut(base+"/"+name, map[string]interface{}{
			"location":   location,
			"properties": map[string]interface{}{},
		})
	case "list":
		doGet(base)
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: postgres server %s\n", args[0])
	}
}

func handlePostgresDatabase(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal postgres database <create|list|delete> [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	server := requireFlag(args, "server")
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.DBforPostgreSQL/flexibleServers/" + server + "/databases"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		doPut(base+"/"+name, map[string]interface{}{
			"properties": map[string]interface{}{},
		})
	case "list":
		doGet(base)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: postgres database %s\n", args[0])
	}
}

// --- Redis ---

func handleRedis(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal redis <create|list|show|delete|list-keys> [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.Cache/redis"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		doPut(base+"/"+name, map[string]interface{}{
			"location": location,
			"properties": map[string]interface{}{
				"sku": map[string]interface{}{
					"name":     "Basic",
					"family":   "C",
					"capacity": 1,
				},
			},
		})
	case "list":
		doGet(base)
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	case "list-keys":
		name := requireFlag(args, "name")
		doPost(base+"/"+name+"/listKeys", nil)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: redis %s\n", args[0])
	}
}

// --- SQL Database ---

func handleSQL(args []string) {
	if len(args) < 2 {
		fmt.Println(`Usage:
  azlocal sql server <create|list|show|delete> [flags]
  azlocal sql database <create|list|delete> [flags]`)
		return
	}
	switch args[0] {
	case "server":
		handleSQLServer(args[1:])
	case "database":
		handleSQLDatabase(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: sql %s\n", args[0])
	}
}

func handleSQLServer(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal sql server <create|list|show|delete> [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.Sql/servers"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		doPut(base+"/"+name, map[string]interface{}{
			"location": location,
			"properties": map[string]interface{}{
				"administratorLogin": getFlag(args, "admin-user"),
			},
		})
	case "list":
		doGet(base)
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: sql server %s\n", args[0])
	}
}

func handleSQLDatabase(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal sql database <create|list|delete> [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	server := requireFlag(args, "server")
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.Sql/servers/" + server + "/databases"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		doPut(base+"/"+name, map[string]interface{}{
			"location":   location,
			"properties": map[string]interface{}{},
		})
	case "list":
		doGet(base)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: sql database %s\n", args[0])
	}
}

// --- MySQL ---

func handleMySQL(args []string) {
	if len(args) < 2 {
		fmt.Println(`Usage:
  azlocal mysql server <create|list|show|delete> [flags]
  azlocal mysql database <create|list|delete> [flags]`)
		return
	}
	switch args[0] {
	case "server":
		handleMySQLServer(args[1:])
	case "database":
		handleMySQLDatabase(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: mysql %s\n", args[0])
	}
}

func handleMySQLServer(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal mysql server <create|list|show|delete> [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.DBforMySQL/flexibleServers"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		doPut(base+"/"+name, map[string]interface{}{
			"location": location,
			"properties": map[string]interface{}{
				"administratorLogin": getFlag(args, "admin-user"),
			},
		})
	case "list":
		doGet(base)
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: mysql server %s\n", args[0])
	}
}

func handleMySQLDatabase(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal mysql database <create|list|delete> [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	server := requireFlag(args, "server")
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.DBforMySQL/flexibleServers/" + server + "/databases"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		doPut(base+"/"+name, map[string]interface{}{
			"properties": map[string]interface{}{},
		})
	case "list":
		doGet(base)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: mysql database %s\n", args[0])
	}
}

// --- ACI (Azure Container Instances) ---

func handleACI(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal aci <create|list|show|delete> [flags]")
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.ContainerInstance/containerGroups"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		image := getFlag(args, "image")
		if image == "" {
			image = "nginx:latest"
		}
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		doPut(base+"/"+name, map[string]interface{}{
			"location": location,
			"properties": map[string]interface{}{
				"osType": "Linux",
				"containers": []interface{}{
					map[string]interface{}{
						"name": name,
						"properties": map[string]interface{}{
							"image": image,
							"ports": []interface{}{
								map[string]interface{}{"port": 80},
							},
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":        1.0,
									"memoryInGB": 1.5,
								},
							},
						},
					},
				},
				"ipAddress": map[string]interface{}{
					"type": "Public",
					"ports": []interface{}{
						map[string]interface{}{
							"protocol": "TCP",
							"port":     80,
						},
					},
				},
			},
		})
	case "list":
		doGet(base)
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: aci %s\n", args[0])
	}
}

// --- AKS ---

func handleAKS(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage:
  azlocal aks create --resource-group RG --name NAME [--location eastus] [--node-count 1] [--kubernetes-version 1.30.0]
  azlocal aks list --resource-group RG
  azlocal aks show --resource-group RG --name NAME
  azlocal aks delete --resource-group RG --name NAME
  azlocal aks get-credentials --resource-group RG --name NAME [--file PATH|-] [--overwrite-existing]
                                                                          (default: merge into ~/.kube/config; --file - for stdout)`)
		return
	}
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.ContainerService/managedClusters"

	switch args[0] {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		nodeCount := 1
		if v := getFlag(args, "node-count"); v != "" {
			fmt.Sscanf(v, "%d", &nodeCount)
		}
		kubeVersion := getFlag(args, "kubernetes-version")
		if kubeVersion == "" {
			kubeVersion = "1.30.0"
		}
		doPut(base+"/"+name, map[string]interface{}{
			"location": location,
			"identity": map[string]interface{}{"type": "SystemAssigned"},
			"properties": map[string]interface{}{
				"kubernetesVersion": kubeVersion,
				"dnsPrefix":         name,
				"agentPoolProfiles": []interface{}{
					map[string]interface{}{
						"name":   "default",
						"count":  nodeCount,
						"vmSize": "Standard_DS2_v2",
						"mode":   "System",
					},
				},
			},
		})
	case "list":
		doGet(base)
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	case "get-credentials":
		name := requireFlag(args, "name")
		writeKubeconfig(base+"/"+name+"/listClusterAdminCredential", getFlag(args, "file"), hasFlag(args, "overwrite-existing"))
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: aks %s\n", args[0])
	}
}

// writeKubeconfig POSTs to a listClusterAdminCredential endpoint and writes
// the decoded kubeconfig either to the path given by --file (or ~/.kube/config
// when omitted), or to stdout when --file=-.
//
// When the target file already exists and overwrite is false (the default,
// matching `az aks get-credentials`), new clusters/contexts/users entries are
// merged in: same-name entries are replaced; the new current-context is set.
// With overwrite=true the file is replaced entirely.
func writeKubeconfig(path, file string, overwrite bool) {
	resp, err := http.Post(baseURL+armPath(path), "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		printResponse(resp)
		os.Exit(1)
	}
	var body struct {
		Kubeconfigs []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"kubeconfigs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to parse credentials response: %v\n", err)
		os.Exit(1)
	}
	if len(body.Kubeconfigs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: empty kubeconfigs in response")
		os.Exit(1)
	}
	cfg, err := base64.StdEncoding.DecodeString(body.Kubeconfigs[0].Value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to base64-decode kubeconfig: %v\n", err)
		os.Exit(1)
	}

	if file == "-" {
		os.Stdout.Write(cfg)
		return
	}
	if file == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot determine home dir for default --file: %v\n", err)
			os.Exit(1)
		}
		file = filepath.Join(home, ".kube", "config")
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: mkdir %s: %v\n", filepath.Dir(file), err)
		os.Exit(1)
	}

	out := cfg
	if !overwrite {
		merged, err := mergeKubeconfig(file, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: merge into %s failed: %v (use --overwrite-existing to replace)\n", file, err)
			os.Exit(1)
		}
		out = merged
	}

	if err := os.WriteFile(file, out, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "Error: write %s: %v\n", file, err)
		os.Exit(1)
	}
	if overwrite {
		fmt.Printf("Wrote kubeconfig to %s (overwritten)\n", file)
	} else {
		fmt.Printf("Merged credentials into %s\n", file)
	}
}

// mergeKubeconfig returns YAML for `file` after merging the clusters,
// contexts, and users from `incoming` into it, replacing same-name entries
// and adopting incoming's current-context. Returns `incoming` unchanged if
// `file` does not exist or cannot be parsed (defensive: treat unreadable
// existing kubeconfig as empty rather than refusing the operation).
func mergeKubeconfig(file string, incoming []byte) ([]byte, error) {
	var newCfg map[string]interface{}
	if err := yaml.Unmarshal(incoming, &newCfg); err != nil {
		return nil, fmt.Errorf("parse incoming kubeconfig: %w", err)
	}

	var existing map[string]interface{}
	if data, err := os.ReadFile(file); err == nil {
		_ = yaml.Unmarshal(data, &existing)
	}
	if existing == nil {
		// Nothing to merge into; just return the incoming kubeconfig.
		return incoming, nil
	}

	for _, section := range []string{"clusters", "contexts", "users"} {
		existing[section] = mergeNamedList(existing[section], newCfg[section])
	}
	if cc, ok := newCfg["current-context"]; ok {
		existing["current-context"] = cc
	}
	if v, ok := newCfg["apiVersion"]; ok {
		existing["apiVersion"] = v
	}
	if v, ok := newCfg["kind"]; ok {
		existing["kind"] = v
	}

	return yaml.Marshal(existing)
}

// mergeNamedList merges two YAML-decoded lists keyed by their "name" field.
// Entries from `incoming` replace same-named entries in `existing`; new
// entries from `incoming` are appended. Order: surviving existing entries
// (in their original order), then incoming entries.
func mergeNamedList(existingI, incomingI interface{}) []interface{} {
	existing, _ := existingI.([]interface{})
	incoming, _ := incomingI.([]interface{})

	incomingNames := map[string]bool{}
	for _, item := range incoming {
		m, _ := item.(map[string]interface{})
		if m == nil {
			continue
		}
		if name, ok := m["name"].(string); ok {
			incomingNames[name] = true
		}
	}

	out := make([]interface{}, 0, len(existing)+len(incoming))
	for _, item := range existing {
		m, _ := item.(map[string]interface{})
		if m != nil {
			if name, ok := m["name"].(string); ok && incomingNames[name] {
				continue
			}
		}
		out = append(out, item)
	}
	out = append(out, incoming...)
	return out
}

// hasFlag returns true if --name is present in args (no value required).
func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == "--"+name {
			return true
		}
	}
	return false
}

// --- Table Storage ---

func handleTable(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage:
  azlocal table create --account ACCOUNT --name TABLE
  azlocal table delete --account ACCOUNT --name TABLE
  azlocal table entity <put|get|delete> [flags]`)
		return
	}
	switch args[0] {
	case "create":
		account := requireFlag(args, "account")
		name := requireFlag(args, "name")
		// Table create uses POST on the data plane
		doPost("/table/"+account+"/"+name, nil)
	case "delete":
		account := requireFlag(args, "account")
		name := requireFlag(args, "name")
		doDelete("/table/" + account + "/" + name)
	case "entity":
		handleTableEntity(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: table %s\n", args[0])
	}
}

func handleTableEntity(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal table entity <put|get|delete> [flags]")
		return
	}
	account := requireFlag(args, "account")
	table := requireFlag(args, "table")
	pk := requireFlag(args, "partition-key")
	rk := requireFlag(args, "row-key")
	base := "/table/" + account + "/" + table + "/" + pk + "/" + rk

	switch args[0] {
	case "put":
		data := getFlag(args, "data")
		var body interface{}
		if data != "" {
			json.Unmarshal([]byte(data), &body)
		}
		if body == nil {
			body = map[string]interface{}{}
		}
		doPut(base, body)
	case "get":
		doGet(base)
	case "delete":
		doDelete(base)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: table entity %s\n", args[0])
	}
}

// --- Queue Storage ---

func handleQueue(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage:
  azlocal queue create --account ACCOUNT --name QUEUE
  azlocal queue delete --account ACCOUNT --name QUEUE
  azlocal queue message <send|receive|clear> [flags]`)
		return
	}
	switch args[0] {
	case "create":
		account := requireFlag(args, "account")
		name := requireFlag(args, "name")
		doPutRaw("/queue/"+account+"/"+name, "application/json", nil)
	case "delete":
		account := requireFlag(args, "account")
		name := requireFlag(args, "name")
		doDelete("/queue/" + account + "/" + name)
	case "message":
		handleQueueMessage(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: queue %s\n", args[0])
	}
}

func handleQueueMessage(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal queue message <send|receive|clear> [flags]")
		return
	}
	account := requireFlag(args, "account")
	queue := requireFlag(args, "queue")
	base := "/queue/" + account + "/" + queue + "/messages"

	switch args[0] {
	case "send":
		body := requireFlag(args, "body")
		doPost(base, map[string]string{"messageText": body})
	case "receive":
		doGet(base)
	case "clear":
		doDelete(base)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: queue message %s\n", args[0])
	}
}

// --- Container Apps ---

func handleContainerApp(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage:
  azlocal containerapp <create|show|list|delete> [flags]
  azlocal containerapp env <create|show|list|delete> [flags]`)
		return
	}
	switch args[0] {
	case "env":
		handleContainerAppEnv(args[1:])
	case "create", "show", "list", "delete":
		handleContainerAppOps(args[0:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: containerapp %s\n", args[0])
	}
}

func handleContainerAppEnv(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal containerapp env <create|show|list|delete> [flags]")
		return
	}
	// Check if it's not a subcommand (like 'create')
	if args[0] == "create" || args[0] == "show" || args[0] == "list" || args[0] == "delete" {
		handleContainerAppEnvOps(args[0:])
	} else {
		fmt.Println("Usage: azlocal containerapp env <create|show|list|delete> [flags]")
	}
}

func handleContainerAppEnvOps(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal containerapp env <create|show|list|delete> [flags]")
		return
	}
	op := args[0]
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.App/managedEnvironments"

	switch op {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}

		props := map[string]interface{}{}
		logsID := getFlag(args, "logs-workspace-id")
		logsKey := getFlag(args, "logs-workspace-key")
		if logsID != "" && logsKey != "" {
			props["appLogsConfiguration"] = map[string]interface{}{
				"logAnalytics": map[string]string{
					"workspaceId": logsID,
					"sharedKey":   logsKey,
				},
			}
		}

		doPut(base+"/"+name, map[string]interface{}{
			"location":   location,
			"properties": props,
		})
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "list":
		doGet(base)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: containerapp env %s\n", op)
	}
}

func handleContainerAppOps(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: azlocal containerapp <create|show|list|delete> [flags]")
		return
	}
	op := args[0]
	rg := requireFlag(args, "resource-group")
	s := sub(args)
	base := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.App/containerApps"

	switch op {
	case "create":
		name := requireFlag(args, "name")
		location := getFlag(args, "location")
		if location == "" {
			location = "eastus"
		}
		image := getFlag(args, "image")
		if image == "" {
			image = "nginx:latest"
		}
		envName := getFlag(args, "environment")
		ingress := getFlag(args, "ingress")
		if ingress == "" {
			ingress = "external"
		}
		targetPort := getFlag(args, "target-port")
		if targetPort == "" {
			targetPort = "80"
		}
		cpu := getFlag(args, "cpu")
		if cpu == "" {
			cpu = "0.5"
		}
		memory := getFlag(args, "memory")
		if memory == "" {
			memory = "1Gi"
		}

		props := map[string]interface{}{}
		if envName != "" {
			envID := "/subscriptions/" + s + "/resourceGroups/" + rg + "/providers/Microsoft.App/managedEnvironments/" + envName
			props["environmentId"] = envID
			props["managedEnvironmentId"] = envID
		}

		props["configuration"] = map[string]interface{}{
			"ingress": map[string]interface{}{
				"external":   ingress == "external",
				"targetPort": targetPort,
			},
		}

		props["template"] = map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"name":  name,
					"image": image,
					"resources": map[string]interface{}{
						"cpu":    cpu,
						"memory": memory,
					},
				},
			},
		}

		doPut(base+"/"+name, map[string]interface{}{
			"location":   location,
			"properties": props,
		})
	case "show":
		name := requireFlag(args, "name")
		doGet(base + "/" + name)
	case "list":
		doGet(base)
	case "delete":
		name := requireFlag(args, "name")
		doDelete(base + "/" + name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: containerapp %s\n", op)
	}
}
