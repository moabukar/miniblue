# Example Terraform config for local-azure
# Run: terraform init && terraform apply

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

  # Point to local-azure
  resource_manager_endpoint         = "http://localhost:4566"
  skip_provider_registration        = true
  use_cli                           = false
  use_msi                           = false

  # Use mock credentials
  subscription_id = "00000000-0000-0000-0000-000000000000"
  tenant_id       = "00000000-0000-0000-0000-000000000001"
  client_id       = "local-azure"
  client_secret   = "local-azure"
}

resource "azurerm_resource_group" "example" {
  name     = "example-rg"
  location = "East US"
}
