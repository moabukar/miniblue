# API Parity Matrix

What's implemented, stubbed, and unsupported in miniblue compared to real Azure APIs.

## Legend

- **Full**: API works and returns realistic responses
- **Stub**: API accepts requests and returns valid responses but no real backend processing
- **None**: API not implemented (returns 404)

## ARM Management APIs

| Service | Create | Get | List | Delete | Update | Notes |
|---------|--------|-----|------|--------|--------|-------|
| Resource Groups | Full | Full | Full | Full | Full (PATCH) | HEAD for existence check |
| Virtual Networks | Full | Full | Full | Full | - | Subnets included |
| Subnets | Full | Full | Full | Full | - | Cascade delete with VNet |
| DNS Zones | Full | Full | Full | Full | - | Auto-creates SOA/NS records |
| DNS Records | Full | Full | - | Full | - | A, CNAME, MX, TXT, SOA, NS |
| Container Registry | Full | Full | Full | Full | - | Name availability check, replications |
| Event Grid Topics | Full | Full | Full | Full | - | Event publishing on data plane |
| Azure Functions | Stub | Stub | Stub | Stub | - | No function execution |
| DB for PostgreSQL | Full | Full | Full | Full | - | Real DB creation via POSTGRES_URL |
| DB for MySQL | Stub | Stub | Stub | Stub | - | ARM only, no real MySQL |
| Azure SQL Database | Stub | Stub | Stub | Stub | - | ARM only, no real SQL Server |
| Azure Cache for Redis | Full | Full | Full | Full | - | Real connectivity check via REDIS_URL |
| Container Instances | Full | Full | Full | Full | - | Real Docker containers when available |
| Public IP Addresses | Full | Full | Full | Full | - | Static/dynamic allocation, auto-generated IPs |
| Network Security Groups | Full | Full | Full | Full | - | Security rules, default rules, cascade delete |
| Load Balancer | Full | Full | Full | Full | - | Frontends, backends, rules, probes, NAT |
| Application Gateway | Full | Full | Full | Full | - | SKU, frontends, backends, listeners, rules, probes, WAF |
| Subscriptions | Full | Full | - | - | - | Mock subscription |
| Tenants | - | - | Full | - | - | Mock tenant |
| Providers | Full | Full | Full | - | - | Registration always succeeds |

## Data Plane APIs

| Service | Operations | Status | Notes |
|---------|-----------|--------|-------|
| Blob Storage | Create container, upload, download, list, delete | Full | Content-length tracking |
| Table Storage | Create table, upsert, get, query, delete entity | Full | Partition/row key support |
| Queue Storage | Create queue, send, receive, clear, delete | Full | Dequeue count tracking |
| Key Vault | Set, get, list, delete secrets | Full | Keys and certificates not supported |
| Cosmos DB | Create, get, replace, delete, list documents | Full | SQL API only, no query language |
| Service Bus | Create queue/topic, send, receive, delete | Full | No subscriptions, no dead-letter |
| App Configuration | Set, get, list, delete key-values | Full | No labels, no feature flags |
| Event Grid | Publish events | Full | No event subscriptions |
| Container Registry | List manifests, tags | Stub | Docker v2 API stubs only |

## Auth and Infrastructure

| Endpoint | Status | Notes |
|----------|--------|-------|
| OAuth2 token (v1 + v2) | Full | Returns valid JWT with oid/tid claims |
| OpenID configuration | Full | OIDC discovery for MSAL |
| Instance discovery | Full | Authority validation (internal) |
| Managed Identity (IMDS) | Full | Token endpoint for workload identity |
| Cloud metadata | Full | Terraform provider metadata |
| /health | Full | 25 services listed |
| /metrics | Full | Uptime, request count, error rate |

## Not Implemented

These Azure services have no emulation:

- Azure Kubernetes Service (AKS)
- Azure App Service
- Azure Storage Accounts (ARM management)
- Azure Monitor / Log Analytics
- Azure Active Directory (beyond mock tokens)
- Azure Policy
- Azure RBAC
- Azure Key Vault (keys, certificates)
- Azure Cosmos DB (Mongo, Cassandra, Gremlin APIs)
- Azure Service Bus (subscriptions, dead-letter)
