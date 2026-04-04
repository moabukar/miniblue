# Three-Tier Architecture on miniblue
# Web tier + App tier + Data tier with VNet isolation
#
# Usage:
#   export SSL_CERT_FILE=~/.miniblue/cert.pem
#   terraform init && terraform apply -auto-approve

# --- Foundation ---

resource "azurerm_resource_group" "main" {
  name     = "three-tier-rg"
  location = "East US"
}

# --- Networking ---

resource "azurerm_virtual_network" "main" {
  name                = "main-vnet"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
}

resource "azurerm_subnet" "web" {
  name                 = "web-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.1.0/24"]
}

resource "azurerm_subnet" "app" {
  name                 = "app-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.2.0/24"]
}

resource "azurerm_subnet" "data" {
  name                 = "data-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.3.0/24"]
}

# --- DNS ---

resource "azurerm_dns_zone" "main" {
  name                = "three-tier.local"
  resource_group_name = azurerm_resource_group.main.name
}

# --- Container Registry ---

resource "azurerm_container_registry" "main" {
  name                = "threetieracr"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  sku                 = "Basic"
}
