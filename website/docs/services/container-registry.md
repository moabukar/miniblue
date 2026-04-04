# Container Registry

miniblue emulates Azure Container Registry (ACR) with ARM management endpoints and basic Docker registry v2 API stubs.

## API endpoints

### ARM (management)

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerRegistry/registries/{name}` | Create or update |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerRegistry/registries/{name}` | Get |
| `DELETE` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerRegistry/registries/{name}` | Delete |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerRegistry/registries` | List |
| `POST` | `/subscriptions/{sub}/providers/Microsoft.ContainerRegistry/checkNameAvailability` | Check name |

### Docker Registry v2 (stubs)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/acr/{registry}/v2/{repository}/manifests` | List manifests |
| `GET` | `/acr/{registry}/v2/{repository}/manifests/{ref}` | Get manifest |
| `GET` | `/acr/{registry}/v2/{repository}/tags/list` | List tags |

## Create a registry

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.ContainerRegistry/registries/myregistry?api-version=2020-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "location": "eastus",
    "sku": {
      "name": "Basic"
    }
  }'
```

Response (`201 Created`):

```json
{
  "id": "/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.ContainerRegistry/registries/myregistry?api-version=2020-06-01",
  "name": "myregistry",
  "type": "Microsoft.ContainerRegistry/registries",
  "location": "eastus",
  "sku": {
    "name": "Basic",
    "tier": "Basic"
  },
  "properties": {
    "loginServer": "myregistry.azurecr.io",
    "provisioningState": "Succeeded",
    "adminUserEnabled": false,
    "creationDate": "2026-01-01T00:00:00Z",
    "publicNetworkAccess": "Enabled",
    "zoneRedundancy": "Disabled",
    "networkRuleBypassOptions": "AzureServices",
    "dataEndpointEnabled": false,
    "encryption": { "status": "disabled" },
    "networkRuleSet": { "defaultAction": "Allow", "ipRules": [] },
    "policies": {
      "quarantinePolicy": { "status": "disabled" },
      "trustPolicy": { "status": "disabled", "type": "Notary" },
      "retentionPolicy": { "status": "disabled", "days": 7 },
      "exportPolicy": { "status": "enabled" }
    }
  }
}
```

## Get a registry

```bash
curl "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.ContainerRegistry/registries/myregistry?api-version=2020-06-01"
```

## List registries

```bash
curl "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.ContainerRegistry/registries?api-version=2020-06-01"
```

## Check name availability

```bash
curl -X POST "http://localhost:4566/subscriptions/sub1/providers/Microsoft.ContainerRegistry/checkNameAvailability?api-version=2020-06-01" \
  -H "Content-Type: application/json" \
  -d '{"name": "myregistry", "type": "Microsoft.ContainerRegistry/registries"}'
```

Response:

```json
{
  "nameAvailable": true
}
```

If the name is already taken:

```json
{
  "nameAvailable": false,
  "reason": "AlreadyExists",
  "message": "The registry myregistry is already in use."
}
```

## Delete a registry

```bash
curl -X DELETE "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.ContainerRegistry/registries/myregistry?api-version=2020-06-01"
```

Response: `202 Accepted`

## Docker Registry v2 stubs

These endpoints return empty/stub responses for compatibility:

```bash
# List manifests (returns empty array)
curl "http://localhost:4566/acr/myregistry/v2/myapp/manifests"

# Get a manifest
curl "http://localhost:4566/acr/myregistry/v2/myapp/manifests/latest"

# List tags (returns empty array)
curl "http://localhost:4566/acr/myregistry/v2/myapp/tags/list"
```

!!! info
    The Docker v2 endpoints are stubs for basic compatibility. Actual image push/pull is not yet supported.

## Terraform

```hcl
resource "azurerm_container_registry" "example" {
  name                = "exampleregistry"
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_resource_group.example.location
  sku                 = "Basic"
}
```

Supported SKU values: `Basic`, `Standard`, `Premium`.

See the [Terraform guide](../guides/terraform.md) for full provider configuration.
