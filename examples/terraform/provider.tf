provider "azurerm" {
  features {}

  # Point to miniblue metadata endpoint (HTTPS)
  metadata_host = "localhost:4567"

  # Skip provider namespace registration
  skip_provider_registration = true

  # Mock credentials - miniblue accepts anything
  subscription_id = "00000000-0000-0000-0000-000000000000"
  tenant_id       = "00000000-0000-0000-0000-000000000001"
  client_id       = "miniblue"
  client_secret   = "miniblue"
}
