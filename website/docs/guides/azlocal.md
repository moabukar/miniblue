# azlocal CLI

`azlocal` is a command-line client for miniblue, similar to `awslocal` for LocalStack. It wraps HTTP calls to the miniblue API -- no authentication required.

## Installation

```bash
# Build from source
git clone https://github.com/moabukar/miniblue.git
cd miniblue
make build

# The binary is at bin/azlocal

# Install globally (optional)
sudo make install
# -> copies to /usr/local/bin/azlocal
```

Or with `go install`:

```bash
go install github.com/moabukar/miniblue/cmd/azlocal@latest
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LOCAL_AZURE_ENDPOINT` | `http://localhost:4566` | miniblue endpoint URL |

```bash
# Point to a remote miniblue instance
export LOCAL_AZURE_ENDPOINT=http://192.168.1.100:4566
azlocal health
```

## Commands

### health

Check if miniblue is running:

```bash
azlocal health
```

```json
{
  "services": ["subscriptions","tenants","resourcegroups","blob","table","queue","keyvault","cosmosdb","servicebus","functions","network","dns","acr","eventgrid","appconfig","identity"],
  "status": "running",
  "version": "0.1.0"
}
```

---

### group (Resource Groups)

```bash
# Create a resource group
azlocal group create --name myRG --location eastus

# Create with a specific subscription
azlocal group create --name myRG --location westus2 --subscription my-sub-id

# List all resource groups
azlocal group list
azlocal group list --subscription my-sub-id

# Show a specific resource group
azlocal group show --name myRG
azlocal group show --name myRG --subscription my-sub-id

# Delete a resource group
azlocal group delete --name myRG
azlocal group delete --name myRG --subscription my-sub-id
```

!!! note
    If `--subscription` is omitted, it defaults to `default`.

---

### keyvault (Key Vault)

```bash
# Set a secret
azlocal keyvault secret set --vault myvault --name db-password --value "P@ssw0rd123!"

# Get a secret
azlocal keyvault secret show --vault myvault --name db-password

# List all secrets in a vault
azlocal keyvault secret list --vault myvault

# Delete a secret
azlocal keyvault secret delete --vault myvault --name db-password
```

---

### storage (Blob Storage)

#### Containers

```bash
# Create a container
azlocal storage container create --account myaccount --name mycontainer

# Delete a container
azlocal storage container delete --account myaccount --name mycontainer
```

#### Blobs

```bash
# Upload from a string
azlocal storage blob upload --account myaccount --container mycontainer \
  --name hello.txt --data "Hello from miniblue!"

# Upload from a file
azlocal storage blob upload --account myaccount --container mycontainer \
  --name config.json --file ./config.json

# Download a blob
azlocal storage blob download --account myaccount --container mycontainer \
  --name hello.txt

# List blobs in a container
azlocal storage blob list --account myaccount --container mycontainer

# Delete a blob
azlocal storage blob delete --account myaccount --container mycontainer \
  --name hello.txt
```

---

### network (Virtual Networks)

```bash
# Create a VNet
azlocal network vnet create --name my-vnet --resource-group myRG \
  --address-prefix 10.0.0.0/16 --location eastus

# Show a VNet
azlocal network vnet show --name my-vnet --resource-group myRG

# List VNets
azlocal network vnet list --resource-group myRG

# Delete a VNet
azlocal network vnet delete --name my-vnet --resource-group myRG
```

All network commands require `--resource-group`. Optionally pass `--subscription` (defaults to `default`).

---

### cosmosdb (Cosmos DB)

```bash
# Create a document
azlocal cosmosdb doc create --account myaccount --database mydb \
  --collection mycoll --id doc1 --data '{"name":"Alice","age":30}'

# Get a document
azlocal cosmosdb doc show --account myaccount --database mydb \
  --collection mycoll --id doc1

# List all documents
azlocal cosmosdb doc list --account myaccount --database mydb \
  --collection mycoll

# Delete a document
azlocal cosmosdb doc delete --account myaccount --database mydb \
  --collection mycoll --id doc1
```

---

### servicebus (Service Bus)

#### Queues

```bash
# Create a queue
azlocal servicebus queue create --namespace my-ns --name my-queue

# Send a message
azlocal servicebus queue send --namespace my-ns --name my-queue \
  --body "Hello from Service Bus!"

# Receive (peek) a message
azlocal servicebus queue receive --namespace my-ns --name my-queue

# Delete a queue
azlocal servicebus queue delete --namespace my-ns --name my-queue
```

#### Topics

```bash
# Create a topic
azlocal servicebus topic create --namespace my-ns --name my-topic

# Publish a message
azlocal servicebus topic send --namespace my-ns --name my-topic \
  --body "Broadcast message"

# Delete a topic
azlocal servicebus topic delete --namespace my-ns --name my-topic
```

---

### appconfig (App Configuration)

```bash
# Set a key-value pair
azlocal appconfig kv set --store mystore --key feature-flag --value "true"

# Get a value
azlocal appconfig kv show --store mystore --key feature-flag

# List all key-value pairs
azlocal appconfig kv list --store mystore

# Delete a key-value pair
azlocal appconfig kv delete --store mystore --key feature-flag
```

---

### functionapp (Azure Functions)

```bash
# Create a function app
azlocal functionapp create --name my-func --resource-group myRG --location eastus

# Show a function app
azlocal functionapp show --name my-func --resource-group myRG

# List function apps
azlocal functionapp list --resource-group myRG

# Delete a function app
azlocal functionapp delete --name my-func --resource-group myRG
```

---

## Full command reference

| Command | Subcommands |
|---------|-------------|
| `azlocal health` | _(none)_ |
| `azlocal group` | `create`, `list`, `show`, `delete` |
| `azlocal keyvault secret` | `set`, `show`, `list`, `delete` |
| `azlocal storage container` | `create`, `delete` |
| `azlocal storage blob` | `upload`, `download`, `list`, `delete` |
| `azlocal network vnet` | `create`, `show`, `list`, `delete` |
| `azlocal cosmosdb doc` | `create`, `show`, `list`, `delete` |
| `azlocal servicebus queue` | `create`, `send`, `receive`, `delete` |
| `azlocal servicebus topic` | `create`, `send`, `delete` |
| `azlocal appconfig kv` | `set`, `show`, `list`, `delete` |
| `azlocal functionapp` | `create`, `show`, `list`, `delete` |

## Scripting example

```bash
#!/bin/bash
set -e

# Set up infrastructure
azlocal group create --name dev-rg --location eastus
azlocal keyvault secret set --vault dev-vault --name api-key --value "sk-12345"
azlocal storage container create --account devstore --name uploads
azlocal storage blob upload --account devstore --container uploads \
  --name seed-data.json --file ./fixtures/seed-data.json

echo "Local environment ready."
```
