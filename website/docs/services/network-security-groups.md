# Network Security Groups

miniblue emulates Azure Network Security Groups (NSGs) via ARM endpoints. NSGs include 6 default security rules and support custom security rule CRUD with cascade delete.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `.../Microsoft.Network/networkSecurityGroups/{name}` | Create or update NSG |
| `GET` | `.../Microsoft.Network/networkSecurityGroups/{name}` | Get NSG |
| `DELETE` | `.../Microsoft.Network/networkSecurityGroups/{name}` | Delete NSG (cascades rules) |
| `GET` | `.../Microsoft.Network/networkSecurityGroups` | List NSGs |
| `PUT` | `.../networkSecurityGroups/{nsg}/securityRules/{rule}` | Create or update rule |
| `GET` | `.../networkSecurityGroups/{nsg}/securityRules/{rule}` | Get rule |
| `DELETE` | `.../networkSecurityGroups/{nsg}/securityRules/{rule}` | Delete rule |
| `GET` | `.../networkSecurityGroups/{nsg}/securityRules` | List rules |

All paths are prefixed with `/subscriptions/{sub}/resourceGroups/{rg}/providers`.

## Create an NSG

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/networkSecurityGroups/my-nsg?api-version=2023-09-01" \
  -H "Content-Type: application/json" \
  -d '{"location": "eastus"}'
```

## Add a security rule

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/networkSecurityGroups/my-nsg/securityRules/allow-http?api-version=2023-09-01" \
  -H "Content-Type: application/json" \
  -d '{
    "properties": {
      "priority": 100,
      "direction": "Inbound",
      "access": "Allow",
      "protocol": "Tcp",
      "sourcePortRange": "*",
      "destinationPortRange": "80",
      "sourceAddressPrefix": "*",
      "destinationAddressPrefix": "*"
    }
  }'
```

## Terraform

```hcl
resource "azurerm_network_security_group" "example" {
  name                = "example-nsg"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name

  security_rule {
    name                       = "allow-http"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "80"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}
```

## Default security rules

Every NSG includes 6 default rules:

- AllowVnetInBound (priority 65000)
- AllowAzureLoadBalancerInBound (priority 65001)
- DenyAllInBound (priority 65500)
- AllowVnetOutBound (priority 65000)
- AllowInternetOutBound (priority 65001)
- DenyAllOutBound (priority 65500)

## Limitations

- No subnet or NIC association tracking
- Rules are not evaluated for traffic filtering
- No application security groups
