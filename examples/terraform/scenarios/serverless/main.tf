# Serverless Event-Driven Architecture on miniblue
# Functions + Event Grid + Service Bus + Key Vault + App Configuration
#
# Usage:
#   export SSL_CERT_FILE=~/.miniblue/cert.pem
#   terraform init && terraform apply -auto-approve

terraform {
  required_providers {
    azurerm = { source = "hashicorp/azurerm", version = "~> 3.0" }
  }
}

provider "azurerm" {
  features {}
  metadata_host              = "localhost:4567"
  skip_provider_registration = true
  subscription_id            = "00000000-0000-0000-0000-000000000000"
  tenant_id                  = "00000000-0000-0000-0000-000000000001"
  client_id                  = "miniblue"
  client_secret              = "miniblue"
}

# --- Foundation ---

resource "azurerm_resource_group" "serverless" {
  name     = "serverless-rg"
  location = "East US"
}

# --- Event Grid Topic (event ingestion) ---

resource "azurerm_eventgrid_topic" "orders" {
  name                = "order-events"
  location            = azurerm_resource_group.serverless.location
  resource_group_name = azurerm_resource_group.serverless.name
}

# --- DNS (custom domain for API) ---

resource "azurerm_dns_zone" "api" {
  name                = "api.serverless.local"
  resource_group_name = azurerm_resource_group.serverless.name
}
