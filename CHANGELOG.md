# Changelog

All notable changes to miniblue are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/). Versions follow [Semantic Versioning](https://semver.org/).

## [0.2.0] - 2026-04-04

### Added
- Azure Database for PostgreSQL (real DB creation via `POSTGRES_URL`)
- Azure Database for MySQL (ARM management API)
- Azure SQL Database (ARM management API)
- Azure Cache for Redis (real connectivity via `REDIS_URL`)
- Azure Container Instances (real Docker containers when available)
- Structured JSON logging (method, path, status, latency)
- `/metrics` endpoint (uptime, request count, error rate)
- PostgreSQL persistent storage backend via `DATABASE_URL`
- Per-service test files (37 tests across 16 files)
- Pre-commit hooks for code quality
- Terraform scenario examples (three-tier, serverless, microservices)
- CHANGELOG.md
- CODEOWNERS
- RELEASING.md with step-by-step release guide
- mkdocs documentation site (17 pages)

### Changed
- Go version bumped from 1.23 to 1.24
- Dockerfile now runs as non-root user (`miniblue`)
- Dockerfile includes HEALTHCHECK
- golangci-lint added to CI pipeline
- Service count: 14 to 19

### Fixed
- Terraform destroy flow (async polling + list resources endpoint)
- VNet/Subnet/ACR responses include all fields the azurerm provider expects
- DNS zones auto-create SOA and NS records
- Metadata trailing slashes removed (prevented double-slash 404s)
- All ARM curl examples include `?api-version=` parameter

## [0.1.0] - 2026-04-04

### Added
- Initial release with 14 Azure services
- Resource Groups, Blob Storage, Table Storage, Queue Storage
- Key Vault, Cosmos DB, Service Bus, Azure Functions
- Virtual Networks, DNS Zones, Container Registry, Event Grid
- App Configuration, Managed Identity
- Terraform azurerm provider support (5 resource types)
- `azlocal` CLI (like awslocal for LocalStack)
- Self-signed TLS certificate management
- Multi-arch Docker images (amd64 + arm64)
- GHCR and DockerHub publishing
- Mock OAuth2/OIDC authentication
- ARM API versioning middleware
- Azure-compatible error responses

[0.2.0]: https://github.com/moabukar/miniblue/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/moabukar/miniblue/releases/tag/v0.1.0
