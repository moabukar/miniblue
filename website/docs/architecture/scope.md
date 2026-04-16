# Scope and Philosophy

miniblue sits between unit tests and full integration tests. It gives you a local Azure environment that is good enough for Terraform plans, SDK smoke tests and CI pipelines without needing a real Azure subscription.

## Control plane vs data plane

Azure APIs have two layers:

- **Control plane (ARM)** - resource management via `/subscriptions/{sub}/resourceGroups/{rg}/providers/...`. This is what Terraform, Bicep, Azure CLI and the Azure Portal use to create, read, update and delete resources.
- **Data plane** - direct operations on the resource itself. Uploading a blob, sending a Service Bus message, reading a Key Vault secret.

miniblue focuses on the **control plane first**. Every service starts with ARM CRUD (PUT, GET, DELETE, List) so that Terraform and infrastructure-as-code tools work. Data plane support is added for high-value developer workflows where calling a real Azure service during local development is impractical.

### What this means in practice

| Use case | miniblue support | Notes |
|----------|-----------------|-------|
| `terraform plan` and `terraform apply` | Primary focus | ARM responses match what the azurerm provider expects |
| `az group create`, `az network vnet create` | Works | ARM endpoints |
| Upload a blob, send a queue message | Works | Data plane for common dev workflows |
| Read a Key Vault secret from your app | Works | Data plane |
| Complex Cosmos DB queries with SQL syntax | Not yet | Document CRUD works, query language is on the roadmap |
| Service Bus topic subscriptions | Not yet | Basic queue send/receive works |
| Run Azure Functions locally | No | Use Azure Functions Core Tools for that |
| Azure AD/Entra ID authentication flows | Mock only | Returns valid JWT tokens but no real identity management |

## When to use miniblue

- **Local development** - run `terraform apply` against miniblue instead of a dev subscription
- **CI/CD pipelines** - spin up miniblue as a Docker container in your pipeline to test infrastructure code
- **SDK smoke tests** - point your SDK client at `localhost:4566` to test CRUD operations
- **Demos and workshops** - no Azure account needed

## When to use real Azure

- **Data plane edge cases** - if your app relies on specific Azure behavior (retry headers, throttling, eventual consistency), test against real Azure
- **Performance testing** - miniblue is fast but does not simulate real Azure latency or rate limits
- **Security testing** - miniblue accepts any credentials by design
- **Production deployments** - miniblue is for development only

## Design principles

1. **Start with ARM, add data plane when it matters.** A service that only has ARM CRUD is still useful for Terraform users.
2. **Good enough beats perfect.** Responses should satisfy Terraform and SDKs, not be byte-for-byte identical to Azure.
3. **Zero config by default.** No accounts, no credentials, no setup. `docker run` and go.
4. **One binary, one port.** No separate emulators for each service.
5. **Tests over mocks.** miniblue is not a mock library. It is a running server that your tools talk to over HTTP.
