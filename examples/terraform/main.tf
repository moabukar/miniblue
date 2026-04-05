# miniblue Terraform example
#
# Start miniblue first:
#   ./bin/miniblue
#
# Trust the cert:
#   export SSL_CERT_FILE=~/.miniblue/cert.pem
#
# Run:
#   terraform init
#   terraform apply -auto-approve
#
# Test with azlocal after apply:
#   azlocal group list
#   azlocal network vnet show --name example-vnet --resource-group example-rg
#   azlocal dns zone show --name example.local --resource-group example-rg
#   azlocal acr show --name exampleregistry --resource-group example-rg
#
# Destroy:
#   terraform destroy -auto-approve

resource "azurerm_resource_group" "example" {
  name     = "example-rg"
  location = "East US"
}
