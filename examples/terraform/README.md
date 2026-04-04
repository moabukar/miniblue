# Terraform + local-azure

This example shows how to use local-azure as a local Azure emulator with Terraform.

## How it works

The `azurerm` provider has a `metadata_host` argument. When set, it fetches
`/metadata/endpoints?api-version=2022-09-01` from that host and uses the
returned `resourceManagerEndpoint` for all ARM API calls.

local-azure serves this endpoint and returns `http://localhost:4566` as the
resource manager endpoint, so all Terraform calls route to our emulator.

## Prerequisites

1. local-azure running:
   ```bash
   ./bin/local-azure
   ```
2. Terraform installed

## Usage

```bash
cd examples/terraform
terraform init
terraform plan
terraform apply -auto-approve
terraform destroy -auto-approve
```

## Authentication

local-azure accepts any credentials. The mock values in `main.tf` are
placeholders - no real Azure auth needed.

You can also set environment variables instead:
```bash
export ARM_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"
export ARM_TENANT_ID="00000000-0000-0000-0000-000000000001"
export ARM_CLIENT_ID="local-azure"
export ARM_CLIENT_SECRET="local-azure"
```

## Notes

- `skip_provider_registration = true` prevents Terraform from trying to
  register resource provider namespaces (Microsoft.Compute etc.)
- `metadata_host` takes hostname:port only (no http:// prefix, no path)
- This approach works because the `azurerm` provider dynamically discovers
  all API endpoints from the metadata service - same mechanism used for
  Azure Stack and sovereign clouds
