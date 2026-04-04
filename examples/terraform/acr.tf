resource "azurerm_container_registry" "example" {
  name                = "exampleregistry"
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_resource_group.example.location
  sku                 = "Basic"
}
