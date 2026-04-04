# Architecture

## Request Flow

```
Client (curl/azlocal/Terraform/SDK)
    |
    v
HTTP :4566  or  HTTPS :4567
    |
    v
Chi Router (middleware chain)
    |
    +-- StructuredLogger (JSON logs)
    +-- PanicRecovery (returns 500 instead of crashing)
    +-- RequestID (x-ms-request-id header)
    +-- AzureHeaders (x-ms-version, correlation IDs)
    +-- APIVersionCheck (rejects ARM calls without ?api-version=)
    |
    v
Route Matching
    |
    +-- /health, /metrics  -->  Server handlers
    +-- /metadata/*        -->  Metadata service
    +-- /{tenant}/oauth2/* -->  Auth service (mock JWT)
    +-- /subscriptions/*   -->  ARM services (RG, VNet, DNS, ACR, etc.)
    +-- /blob/*            -->  Blob data plane
    +-- /keyvault/*        -->  Key Vault data plane
    +-- /cosmosdb/*        -->  Cosmos DB data plane
    +-- /servicebus/*      -->  Service Bus data plane
    +-- /appconfig/*       -->  App Configuration data plane
    |
    v
Service Handler
    |
    v
Store (Backend interface)
    |
    +-- MemoryBackend (default, in-process map)
    +-- PostgresBackend (when DATABASE_URL is set)
```

## Project Structure

```
cmd/
  miniblue/       Server binary
  azlocal/        CLI tool
  healthcheck/    Docker healthcheck binary

internal/
  server/
    server.go     Router setup, route registration
    middleware.go  Azure headers, API version check
    logging.go     Structured JSON logger, metrics
  store/
    backend.go    Backend interface
    memory.go     In-memory implementation
    postgres.go   PostgreSQL implementation
    factory.go    Backend selection from env
  azerr/
    errors.go     Azure-compatible error responses
  services/
    acr/          Container Registry (ARM + Docker v2)
    aci/          Container Instances (ARM + real Docker)
    appconfig/    App Configuration (data plane)
    auth/         OAuth2/OIDC (mock JWT tokens)
    blob/         Blob Storage (data plane)
    cosmosdb/     Cosmos DB (data plane)
    dbmysql/      Database for MySQL (ARM stub)
    dbpostgres/   Database for PostgreSQL (ARM + real Postgres)
    dns/          DNS Zones (ARM)
    eventgrid/    Event Grid (ARM + data plane)
    functions/    Azure Functions (ARM stub)
    identity/     Managed Identity (IMDS)
    keyvault/     Key Vault (data plane)
    metadata/     Cloud metadata endpoint
    network/      Virtual Networks + Subnets (ARM)
    queue/        Queue Storage (data plane)
    redis/        Azure Cache for Redis (ARM + real Redis)
    resourcegroups/ Resource Groups (ARM)
    servicebus/   Service Bus (data plane)
    sqldb/        Azure SQL Database (ARM stub)
    subscriptions/ Subscriptions, Tenants, Providers (ARM)
    table/        Table Storage (data plane)
```

## Adding a New Service

1. Create `internal/services/yourservice/handler.go`
2. Implement `NewHandler(s *store.Store) *Handler` and `Register(r chi.Router)`
3. Register in `internal/server/server.go` (import + call in `setupRoutes`)
4. Add to health endpoint service list
5. Add to `supportedProviders` in subscriptions handler (if ARM)
6. Write tests in `tests/yourservice_test.go`

## Storage

The `Store` wraps a `Backend` interface. All handlers use `*store.Store` without knowing which backend is active.

- **Set/Get/Delete**: Key-value operations
- **ListByPrefix**: Used for listing resources within a scope (e.g., all VNets in a resource group)
- **DeleteByPrefix**: Cascade deletes (e.g., deleting a resource group removes all child resources)

Keys follow the pattern: `service:subscriptionId:resourceGroupName:resourceName`

## Auth

miniblue returns mock JWT tokens with valid structure (`header.payload.signature`). The payload includes `oid`, `tid`, `appid` claims that Terraform and Azure SDKs parse. No real token validation is performed.
