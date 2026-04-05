# Microservices Architecture on miniblue
# Multiple services with VNet isolation, ACR, DNS-based discovery
#
# Usage:
#   export SSL_CERT_FILE=~/.miniblue/cert.pem
#   terraform init && terraform apply -auto-approve
#
# Test with azlocal after apply:
#   azlocal group list
#   azlocal network vnet show --name microservices-vnet --resource-group shared-infra-rg
#   azlocal acr show --name sharedmicroacr --resource-group shared-infra-rg
#   azlocal dns zone show --name svc.internal --resource-group shared-infra-rg
#   azlocal eventgrid topic show --name notification-events --resource-group svc-notify-rg
#
# Destroy:
#   terraform destroy -auto-approve

# --- Shared Infrastructure ---

resource "azurerm_resource_group" "shared" {
  name     = "shared-infra-rg"
  location = "East US"
}

resource "azurerm_container_registry" "shared" {
  name                = "sharedmicroacr"
  resource_group_name = azurerm_resource_group.shared.name
  location            = azurerm_resource_group.shared.location
  sku                 = "Standard"
}

resource "azurerm_virtual_network" "shared" {
  name                = "microservices-vnet"
  address_space       = ["10.0.0.0/8"]
  location            = azurerm_resource_group.shared.location
  resource_group_name = azurerm_resource_group.shared.name
}

resource "azurerm_dns_zone" "internal" {
  name                = "svc.internal"
  resource_group_name = azurerm_resource_group.shared.name
}

# --- Service A: User API ---

resource "azurerm_resource_group" "svc_users" {
  name     = "svc-users-rg"
  location = "East US"
}

resource "azurerm_subnet" "svc_users" {
  name                 = "svc-users-subnet"
  resource_group_name  = azurerm_resource_group.shared.name
  virtual_network_name = azurerm_virtual_network.shared.name
  address_prefixes     = ["10.1.0.0/16"]
}

# --- Service B: Order API ---

resource "azurerm_resource_group" "svc_orders" {
  name     = "svc-orders-rg"
  location = "East US"
}

resource "azurerm_subnet" "svc_orders" {
  name                 = "svc-orders-subnet"
  resource_group_name  = azurerm_resource_group.shared.name
  virtual_network_name = azurerm_virtual_network.shared.name
  address_prefixes     = ["10.2.0.0/16"]
}

# --- Service C: Notification Service ---

resource "azurerm_resource_group" "svc_notify" {
  name     = "svc-notify-rg"
  location = "East US"
}

resource "azurerm_subnet" "svc_notify" {
  name                 = "svc-notify-subnet"
  resource_group_name  = azurerm_resource_group.shared.name
  virtual_network_name = azurerm_virtual_network.shared.name
  address_prefixes     = ["10.3.0.0/16"]
}

resource "azurerm_eventgrid_topic" "notifications" {
  name                = "notification-events"
  location            = azurerm_resource_group.svc_notify.location
  resource_group_name = azurerm_resource_group.svc_notify.name
}
