# Terraform + local-azure example
# Start local-azure first: ./bin/local-azure
# Then: terraform init && terraform apply -auto-approve

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
  # The provider fetches /metadata/endpoints and uses the returned URLs
  metadata_host = "localhost:4566"

  # Skip provider namespace registration (no real Azure)
  skip_provider_registration = true

  # Mock credentials - local-azure accepts anything
  subscription_id = "00000000-0000-0000-0000-000000000000"
  tenant_id       = "00000000-0000-0000-0000-000000000001"
  client_id       = "local-azure"
  client_secret   = "local-azure"
}

resource "azurerm_resource_group" "example" {
  name     = "example-rg"
  location = "East US"
}
