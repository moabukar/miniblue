# Virtual Networks

miniblue emulates Azure Virtual Networks (VNets) and subnets via the ARM API. These endpoints are compatible with the Terraform `azurerm_virtual_network` and `azurerm_subnet` resources.

## API endpoints

### Virtual Networks

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{name}` | Create or update |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{name}` | Get |
| `DELETE` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{name}` | Delete |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks` | List |

### Subnets

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `.../virtualNetworks/{vnet}/subnets/{subnet}` | Create or update |
| `GET` | `.../virtualNetworks/{vnet}/subnets/{subnet}` | Get |
| `DELETE` | `.../virtualNetworks/{vnet}/subnets/{subnet}` | Delete |
| `GET` | `.../virtualNetworks/{vnet}/subnets` | List |

## Create a VNet

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/my-vnet?api-version=2020-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "location": "eastus",
    "properties": {
      "addressSpace": {
        "addressPrefixes": ["10.0.0.0/16"]
      }
    }
  }'
```

Response (`201 Created`):

```json
{
  "id": "/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/my-vnet?api-version=2020-06-01",
  "name": "my-vnet",
  "type": "Microsoft.Network/virtualNetworks",
  "location": "eastus",
  "etag": "W/\"miniblue\"",
  "properties": {
    "provisioningState": "Succeeded",
    "resourceGuid": "00000000-0000-0000-0000-000000000000",
    "addressSpace": {
      "addressPrefixes": ["10.0.0.0/16"]
    },
    "dhcpOptions": {
      "dnsServers": []
    },
    "subnets": [],
    "virtualNetworkPeerings": [],
    "enableDdosProtection": false,
    "enableVmProtection": false
  }
}
```

## Get a VNet

```bash
curl "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/my-vnet?api-version=2020-06-01"
```

The response includes a `subnets` array populated from the store.

## Create a subnet

The parent VNet must exist first.

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/my-vnet/subnets/my-subnet?api-version=2020-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "properties": {
      "addressPrefix": "10.0.1.0/24"
    }
  }'
```

Response (`201 Created`):

```json
{
  "id": "/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/my-vnet/subnets/my-subnet?api-version=2020-06-01",
  "name": "my-subnet",
  "etag": "W/\"miniblue\"",
  "type": "Microsoft.Network/virtualNetworks/subnets",
  "properties": {
    "provisioningState": "Succeeded",
    "addressPrefix": "10.0.1.0/24",
    "addressPrefixes": ["10.0.1.0/24"],
    "serviceEndpoints": [],
    "delegations": [],
    "privateEndpointNetworkPolicies": "Disabled",
    "privateLinkServiceNetworkPolicies": "Enabled",
    "defaultOutboundAccess": true
  }
}
```

## List subnets

```bash
curl "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/my-vnet/subnets?api-version=2020-06-01"
```

## Delete a subnet

```bash
curl -X DELETE "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/my-vnet/subnets/my-subnet?api-version=2020-06-01"
```

## Delete a VNet

```bash
curl -X DELETE "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/my-vnet?api-version=2020-06-01"
```

Deleting a VNet also deletes all its subnets.

## azlocal

```bash
# Create
azlocal network vnet create --name my-vnet --resource-group myRG \
  --address-prefix 10.0.0.0/16 --location eastus

# Show
azlocal network vnet show --name my-vnet --resource-group myRG

# List
azlocal network vnet list --resource-group myRG

# Delete
azlocal network vnet delete --name my-vnet --resource-group myRG
```

## Terraform

```hcl
resource "azurerm_virtual_network" "example" {
  name                = "example-vnet"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
}

resource "azurerm_subnet" "example" {
  name                 = "example-subnet"
  resource_group_name  = azurerm_resource_group.example.name
  virtual_network_name = azurerm_virtual_network.example.name
  address_prefixes     = ["10.0.1.0/24"]
}
```

See the [Terraform guide](../guides/terraform.md) for full provider configuration.
