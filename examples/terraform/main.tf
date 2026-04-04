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
#   terraform destroy -auto-approve

resource "azurerm_resource_group" "example" {
  name     = "example-rg"
  location = "East US"
}
