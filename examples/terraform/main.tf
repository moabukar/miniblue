# Terraform + local-azure example
# Start local-azure first: ./bin/local-azure
# Then: export SSL_CERT_FILE=~/.local-azure/cert.pem
#        terraform init && terraform apply -auto-approve

terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

provider "azurerm" {
  features {}

  # Point to local-azure metadata endpoint
  metadata_host = "localhost:4567"

  # Skip provider namespace registration
  skip_provider_registration = true

  # Mock credentials - local-azure accepts anything
  subscription_id = "00000000-0000-0000-0000-000000000000"
  tenant_id       = "00000000-0000-0000-0000-000000000001"
  client_id       = "local-azure"
  client_secret   = "local-azure"
}

# --- Resource Group ---
resource "azurerm_resource_group" "example" {
  name     = "example-rg"
  location = "East US"
}

# --- Virtual Network ---
resource "azurerm_virtual_network" "example" {
  name                = "example-vnet"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
}

# --- Subnet ---
resource "azurerm_subnet" "example" {
  name                 = "example-subnet"
  resource_group_name  = azurerm_resource_group.example.name
  virtual_network_name = azurerm_virtual_network.example.name
  address_prefixes     = ["10.0.1.0/24"]
}

# --- DNS Zone ---
resource "azurerm_dns_zone" "example" {
  name                = "example.local"
  resource_group_name = azurerm_resource_group.example.name
}

# --- Container Registry ---
resource "azurerm_container_registry" "example" {
  name                = "exampleregistry"
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_resource_group.example.location
  sku                 = "Basic"
}
