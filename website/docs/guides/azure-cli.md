# Azure CLI

The `az` CLI can be configured to talk to miniblue by registering a custom cloud. This page explains the setup and its limitations.

## Background

The Azure CLI uses MSAL (Microsoft Authentication Library) for login, which validates authority endpoints against Microsoft's servers. This makes a standard `az login` impossible with a local emulator. miniblue ships a helper script that bypasses this by writing a mock token profile directly.

## Setup

### Using the helper script

```bash
# Start miniblue first
docker run -d -p 4566:4566 -p 4567:4567 moabukar/miniblue:latest

# Configure az CLI
bash scripts/az-login-local.sh
```

The script does the following:

1. Registers a custom cloud called `miniblue` pointing at `http://localhost:4566`
2. Sets `miniblue` as the active cloud
3. Writes a mock Azure profile to `~/.azure/azureProfile.json`

### Manual setup

If you prefer to configure it yourself:

```bash
# Remove stale registration (if any)
az cloud unregister --name miniblue 2>/dev/null || true

# Register miniblue as a custom cloud
az cloud register --name miniblue \
  --endpoint-resource-manager "http://localhost:4566" \
  --endpoint-active-directory "http://localhost:4566" \
  --endpoint-active-directory-resource-id "http://localhost:4566" \
  --endpoint-active-directory-graph-resource-id "http://localhost:4566"

# Switch to the miniblue cloud
az cloud set --name miniblue
```

Then create `~/.azure/azureProfile.json`:

```json
{
  "installationId": "miniblue-mock",
  "subscriptions": [
    {
      "id": "00000000-0000-0000-0000-000000000000",
      "name": "miniblue",
      "state": "Enabled",
      "user": {
        "name": "miniblue@localhost",
        "type": "servicePrincipal"
      },
      "isDefault": true,
      "tenantId": "00000000-0000-0000-0000-000000000001",
      "environmentName": "miniblue",
      "homeTenantId": "00000000-0000-0000-0000-000000000001",
      "managedByTenants": []
    }
  ]
}
```

## Usage

Once configured, standard `az` commands work:

```bash
# Create a resource group
az group create --name myRG --location eastus

# List resource groups
az group list

# Show a resource group
az group show --name myRG
```

## Switching back to real Azure

```bash
az cloud set --name AzureCloud
az login
```

!!! warning
    Remember to switch back before running commands against your real Azure subscription.

## Limitations

- `az login` does not work -- MSAL requires real Microsoft authority endpoints
- The mock profile bypasses authentication entirely
- Not all `az` subcommands may produce the expected output (miniblue's API surface is a subset of Azure's)
- For scripting and automation, consider using [azlocal](azlocal.md) or curl instead -- they are simpler and more reliable with miniblue

## Custom port

If miniblue runs on a non-default port, adjust the script:

```bash
LOCAL_AZURE_PORT=8080 bash scripts/az-login-local.sh
```
