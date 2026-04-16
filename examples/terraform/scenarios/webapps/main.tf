# App Service Web Apps Architecture on miniblue
# Service Plan + Linux Web App for testing azurerm providers
#
# Usage:
#   export SSL_CERT_FILE=~/.miniblue/cert.pem
#   terraform init && terraform apply -auto-approve
#
# Test with azlocal after apply:
#   azlocal group list
#   azlocal serviceplan show --name webapp-plan --resource-group webapps-rg
#   azlocal webapp show --name my-linux-webapp --resource-group webapps-rg
#
# Destroy:
#   terraform destroy -auto-approve

resource "azurerm_resource_group" "webapps" {
  name     = "webapps-rg"
  location = "East US"
}

# --- App Service Plan (Server Farm) ---

resource "azurerm_service_plan" "webapp" {
  name                = "webapp-plan"
  resource_group_name = azurerm_resource_group.webapps.name
  location            = azurerm_resource_group.webapps.location
  os_type             = "Linux"
  sku_name            = "P1v2"
}

# --- Linux Web App ---

resource "azurerm_linux_web_app" "webapp" {
  name                = "my-linux-webapp"
  resource_group_name = azurerm_resource_group.webapps.name
  location            = azurerm_resource_group.webapps.location
  service_plan_id     = azurerm_service_plan.webapp.id

  site_config {
    always_on  = false
    ftps_state = "Disabled"
  }
}
