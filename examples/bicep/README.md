# Bicep / ARM template example

Deploy a Bicep file against miniblue end to end.

## Prerequisites

```bash
./bin/miniblue
export SSL_CERT_FILE=~/.miniblue/cert.pem  # or run scripts/trust-cert.sh once
```

## Apply

```bash
azlocal group create --name bicep-rg --location eastus

az deployment group create \
  --resource-group bicep-rg \
  --template-file main.bicep \
  --parameters saName=mystorage clusterName=mycluster
```

`az` builds the Bicep file to ARM JSON and PUTs it to miniblue's `Microsoft.Resources/deployments` endpoint, which dispatches each resource to its handler.

Verify:

```bash
azlocal storage account show --resource-group bicep-rg --name mystorage
azlocal aks list --resource-group bicep-rg
```

## Phase 1 scope

Supported in this release:

- PUT/GET/DELETE/List on `Microsoft.Resources/deployments`
- Linear iteration of `template.resources` in declaration order
- `[parameters('name')]` substitution (with `defaultValue` fallback)
- `[variables('name')]` substitution
- Each resource dispatched as a synthetic PUT against its existing handler
- `provisioningState` of `Succeeded` or `Failed`; per-resource error message on failure

Not supported yet (open follow-up issue #74):

- `[concat(...)]`, `[resourceId(...)]`, `[reference(...)]`, and other template functions
- `copy` loops
- `condition` on resources
- Nested templates and module references
- `outputs` evaluation
- `dependsOn` ordering (resources are applied in declaration order)
- Subscription-scope, management-group-scope, and tenant-scope deployments
- `What-If` and `Validate` operations

For Bicep files that use unsupported constructs, the deployment endpoint will pass them through to the handler unchanged (which usually means the resource gets a literal expression string in the field). Use literals or pre-flattened ARM JSON in the meantime.
