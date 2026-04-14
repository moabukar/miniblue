# Application Gateway

miniblue emulates Azure Application Gateway via ARM endpoints. Supports full L7 configuration including SKU, gateway IP configs, frontend IPs/ports, backend pools, HTTP settings, listeners, routing rules, probes, SSL certificates, URL path maps, redirect configs and WAF.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/applicationGateways/{name}` | Create or update |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/applicationGateways/{name}` | Get |
| `DELETE` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/applicationGateways/{name}` | Delete |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/applicationGateways` | List |

## Create an application gateway

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/applicationGateways/my-appgw?api-version=2023-09-01" \
  -H "Content-Type: application/json" \
  -d '{
    "location": "eastus",
    "properties": {
      "sku": {"name": "Standard_v2", "tier": "Standard_v2", "capacity": 2},
      "frontendIPConfigurations": [{"name": "fe-ip"}],
      "frontendPorts": [{"name": "port80", "properties": {"port": 80}}],
      "backendAddressPools": [{"name": "backend"}],
      "backendHttpSettingsCollection": [{"name": "settings", "properties": {"port": 80, "protocol": "Http"}}],
      "httpListeners": [{"name": "listener", "properties": {"protocol": "Http"}}],
      "requestRoutingRules": [{"name": "rule1", "properties": {"ruleType": "Basic", "priority": 100}}]
    }
  }'
```

## Terraform

```hcl
resource "azurerm_application_gateway" "example" {
  name                = "example-appgw"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name

  sku {
    name     = "Standard_v2"
    tier     = "Standard_v2"
    capacity = 2
  }

  gateway_ip_configuration {
    name      = "gw-ip"
    subnet_id = azurerm_subnet.appgw.id
  }

  frontend_ip_configuration {
    name                 = "frontend"
    public_ip_address_id = azurerm_public_ip.appgw.id
  }

  frontend_port {
    name = "http"
    port = 80
  }

  backend_address_pool {
    name = "backend"
  }

  backend_http_settings {
    name                  = "settings"
    cookie_based_affinity = "Disabled"
    port                  = 80
    protocol              = "Http"
  }

  http_listener {
    name                           = "listener"
    frontend_ip_configuration_name = "frontend"
    frontend_port_name             = "http"
    protocol                       = "Http"
  }

  request_routing_rule {
    name                       = "rule"
    priority                   = 100
    rule_type                  = "Basic"
    http_listener_name         = "listener"
    backend_address_pool_name  = "backend"
    backend_http_settings_name = "settings"
  }
}
```

## Supported SKUs

- Standard_v2
- WAF_v2

Default SKU is Standard_v2 with capacity 2 if not specified.

## Limitations

- No actual L7 routing or traffic processing
- WAF configuration is stored but not enforced
- No SSL termination
- No autoscaling (capacity is static)
