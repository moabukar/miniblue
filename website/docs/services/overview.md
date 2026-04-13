# Services Overview

miniblue emulates 25 Azure services on a single port. All services use in-memory storage and require no authentication.

## Service status

| Service | Azure Provider | Status | ARM API | Data Plane |
|---------|---------------|--------|---------|------------|
| [Resource Groups](resource-groups.md) | `Microsoft.Resources` | Done | Yes | -- |
| [Blob Storage](blob-storage.md) | `Microsoft.Storage` | Done | -- | Yes |
| Table Storage | `Microsoft.Storage` | Done | -- | Yes |
| Queue Storage | `Microsoft.Storage` | Done | -- | Yes |
| [Key Vault](key-vault.md) | `Microsoft.KeyVault` | Done | -- | Yes |
| [Cosmos DB](cosmos-db.md) | `Microsoft.DocumentDB` | Done | -- | Yes |
| [Service Bus](service-bus.md) | `Microsoft.ServiceBus` | Done | -- | Yes |
| Azure Functions | `Microsoft.Web` | Done | Yes | -- |
| [Virtual Networks](virtual-networks.md) | `Microsoft.Network` | Done | Yes | -- |
| [DNS Zones](dns-zones.md) | `Microsoft.Network` | Done | Yes | -- |
| [Container Registry](container-registry.md) | `Microsoft.ContainerRegistry` | Done | Yes | Yes |
| Event Grid | `Microsoft.EventGrid` | Done | Yes | Yes |
| App Configuration | `Microsoft.AppConfiguration` | Done | -- | Yes |
| Managed Identity | IMDS | Done | -- | Yes |
| [DB for PostgreSQL](database-postgresql.md) | `Microsoft.DBforPostgreSQL` | Done | Yes | -- |
| [DB for MySQL](database-mysql.md) | `Microsoft.DBforMySQL` | Done | Yes | -- |
| [Azure SQL Database](database-sql.md) | `Microsoft.Sql` | Done | Yes | -- |
| [Azure Cache for Redis](redis.md) | `Microsoft.Cache` | Done | Yes | -- |
| [Container Instances](container-instances.md) | `Microsoft.ContainerInstance` | Done | Yes | -- |
| Public IP Addresses | `Microsoft.Network` | Done | Yes | -- |
| Network Security Groups | `Microsoft.Network` | Done | Yes | -- |
| Load Balancer | `Microsoft.Network` | Done | Yes | -- |
| Application Gateway | `Microsoft.Network` | Done | Yes | -- |

### What "ARM API" and "Data Plane" mean

- **ARM API** -- the service responds to Azure Resource Manager style URLs (`/subscriptions/{sub}/resourceGroups/{rg}/providers/...`). This is what Terraform and the Azure CLI use.
- **Data Plane** -- the service responds to simplified REST URLs for direct data operations (e.g. `/blob/{account}/{container}/{blob}`). This is what the azlocal CLI and curl use.

## Terraform compatibility

The following resources work with `hashicorp/azurerm` provider v3.x:

| Terraform Resource | miniblue Service |
|--------------------|-----------------|
| `azurerm_resource_group` | Resource Groups |
| `azurerm_virtual_network` | Virtual Networks |
| `azurerm_subnet` | Virtual Networks |
| `azurerm_dns_zone` | DNS Zones |
| `azurerm_container_registry` | Container Registry |
| `azurerm_public_ip` | Public IP Addresses |
| `azurerm_network_security_group` | Network Security Groups |
| `azurerm_network_security_rule` | Network Security Groups |
| `azurerm_lb` | Load Balancer |
| `azurerm_lb_backend_address_pool` | Load Balancer |
| `azurerm_lb_probe` | Load Balancer |
| `azurerm_lb_rule` | Load Balancer |
| `azurerm_application_gateway` | Application Gateway |

See the [Terraform guide](../guides/terraform.md) for a full working example.

## API patterns

All ARM-style endpoints follow the Azure REST API convention:

```
PUT /subscriptions/{subscriptionId}/resourceGroups/{rg}/providers/{provider}/{resourceType}/{name}
```

All data plane endpoints use simplified paths:

```
PUT /blob/{account}/{container}/{blob}
PUT /keyvault/{vault}/secrets/{name}
POST /cosmosdb/{account}/dbs/{db}/colls/{coll}/docs
```

## Common query parameters

| Parameter | Notes |
|-----------|-------|
| `api-version` | Accepted but not enforced. miniblue responds the same regardless of version. |

## Error format

Errors follow the Azure error response format:

```json
{
  "error": {
    "code": "ResourceNotFound",
    "message": "Resource 'myRG' of type 'Microsoft.Resources/resourceGroups' was not found."
  }
}
```

HTTP status codes used:

| Code | Meaning |
|------|---------|
| `200` | OK (update, get) |
| `201` | Created |
| `202` | Accepted (async delete) |
| `204` | No Content |
| `400` | Bad Request |
| `404` | Not Found |
| `409` | Conflict |
