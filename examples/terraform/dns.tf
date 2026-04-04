resource "azurerm_dns_zone" "example" {
  name                = "example.local"
  resource_group_name = azurerm_resource_group.example.name
}
