# Public IP Addresses

miniblue emulates Azure Public IP Addresses via ARM endpoints. IPs are auto-generated with mock addresses in the `20.0.x.x` range.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/publicIPAddresses/{name}` | Create or update |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/publicIPAddresses/{name}` | Get |
| `DELETE` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/publicIPAddresses/{name}` | Delete |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/publicIPAddresses` | List |

## Create a public IP

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/publicIPAddresses/my-pip?api-version=2023-09-01" \
  -H "Content-Type: application/json" \
  -d '{
    "location": "eastus",
    "sku": {"name": "Standard", "tier": "Regional"},
    "properties": {
      "publicIPAllocationMethod": "Static",
      "publicIPAddressVersion": "IPv4"
    }
  }'
```

## Terraform

```hcl
resource "azurerm_public_ip" "example" {
  name                = "example-pip"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  allocation_method   = "Static"
  sku                 = "Standard"
}
```

## Limitations

- IP addresses are mock values in the `20.0.x.x` range
- DNS settings are stored but not functional
- No availability zone support
