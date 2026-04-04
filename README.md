# miniblue

**The free, open-source Azure emulator. Develop and test your Azure apps locally.**

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/moabukar/miniblue/actions/workflows/ci.yml/badge.svg)](https://github.com/moabukar/miniblue/actions/workflows/ci.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/moabukar/miniblue)](https://hub.docker.com/r/moabukar/miniblue)
[![Docs](https://img.shields.io/badge/docs-moabukar.github.io%2Fminiblue-blue)](https://moabukar.github.io/miniblue)

Local Azure development. One binary. No account needed.

---

## Why miniblue?

**AWS has LocalStack and MiniStack. Azure has... nothing unified.**

The Azure ecosystem today forces developers to cobble together individual emulators:

| Tool | Services covered | Maintained by |
|------|-----------------|---------------|
| Azurite | Blob, Queue, Table storage only | Microsoft |
| Cosmos DB Emulator | Cosmos DB only | Microsoft |
| Azure Functions Core Tools | Functions only | Microsoft |
| Service Bus Emulator | Service Bus only | Microsoft |
| App Configuration Emulator | App Config only | Microsoft |

That is 5 separate tools, 5 different Docker images, 5 different ports and configs - just to get basic local dev working. And you still have no local emulation for Resource Groups, Key Vault, DNS, VNets, Event Grid, ACR, or Managed Identity.

**miniblue gives you 14+ Azure services on a single port.** One binary, one Docker image, zero config.

### How it compares

| | LocalStack (AWS) | MiniStack (AWS) | Azurite (Azure) | miniblue |
|---|---|---|---|---|
| Services | 80+ | 36 | 3 (storage only) | 14+ |
| Single port | 4566 | 4566 | 10000-10002 | 4566 |
| Language | Python | Python | Node.js | Go |
| Auth required | No (free tier) | No | No | No |
| Docker image | ~1GB | ~200MB | ~300MB | ~15MB |
| CLI wrapper | awslocal | awslocal | N/A | azlocal |
| License | BSL (was Apache) | MIT | MIT | MIT |
| ARM API support | N/A (AWS) | N/A (AWS) | No | Yes |

### Why has no one built this before?

1. **Microsoft ships individual emulators** - so the pain is spread across tools rather than being a single obvious gap
2. **Azure's API surface is huge** - ARM (resource management) + data plane APIs per service means a lot of surface area
3. **MSAL auth is complex** - Azure CLI requires HTTPS + Microsoft identity validation, making local dev harder than `aws --endpoint-url`
4. **LocalStack had first-mover advantage** - AWS developers hit the "I need local dev" wall first and built solutions
5. **GCP has the same gap** - Google also only ships per-service emulators (Spanner, Pub/Sub, Firestore, etc.) with no unified tool

miniblue fills this gap for Azure developers.

---

## What miniblue is

- **19 Azure services** emulated behind a single port (4566)
- **Drop-in compatible** with Azure SDKs, Terraform, Pulumi
- **In-memory storage** by default (fast, ephemeral)
- **Docker-first** deployment
- **Zero configuration** required
- **ARM API compatible** responses
- **azlocal CLI** included (like awslocal for LocalStack)

## What miniblue is NOT

miniblue is an **API emulator** for local development and testing. It does not run real Azure infrastructure.

- **No VMs** - Azure VMs, VMSS, and compute are not emulated
- **No real networking** - VNets/subnets are API stubs, no actual network layer
- **No real containers** - AKS/ACI endpoints are not supported (yet)
- **No data persistence** - all state is in-memory and lost on restart
- **Not production-ready** - never expose miniblue to the internet

miniblue is designed for **CI pipelines, integration tests, and local development** where you need Azure API compatibility without an Azure account.

## Quick Start

### Docker Run

```bash
# From Docker Hub
docker run -p 4566:4566 -p 4567:4567 moabukar/miniblue:latest

# From GitHub Container Registry
docker run -p 4566:4566 -p 4567:4567 ghcr.io/moabukar/miniblue:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  miniblue:
    image: moabukar/miniblue:latest
    ports:
      - "4566:4566"
      - "4567:4567"
```

```bash
docker-compose up
```

### Build from Source

```bash
git clone https://github.com/moabukar/miniblue.git
cd miniblue
make build
./bin/miniblue
```

## azlocal CLI

Just like `awslocal` for LocalStack, `azlocal` wraps HTTP calls to miniblue. No auth needed.

```bash
# Build
make build

# Install globally (optional)
sudo make install

# Usage
azlocal health
azlocal group create --name myRG --location eastus
azlocal group list
azlocal keyvault secret set --vault myvault --name db-pass --value secret123
azlocal storage container create --account myaccount --name mycontainer
azlocal storage blob upload --account myaccount --container mycontainer --name file.txt --data "Hello!"
```

## Supported Services

| Service | Description | Real Backend |
|---------|-------------|-------------|
| Resource Groups | ARM resource group management | In-memory / Postgres |
| Blob Storage | Containers, blobs, upload/download | In-memory |
| Table Storage | Entity CRUD operations | In-memory |
| Queue Storage | Send/receive/peek messages | In-memory |
| Key Vault | Secrets management | In-memory |
| Cosmos DB | Document CRUD (SQL API) | In-memory / Postgres |
| Service Bus | Queues, topics, messaging | In-memory |
| Azure Functions | Function app registration | Stub |
| Virtual Networks | VNets and subnets | In-memory |
| DNS Zones | Zone and record management | In-memory |
| Container Registry | Registry management, manifests | In-memory |
| Event Grid | Topics and event publishing | In-memory |
| App Configuration | Key-value configuration store | In-memory |
| Managed Identity | IMDS token endpoint | Mock tokens |
| DB for PostgreSQL | Flexible server + database CRUD | Real Postgres via `POSTGRES_URL` |
| DB for MySQL | Flexible server + database CRUD | Stub (real backend planned) |
| Azure SQL Database | Server + database CRUD | Stub (real backend planned) |
| Azure Cache for Redis | Cache management + key listing | Real Redis via `REDIS_URL` |
| Container Instances | Container group lifecycle | Real Docker containers |

## Usage Examples

### curl

```bash
# Health check
curl http://localhost:4566/health

# Create a resource group
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourcegroups/myRG?api-version=2020-06-01" \
  -H "Content-Type: application/json" \
  -d '{"location": "eastus"}'

# Set a Key Vault secret
curl -X PUT "http://localhost:4566/keyvault/myvault/secrets/mysecret" \
  -H "Content-Type: application/json" \
  -d '{"value": "supersecret"}'

# Upload a blob
curl -X PUT "http://localhost:4566/blob/myaccount/mycontainer"
curl -X PUT "http://localhost:4566/blob/myaccount/mycontainer/hello.txt" \
  -H "Content-Type: text/plain" \
  -d "Hello from miniblue!"
curl "http://localhost:4566/blob/myaccount/mycontainer/hello.txt"
```

### Terraform

```bash
# Trust the self-signed cert (one-time)
bash scripts/trust-cert.sh
# Or for current session only:
export SSL_CERT_FILE=~/.miniblue/cert.pem
```

```hcl
provider "azurerm" {
  features {}

  metadata_host              = "localhost:4567"
  skip_provider_registration = true

  subscription_id = "00000000-0000-0000-0000-000000000000"
  tenant_id       = "00000000-0000-0000-0000-000000000001"
  client_id       = "miniblue"
  client_secret   = "miniblue"
}

resource "azurerm_resource_group" "example" {
  name     = "example-rg"
  location = "East US"
}

resource "azurerm_virtual_network" "example" {
  name                = "example-vnet"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
}
```

See [examples/terraform/](examples/terraform/) for a full working example.

### Go SDK

```go
// Override the endpoint
endpoint := "http://localhost:4566"
```

## az CLI Setup

The az CLI uses MSAL for authentication which requires HTTPS and validates authority endpoints against Microsoft's servers. To use az CLI with miniblue, use the helper script:

```bash
./scripts/az-login-local.sh

az group create --name myRG --location eastus
az group list

# Switch back to real Azure when done
az cloud set --name AzureCloud
az login
```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `PORT` | `4566` | HTTP server port |
| `TLS_PORT` | `4567` | HTTPS server port (self-signed cert) |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `DATABASE_URL` | (none) | PostgreSQL backend for persistent storage |
| `POSTGRES_URL` | (none) | Real PostgreSQL for DB for PostgreSQL service |
| `REDIS_URL` | (none) | Real Redis for Azure Cache for Redis service |
| `LOCAL_AZURE_ENDPOINT` | `http://localhost:4566` | azlocal CLI endpoint override |
| `LOCAL_AZURE_CERT_DIR` | `~/.miniblue` | Directory for TLS certificate storage |

## Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 4566 | HTTP | Resource Manager API, SDKs, curl, Terraform |
| 4567 | HTTPS | Auth endpoints (self-signed cert for MSAL) |

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

Adding a new service is straightforward - each service is its own Go package under `internal/services/`. See the contributing guide for the pattern.

## Roadmap

- [ ] Persistent storage (file-backed)
- [ ] Azure SDK wire-compatibility improvements
- [ ] More services (Redis Cache, App Service, AKS, etc.)
- [ ] Terraform provider integration tests
- [ ] Web UI for visualising resources
- [ ] Pulumi integration

## License

MIT - see [LICENSE](LICENSE).
