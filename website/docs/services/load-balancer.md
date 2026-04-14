# Load Balancer

miniblue emulates Azure Load Balancer via ARM endpoints. Supports frontend IP configurations, backend address pools, load balancing rules, probes, inbound NAT rules and outbound rules.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/loadBalancers/{name}` | Create or update |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/loadBalancers/{name}` | Get |
| `DELETE` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/loadBalancers/{name}` | Delete |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/loadBalancers` | List |

## Create a load balancer

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/loadBalancers/my-lb?api-version=2023-09-01" \
  -H "Content-Type: application/json" \
  -d '{
    "location": "eastus",
    "sku": {"name": "Standard", "tier": "Regional"},
    "properties": {
      "frontendIPConfigurations": [{"name": "frontend", "properties": {"publicIPAddress": {"id": "/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/publicIPAddresses/my-pip"}}}],
      "backendAddressPools": [{"name": "backend-pool"}],
      "probes": [{"name": "http-probe", "properties": {"protocol": "Tcp", "port": 80}}],
      "loadBalancingRules": [{"name": "http-rule", "properties": {"protocol": "Tcp", "frontendPort": 80, "backendPort": 80}}]
    }
  }'
```

## Terraform

```hcl
resource "azurerm_lb" "example" {
  name                = "example-lb"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  sku                 = "Standard"

  frontend_ip_configuration {
    name                 = "frontend"
    public_ip_address_id = azurerm_public_ip.example.id
  }
}
```

## Limitations

- No actual traffic routing or health checking
- Backend pool members are stored but not validated
- No cross-region load balancer support
