# Storage Accounts

miniblue emulates Azure Storage Accounts via ARM endpoints with shared key authentication. Storage accounts provide the management layer on top of blob, queue, table and file storage.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}` | Create or update |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}` | Get |
| `DELETE` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}` | Delete |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts` | List |

## Create a storage account

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Storage/storageAccounts/mystorageacct?api-version=2023-05-01" \
  -H "Content-Type: application/json" \
  -d '{
    "location": "eastus",
    "kind": "StorageV2",
    "sku": {"name": "Standard_LRS"}
  }'
```

## Terraform

```hcl
resource "azurerm_storage_account" "example" {
  name                     = "examplestorage"
  resource_group_name      = azurerm_resource_group.example.name
  location                 = azurerm_resource_group.example.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
}
```

## Shared key authentication

miniblue supports Azure shared key authentication for storage data plane operations. Keys are auto-generated when a storage account is created.

## Limitations

- No lifecycle management policies
- No private endpoints or firewall rules
- No geo-replication
- File storage data plane is not yet implemented
