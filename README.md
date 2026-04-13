<p align="center">
  <img src="assets/logo.svg" alt="miniblue" width="120" />
</p>

<h1 align="center">miniblue</h1>

<p align="center"><strong>The free, open-source Azure emulator. Develop and test your Azure apps locally.</strong></p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.26+-blue.svg" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://github.com/moabukar/miniblue/actions/workflows/ci.yml"><img src="https://github.com/moabukar/miniblue/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://hub.docker.com/r/moabukar/miniblue"><img src="https://img.shields.io/badge/docker-moabukar%2Fminiblue-blue?logo=docker" alt="Docker"></a>
  <a href="https://miniblue.io"><img src="https://img.shields.io/badge/docs-miniblue.io-blue" alt="Docs"></a>
</p>

<p align="center">Local Azure development. One binary. No account needed.</p>

> **Disclaimer:** miniblue is an independent open-source project. It is not affiliated with, endorsed by, or associated with Microsoft or Azure.

---

## Quick Start

```bash
# Homebrew
brew tap moabukar/tap && brew install miniblue

# Or Docker
docker run -p 4566:4566 -p 4567:4567 moabukar/miniblue:latest
```

Then interact with it:

```bash
azlocal health
azlocal group create --name myRG --location eastus
azlocal keyvault secret set --vault myvault --name db-pass --value secret123
azlocal storage blob upload --account myacct --container data --name file.txt --data "Hello!"
```

No Azure account or credentials needed.

## What it does

25 Azure services emulated behind a single port. Works with Terraform, Pulumi, Azure SDKs, and curl.

| Service | Service | Service |
|---------|---------|---------|
| Resource Groups | Blob Storage | Table Storage |
| Queue Storage | Key Vault | Cosmos DB |
| Service Bus | Azure Functions | Virtual Networks |
| DNS Zones | Container Registry | Event Grid |
| App Configuration | Managed Identity | DB for PostgreSQL |
| DB for MySQL | Azure SQL Database | Azure Cache for Redis |
| Container Instances | Public IP Addresses | Network Security Groups |
| Load Balancer | Application Gateway | Subscriptions |

## How it compares

| | LocalStack (AWS) | MiniStack (AWS) | Azurite (Azure) | miniblue |
|---|---|---|---|---|
| Services | 80+ | 36 | 3 (storage only) | **25** |
| Docker image | ~1GB | ~200MB | ~300MB | **~8MB** |
| Startup | ~10s | ~5s | ~3s | **<1s** |
| Real backends | DynamoDB Local | RDS, S3, SQS | No | **Postgres, Redis, Docker** |
| Terraform | Yes | Yes | No | **Yes** |
| License | BSL | MIT | MIT | MIT |

## Terraform

```bash
bash scripts/trust-cert.sh  # one-time cert trust
```

```hcl
provider "azurerm" {
  features {}
  metadata_host              = "localhost:4567"
  skip_provider_registration = true
  subscription_id            = "00000000-0000-0000-0000-000000000000"
  tenant_id                  = "00000000-0000-0000-0000-000000000001"
  client_id                  = "miniblue"
  client_secret              = "miniblue"
}
```

See [examples/terraform/](examples/terraform/) for full examples including three-tier, serverless, and microservices scenarios.

## Real Backends (optional)

| Feature | Env var |
|---------|---------|
| Real PostgreSQL databases | `POSTGRES_URL=postgres://user:pass@host:5432/db` |
| Real Redis connectivity | `REDIS_URL=redis://host:6379` |
| Real Docker containers (ACI) | Docker daemon running |
| File persistence | `PERSISTENCE=1` |
| Postgres persistence | `DATABASE_URL=postgres://...` |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `4566` | HTTP port |
| `TLS_PORT` | `4567` | HTTPS port |
| `LOG_LEVEL` | `info` | debug, info, warn, error |
| `SERVICES` | all | Comma-separated list to enable selectively |
| `PERSISTENCE` | off | Set to `1` for file-based persistence |

Full configuration reference in the [docs](https://miniblue.io/getting-started/configuration/).

## Documentation

Full docs at **[miniblue.io](https://miniblue.io)** covering:

- [Installation](https://miniblue.io/getting-started/installation/) (Homebrew, Docker, binary, Helm)
- [Quick Start](https://miniblue.io/getting-started/quickstart/)
- [Terraform Guide](https://miniblue.io/guides/terraform/)
- [Pulumi Guide](https://miniblue.io/guides/pulumi/)
- [SDK Guide](https://miniblue.io/guides/sdk/) (Python, Go, JavaScript)
- [CLI Reference](https://miniblue.io/guides/azure-cli/)
- [API Parity Matrix](https://miniblue.io/services/parity/)
- [Architecture](https://miniblue.io/architecture/overview/)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Each service is its own Go package under `internal/services/`.

## License

MIT - see [LICENSE](LICENSE).
