resource "azurerm_resource_group" "example" {
  name     = "aks-example-rg"
  location = "East US"
}

resource "azurerm_kubernetes_cluster" "example" {
  name                = "miniblue-aks"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  dns_prefix          = "miniblue"

  default_node_pool {
    name       = "default"
    node_count = 1
    vm_size    = "Standard_DS2_v2"
  }

  identity {
    type = "SystemAssigned"
  }
}

output "kube_config_host" {
  value = azurerm_kubernetes_cluster.example.kube_config[0].host
}

output "node_resource_group" {
  value = azurerm_kubernetes_cluster.example.node_resource_group
}
