# Serverless Event-Driven Architecture on miniblue
# Event Grid + DNS for custom API domain
#
# Usage:
#   export SSL_CERT_FILE=~/.miniblue/cert.pem
#   terraform init && terraform apply -auto-approve
#
# Test with azlocal after apply:
#   azlocal group list
#   azlocal eventgrid topic show --name order-events --resource-group serverless-rg
#   azlocal dns zone show --name api.serverless.local --resource-group serverless-rg
#
# Destroy:
#   terraform destroy -auto-approve

resource "azurerm_resource_group" "serverless" {
  name     = "serverless-rg"
  location = "East US"
}

# --- Event Grid Topic ---

resource "azurerm_eventgrid_topic" "orders" {
  name                = "order-events"
  location            = azurerm_resource_group.serverless.location
  resource_group_name = azurerm_resource_group.serverless.name
}

# --- DNS ---

resource "azurerm_dns_zone" "api" {
  name                = "api.serverless.local"
  resource_group_name = azurerm_resource_group.serverless.name
}
