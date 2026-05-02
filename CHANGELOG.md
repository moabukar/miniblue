# Changelog

All notable changes to miniblue are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/). Versions follow [Semantic Versioning](https://semver.org/).

## [0.7.0] - 2026-05-02

### Added
- **`Microsoft.Resources/deployments` endpoint Phase 1** (#118, addresses parts of #66, #74). PUT/GET/DELETE/List a deployment, walk `template.resources` in declaration order, resolve `[parameters('x')]` and `[variables('x')]`, dispatch each resource to its existing handler via the embedded chi router. Lets `az deployment group create --template-file main.bicep` work end to end against miniblue. Phase 1 does not support `[concat()]` or other template functions, copy loops, conditions, nested templates, `dependsOn` ordering, outputs evaluation, or non-RG-scope deployments
- **Helm / Kubernetes deployment guide** at `/guides/helm` (#117, closes #86) covering `helm install`, persistence, PostgreSQL backend via secret, exposing the service, and the real-backend limitations in-cluster
- **`SHA256SUMS` release asset** (#115, closes #94). The release workflow now publishes a single checksums file alongside the binaries

### Changed
- README comparison table service count bumped 26 to 27 (AKS shipped in v0.6.0)
- CHANGELOG backfilled for v0.4.2 through v0.6.0 from each release's notes (#116)

### CI
- New AKS pipeline job that exercises the `:full` Docker image with a mounted docker socket, locking in the in-container real-backend path against future Dockerfile or alpine bumps (#114)

## [0.6.0] - 2026-05-01

### Added
- **Azure Kubernetes Service (AKS) emulation** (#111, closes #50). `Microsoft.ContainerService/managedClusters` with two backends: stub (default, ARM only) and real (rancher/k3s in Docker, opt-in via `AKS_BACKEND=k3s`). `azlocal aks create/list/show/delete/get-credentials` commands. `get-credentials` merges into `~/.kube/config` like `az aks` does
- **`full` Dockerfile target** that adds the docker CLI so AKS and ACI real backends work in-container with the host docker socket mounted
- **Dedicated AKS CI pipeline** (`aks-e2e.yml`): real-backend k3s smoke (DinD on the runner) and a Terraform apply against the AKS example
- Resource group cascade delete now cleans up child AKS resources

### Changed
- `cmd/miniblue` graceful shutdown now propagates to AKS so k3s containers do not leak past miniblue's lifetime
- 27 services total (was 26)

## [0.5.1] - 2026-05-01

### Fixed
- Container startup crash on fresh Docker runs. Scratch image runs as `USER 65534` but had no writable home directory. Now ships with `/home/nonroot` owned by 65534 and `HOME` set, so the cert dir can be created cleanly (#110)
- Cert load when existing certs are unusable: missing, corrupted, or expired files now fall through to regeneration instead of `log.Fatal`. Thanks @KarasAlison (#109)
- Bump `github.com/go-sql-driver/mysql` to v1.10.0 (#108)

### Changed
- E2E now builds the image from the PR's `Dockerfile` instead of pulling `:latest` from Docker Hub, so regressions are caught on the PR that introduces them. Failed startups dump `docker logs miniblue` for diagnostics (#110)
- Bump `mislav/bump-homebrew-formula-action` to v4 (#107)

## [0.5.0] - 2026-04-21

### Added
- **App Service & Web Apps** (`Microsoft.Web/sites`) with full CRUD, slots, publishing credentials, and config sub-resources (app settings, auth settings v1/v2, connection strings, backup, logs, metadata, push settings, slot config names, storage accounts) (#103, contributed by @abusarah-tech)
- **App Service Plans** (`Microsoft.Web/serverFarms`) with SKU tier detection
- **Container Apps** (`Microsoft.App/containerApps`) with CRUD, start/stop, revisions, secrets, auth tokens, custom domain analysis, and subscription-level listing
- **Managed Environments** (`Microsoft.App/managedEnvironments`) with CRUD and app logs configuration
- **Container App Jobs** (`Microsoft.App/jobs`) with CRUD, start/stop, executions, detectors, and secrets
- `CheckNameAvailability` endpoint for sites
- New `containerapp` and `containerapp env` commands in `azlocal`
- New `containerapps` and `webapps` Terraform scenarios

### Changed
- Existing Azure Functions emulation refactored into the broader `sites` package

### Fixed
- Improved error handling when loading existing TLS certificates and keys (#105, fixes #104)

## [0.4.4] - 2026-04-16

### Added
- Persistent volume support in the Helm chart (#83). Survives pod restarts when enabled in values.yaml
- PostgreSQL backend support in the Helm chart via Kubernetes secret (`databaseUrlSecret.name` / `.key`)

### Changed
- Secure defaults set on the Helm container, matching the Dockerfile's non-root user (#88)

## [0.4.3] - 2026-04-16

### Fixed
- Store backend data race in `FileBackend.Save()`: auto-save goroutine could read inconsistent state while handlers were writing. Now uses an atomic `Snapshot()` under a single lock (#84, #89)
- `PostgresBackend.List()` and `ListByPrefix()` returned `nil` on error instead of empty slices, causing inconsistent behavior with the memory backend
- `azlocal` CLI no longer silently swallows `json.Marshal` and `io.ReadAll` errors in `doPut`/`doPost`/`printResponse`. Errors now go to stderr (#85)

### Changed
- E2E workflow now runs on every push to main and every PR (was `workflow_dispatch` only) so breaking changes can not slip through unnoticed (#82)
- CI example bumped to `actions/checkout@v6` (#91)

## [0.4.2] - 2026-04-15

### Fixed
- `miniblue --version` now reports the correct version instead of `dev`. Version is injected via ldflags in all build targets: release binaries, Docker images, and Homebrew formula

## [0.4.1] - 2026-04-15

### Fixed
- Resource tags now preserved and returned on all ARM services (VNets, NSGs, Public IPs, Load Balancers, App Gateways, DNS Zones, Storage Accounts). Fixes constant Terraform drift
- Resource group cascade delete now cleans up all 26 child resource types (was missing NSGs, Public IPs, Load Balancers, App Gateways, Storage Accounts, Cosmos DB, Service Bus, App Config)
- Subscription-level list endpoints added for VNets, NSGs, Public IPs, Load Balancers, App Gateways and DNS Zones. Fixes Terraform data source 404s
- Renamed `_arm_test.go` files that were silently skipped on amd64/arm64 (34 tests now run)
- Added 3 missing providers to registration list (PostgreSQL, Redis, Container Instances)
- Added 10 missing service documentation pages to website

### Changed
- PR template updated with all 26 services
- Added @abusarah-tech to CODEOWNERS
- Added public ROADMAP.md

## [0.3.0] - 2026-04-14

### Added
- Public IP Addresses service (`Microsoft.Network/publicIPAddresses`) with full CRUD, static/dynamic allocation
- Network Security Groups service (`Microsoft.Network/networkSecurityGroups`) with security rules and cascade delete
- Azure Load Balancer service (`Microsoft.Network/loadBalancers`) with frontend IPs, backend pools, rules and probes
- Application Gateway service (`Microsoft.Network/applicationGateways`) with full L7 config
- Azure Storage Accounts ARM service with shared key auth (contributed by @abusarah-tech)
- Terraform examples for load balancer and application gateway
- Terraform example for storage accounts
- .NET example using HttpClient (contributed by @sa-es-ir)
- E2E test suite for storage accounts
- `azlocal` CLI commands for storage account operations

### Changed
- Service count: 21 to 26
- Blob handler refactored to work under storage accounts

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
- Go version bumped to 1.25
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

[0.4.1]: https://github.com/moabukar/miniblue/compare/v0.4.0...v0.4.1
[0.3.0]: https://github.com/moabukar/miniblue/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/moabukar/miniblue/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/moabukar/miniblue/releases/tag/v0.1.0
