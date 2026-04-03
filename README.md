# local-azure

**The free, open-source Azure emulator. Develop and test your Azure apps locally.**

[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/moabukar/local-azure)](https://hub.docker.com/r/moabukar/local-azure)

Single port · No account · No license key · No telemetry · Just Azure APIs, locally.

<p align="center">
  <img src="https://img.shields.io/badge/Azure-Emulator-0078D4?style=for-the-badge&logo=microsoftazure&logoColor=white" alt="Azure Emulator"/>
</p>

## Features

- **14 Azure services** emulated on a single port (4566)
- **Drop-in compatible** with Azure SDKs, Azure CLI, Terraform, Pulumi
- **In-memory storage** by default (fast, ephemeral)
- **Docker-first** deployment
- **Zero configuration** required
- **ARM API compatible** responses

## Quick Start

### Docker Run

```bash
docker run -p 4566:4566 moabukar/local-azure:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  local-azure:
    image: moabukar/local-azure:latest
    ports:
      - "4566:4566"
```

```bash
docker-compose up
```

### Build from Source

```bash
git clone https://github.com/moabukar/local-azure.git
cd local-azure
make build
./bin/local-azure
```

## Supported Services

| Service | Status | Description |
|---------|--------|-------------|
| Resource Groups | ✅ | ARM resource group management |
| Blob Storage | ✅ | Containers, blobs, upload/download |
| Table Storage | ✅ | Entity CRUD operations |
| Queue Storage | ✅ | Send/receive/peek messages |
| Key Vault | ✅ | Secrets management |
| Cosmos DB | ✅ | Document CRUD (SQL API) |
| Service Bus | ✅ | Queues, topics, messaging |
| Azure Functions | ✅ | Function app registration (stub) |
| Virtual Networks | ✅ | VNets and subnets |
| DNS Zones | ✅ | Zone and record management |
| Container Registry | ✅ | Registry management, manifest listing |
| Event Grid | ✅ | Topics, subscriptions, event publish |
| App Configuration | ✅ | Key-value configuration store |
| Managed Identity | ✅ | Token endpoint (IMDS) |

## Usage Examples

### Azure CLI

```bash
# Configure Azure CLI to use local-azure
export AZURE_RESOURCE_MANAGER_ENDPOINT=http://localhost:4566

# Create a resource group
az group create --name myResourceGroup --location eastus

# List resource groups
az group list
```

### Terraform

```hcl
provider "azurerm" {
  features {}
  
  # Point to local-azure
  resource_manager_endpoint = "http://localhost:4566"
  skip_provider_registration = true
}

resource "azurerm_resource_group" "example" {
  name     = "example-resources"
  location = "East US"
}
```

### Go SDK

```go
import (
    "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
    "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// Override the endpoint
endpoint := "http://localhost:4566"
```

### curl

```bash
# Health check
curl http://localhost:4566/health

# Create a resource group
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourcegroups/myRG" \
  -H "Content-Type: application/json" \
  -d '{"location": "eastus"}'

# Create a blob container
curl -X PUT "http://localhost:4566/blob/myaccount/mycontainer"

# Set a Key Vault secret
curl -X PUT "http://localhost:4566/keyvault/myvault/secrets/mysecret" \
  -H "Content-Type: application/json" \
  -d '{"value": "supersecret"}'
```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `PORT` | `4566` | HTTP server port |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

## API Reference

### Health Endpoint

```
GET /health
```

Returns service status and list of available services.

### Resource Groups (ARM)

```
PUT    /subscriptions/{sub}/resourcegroups/{name}
GET    /subscriptions/{sub}/resourcegroups/{name}
DELETE /subscriptions/{sub}/resourcegroups/{name}
GET    /subscriptions/{sub}/resourcegroups
```

### Blob Storage

```
PUT    /blob/{account}/{container}
DELETE /blob/{account}/{container}
GET    /blob/{account}/{container}
PUT    /blob/{account}/{container}/{blob}
GET    /blob/{account}/{container}/{blob}
DELETE /blob/{account}/{container}/{blob}
```

### Key Vault

```
PUT    /keyvault/{vault}/secrets/{name}
GET    /keyvault/{vault}/secrets/{name}
DELETE /keyvault/{vault}/secrets/{name}
GET    /keyvault/{vault}/secrets
```

### Managed Identity (IMDS)

```
GET /metadata/identity/oauth2/token?resource=https://management.azure.com/
GET /metadata/instance
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Inspired by [LocalStack](https://github.com/localstack/localstack) and [MiniStack](https://github.com/moabukar/ministack).

---

Made with ❤️ for the Azure developer community
