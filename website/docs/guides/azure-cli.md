# CLI Usage

Use `azlocal` to interact with miniblue. It works out of the box with no authentication or setup.

## Install

```bash
# From source
git clone https://github.com/moabukar/miniblue.git
cd miniblue && make build
sudo make install

# Or with Go
go install github.com/moabukar/miniblue/cmd/azlocal@latest
```

## Quick start

```bash
# Start miniblue
miniblue

# In another terminal
azlocal health
azlocal group create --name myRG --location eastus
azlocal group list
```

## All commands

```bash
# Resource groups
azlocal group create --name myRG --location eastus
azlocal group list
azlocal group show --name myRG
azlocal group delete --name myRG

# Key Vault secrets
azlocal keyvault secret set --vault myvault --name db-pass --value secret123
azlocal keyvault secret show --vault myvault --name db-pass
azlocal keyvault secret list --vault myvault
azlocal keyvault secret delete --vault myvault --name db-pass

# Blob storage
azlocal storage container create --account myaccount --name mycontainer
azlocal storage container delete --account myaccount --name mycontainer
azlocal storage blob upload --account myaccount --container mycontainer --name file.txt --data "Hello"
azlocal storage blob upload --account myaccount --container mycontainer --name file.txt --file ./local-file.txt
azlocal storage blob download --account myaccount --container mycontainer --name file.txt
azlocal storage blob list --account myaccount --container mycontainer
azlocal storage blob delete --account myaccount --container mycontainer --name file.txt

# Cosmos DB
azlocal cosmosdb doc create --account db1 --database mydb --collection users --id user1 --data '{"name":"Mo"}'
azlocal cosmosdb doc show --account db1 --database mydb --collection users --id user1
azlocal cosmosdb doc list --account db1 --database mydb --collection users
azlocal cosmosdb doc delete --account db1 --database mydb --collection users --id user1

# Service Bus
azlocal servicebus queue create --namespace ns1 --name orders
azlocal servicebus queue send --namespace ns1 --name orders --body "order-001"
azlocal servicebus queue receive --namespace ns1 --name orders
azlocal servicebus queue delete --namespace ns1 --name orders

# Networking
azlocal network vnet create --name my-vnet --resource-group myRG --address-prefix 10.0.0.0/16
azlocal network vnet show --name my-vnet --resource-group myRG
azlocal network vnet list --resource-group myRG
azlocal network vnet delete --name my-vnet --resource-group myRG

# App Configuration
azlocal appconfig kv set --store mystore --key db-host --value localhost:5432
azlocal appconfig kv show --store mystore --key db-host
azlocal appconfig kv list --store mystore
azlocal appconfig kv delete --store mystore --key db-host

# Function Apps
azlocal functionapp create --name my-func --resource-group myRG --location eastus
azlocal functionapp show --name my-func --resource-group myRG
azlocal functionapp list --resource-group myRG
azlocal functionapp delete --name my-func --resource-group myRG

# Health
azlocal health
```

## Custom endpoint

By default azlocal connects to `http://localhost:4566`. Override with:

```bash
LOCAL_AZURE_ENDPOINT=http://my-server:4566 azlocal health
```

## Why not the native Azure CLI?

The native `az` CLI uses Microsoft's MSAL authentication library which requires real Azure AD token negotiation. This can't be reliably emulated locally. `azlocal` bypasses this entirely by talking directly to miniblue's HTTP API.

This is the same approach used by other emulators:

- LocalStack uses `awslocal` instead of `aws`
- MiniStack uses `awslocal` instead of `aws`
