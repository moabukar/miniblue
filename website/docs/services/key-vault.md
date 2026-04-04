# Key Vault

miniblue emulates Azure Key Vault secret management. Create, read, list, and delete secrets without an Azure subscription.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/keyvault/{vault}/secrets/{name}` | Set a secret |
| `GET` | `/keyvault/{vault}/secrets/{name}` | Get a secret |
| `DELETE` | `/keyvault/{vault}/secrets/{name}` | Delete a secret |
| `GET` | `/keyvault/{vault}/secrets` | List all secrets |

## Set a secret

```bash
curl -X PUT "http://localhost:4566/keyvault/myvault/secrets/db-password" \
  -H "Content-Type: application/json" \
  -d '{"value": "P@ssw0rd123!"}'
```

Response:

```json
{
  "id": "https://myvault.vault.azure.net/secrets/db-password",
  "value": "P@ssw0rd123!",
  "attributes": {
    "created": "2026-01-01T00:00:00Z",
    "enabled": "true",
    "updated": "2026-01-01T00:00:00Z"
  }
}
```

Setting the same secret name again overwrites the value.

## Get a secret

```bash
curl "http://localhost:4566/keyvault/myvault/secrets/db-password"
```

Returns the same JSON structure as above. Returns `404` if the secret does not exist.

## List all secrets in a vault

```bash
curl "http://localhost:4566/keyvault/myvault/secrets"
```

```json
{
  "value": [
    {
      "id": "https://myvault.vault.azure.net/secrets/db-password",
      "value": "P@ssw0rd123!",
      "attributes": {
        "created": "2026-01-01T00:00:00Z",
        "enabled": "true",
        "updated": "2026-01-01T00:00:00Z"
      }
    },
    {
      "id": "https://myvault.vault.azure.net/secrets/api-key",
      "value": "sk-abc123",
      "attributes": {
        "created": "2026-01-01T00:00:01Z",
        "enabled": "true",
        "updated": "2026-01-01T00:00:01Z"
      }
    }
  ]
}
```

## Delete a secret

```bash
curl -X DELETE "http://localhost:4566/keyvault/myvault/secrets/db-password"
```

Response: `200 OK`

## Multiple vaults

Vaults are separated by name. No explicit vault creation is needed -- secrets are scoped to whatever vault name you use in the URL.

```bash
# Different vaults, same secret name
curl -X PUT "http://localhost:4566/keyvault/prod-vault/secrets/api-key" \
  -H "Content-Type: application/json" \
  -d '{"value": "prod-key-123"}'

curl -X PUT "http://localhost:4566/keyvault/dev-vault/secrets/api-key" \
  -H "Content-Type: application/json" \
  -d '{"value": "dev-key-456"}'
```

## azlocal

```bash
# Set
azlocal keyvault secret set --vault myvault --name db-password --value "P@ssw0rd123!"

# Get
azlocal keyvault secret show --vault myvault --name db-password

# List
azlocal keyvault secret list --vault myvault

# Delete
azlocal keyvault secret delete --vault myvault --name db-password
```

## Full example

```bash
#!/bin/bash
set -e

VAULT="app-vault"

# Store application secrets
curl -X PUT "http://localhost:4566/keyvault/${VAULT}/secrets/db-host" \
  -H "Content-Type: application/json" \
  -d '{"value": "localhost"}'

curl -X PUT "http://localhost:4566/keyvault/${VAULT}/secrets/db-password" \
  -H "Content-Type: application/json" \
  -d '{"value": "supersecret"}'

curl -X PUT "http://localhost:4566/keyvault/${VAULT}/secrets/jwt-signing-key" \
  -H "Content-Type: application/json" \
  -d '{"value": "my-signing-key-256"}'

# List all secrets
curl -s "http://localhost:4566/keyvault/${VAULT}/secrets" | jq '.value[].id'

# Retrieve one
DB_PASS=$(curl -s "http://localhost:4566/keyvault/${VAULT}/secrets/db-password" | jq -r '.value')
echo "DB password: ${DB_PASS}"
```
