# Terraform + miniblue

Use miniblue as a local Azure emulator with Terraform.

## Quick Start

```bash
# 1. Start miniblue
./bin/miniblue

# 2. Trust the self-signed certificate (one-time)
#    Option A: System-wide (recommended)
bash scripts/trust-cert.sh

#    Option B: Session only
export SSL_CERT_FILE=~/.miniblue/cert.pem

# 3. Run Terraform
cd examples/terraform
terraform init
terraform plan
terraform apply -auto-approve
```

## How it works

The `azurerm` provider has a `metadata_host` argument. When set, it fetches
`/metadata/endpoints?api-version=2022-09-01` over HTTPS and uses the returned
`resourceManagerEndpoint` for all ARM API calls.

miniblue serves this on port 4567 (HTTPS with a self-signed cert) and
returns `http://localhost:4566` as the resource manager endpoint.

## Certificate Trust

The `azurerm` provider always uses HTTPS for `metadata_host`. miniblue
generates a self-signed certificate on first run and saves it to
`~/.miniblue/cert.pem`.

You must trust this certificate. Options:

| Method | Command | Scope |
|--------|---------|-------|
| Script | `bash scripts/trust-cert.sh` | System-wide (persistent) |
| macOS | `sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/.miniblue/cert.pem` | System-wide |
| Linux | `sudo cp ~/.miniblue/cert.pem /usr/local/share/ca-certificates/miniblue.crt && sudo update-ca-certificates` | System-wide |
| Env var | `export SSL_CERT_FILE=~/.miniblue/cert.pem` | Current session only |

## Authentication

miniblue accepts any credentials. The mock values in `main.tf` are
placeholders - no real Azure auth needed.

## Notes

- `metadata_host` takes hostname:port only (no https:// prefix)
- The cert is reused across restarts (regenerated if expiring within 24h)
- `skip_provider_registration = true` prevents namespace registration calls
- `LOCAL_AZURE_CERT_DIR` env var overrides cert storage location
