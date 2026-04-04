# Terraform Example

This example shows how to use local-azure with Terraform.

## Prerequisites

1. local-azure running: `./bin/local-azure`
2. Terraform installed

## Usage

```bash
terraform init
terraform apply -auto-approve
terraform destroy -auto-approve
```

## Notes

- Uses mock credentials (no real Azure auth needed)
- `skip_provider_registration = true` avoids provider namespace registration calls
- `use_cli = false` and `use_msi = false` prevent Terraform from trying real Azure auth
