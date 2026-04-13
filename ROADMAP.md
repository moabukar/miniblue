# Roadmap

This is the public roadmap for miniblue. It outlines what we are working on and what is planned.

Priorities shift based on community feedback. Open an issue or discussion if something here matters to you.

## Now (in progress)

- **Improve Terraform parity** - fill gaps in existing service responses so more `azurerm` resources work out of the box
- **Azure SDK compatibility** - test and fix responses for the official Azure SDKs (Go, Python, .NET, JS)
- **Better error messages** - return the exact error codes and formats that Azure returns so client libraries handle errors correctly

## Next (planned)

### New services

| Service | Provider | Priority | Notes |
|---------|----------|----------|-------|
| Azure Kubernetes Service (AKS) | `Microsoft.ContainerService` | High | ARM management only, no real cluster |
| App Service / Web Apps | `Microsoft.Web` | High | Extends existing Functions stub |
| Azure Monitor | `Microsoft.Insights` | Medium | Metrics and diagnostic settings |
| Traffic Manager | `Microsoft.Network` | Medium | DNS-based routing profiles |
| Azure Front Door | `Microsoft.Cdn` | Medium | CDN and WAF policies |
| Private DNS Zones | `Microsoft.Network` | Medium | Extends existing DNS service |
| Network Interfaces (NIC) | `Microsoft.Network` | Medium | Required for VM provisioning |
| Managed Disks | `Microsoft.Compute` | Low | Disk resources for VMs |
| Virtual Machines | `Microsoft.Compute` | Low | ARM management only |
| Azure Policy | `Microsoft.Authorization` | Low | Policy definitions and assignments |

### Platform improvements

- **Persistent state across restarts** - improve file and PostgreSQL backends for production-like local environments
- **Webhooks and event delivery** - Event Grid subscriptions that call real HTTP endpoints
- **Service Bus subscriptions and dead-letter** - complete the messaging story
- **Key Vault keys and certificates** - extend beyond secrets
- **Cosmos DB query language** - basic SQL query support
- **Multi-subscription support** - allow creating and switching between subscriptions
- **ARM template deployment** - accept and process ARM template JSON

## Later (exploratory)

- Azure Active Directory / Entra ID (beyond mock tokens)
- Azure RBAC and role assignments
- Cosmos DB Mongo and Cassandra APIs
- Azure Batch
- Azure SignalR
- Logic Apps

## Non-goals

These are explicitly out of scope:

- **Full Azure API compatibility** - miniblue aims for "good enough" responses that satisfy Terraform, SDKs and common workflows. Not 1:1 API parity
- **Production workloads** - miniblue is for local development and CI/CD testing
- **Authentication enforcement** - miniblue accepts any credentials by design
- **Multi-region replication** - all data is local, regions are cosmetic

## How to contribute

Pick anything from the roadmap and open a PR. The [CONTRIBUTING.md](CONTRIBUTING.md) has the full guide. Each service follows the same handler pattern. Look at any file in `internal/services/` for a template.

If you want to work on something, open an issue first so we can coordinate and avoid duplicate work.
