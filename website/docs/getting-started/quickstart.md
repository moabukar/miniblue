# Quick Start

Your first 5 minutes with miniblue. By the end you will have created a resource group, stored a secret in Key Vault, and uploaded a blob -- all locally, no Azure account needed.

## 1. Start miniblue

=== "Docker"

    ```bash
    docker run -d --name miniblue -p 4566:4566 -p 4567:4567 moabukar/miniblue:latest
    ```

=== "Binary"

    ```bash
    ./bin/miniblue
    ```

Wait for the health check to pass:

```bash
curl -s http://localhost:4566/health | jq .status
```

```
"running"
```

## 2. Create a resource group

Every Azure resource lives inside a resource group. Create one:

=== "curl"

    ```bash
    curl -X PUT "http://localhost:4566/subscriptions/sub1/resourcegroups/quickstart-rg" \
      -H "Content-Type: application/json" \
      -d '{"location": "eastus"}'
    ```

=== "azlocal"

    ```bash
    azlocal group create --name quickstart-rg --location eastus
    ```

Response:

```json
{
  "id": "/subscriptions/sub1/resourceGroups/quickstart-rg",
  "name": "quickstart-rg",
  "type": "Microsoft.Resources/resourceGroups",
  "location": "eastus",
  "properties": {
    "provisioningState": "Succeeded"
  }
}
```

List all resource groups to confirm:

```bash
azlocal group list
```

## 3. Store a secret in Key Vault

Create a secret called `db-password` in a vault called `myvault`:

=== "curl"

    ```bash
    curl -X PUT "http://localhost:4566/keyvault/myvault/secrets/db-password" \
      -H "Content-Type: application/json" \
      -d '{"value": "P@ssw0rd123!"}'
    ```

=== "azlocal"

    ```bash
    azlocal keyvault secret set --vault myvault --name db-password --value "P@ssw0rd123!"
    ```

Retrieve it:

=== "curl"

    ```bash
    curl -s "http://localhost:4566/keyvault/myvault/secrets/db-password" | jq .value
    ```

=== "azlocal"

    ```bash
    azlocal keyvault secret show --vault myvault --name db-password
    ```

## 4. Upload a blob

Create a storage container and upload a file:

=== "curl"

    ```bash
    # Create container
    curl -X PUT "http://localhost:4566/blob/myaccount/mycontainer"

    # Upload blob
    curl -X PUT "http://localhost:4566/blob/myaccount/mycontainer/hello.txt" \
      -H "Content-Type: text/plain" \
      -d "Hello from miniblue!"

    # Download blob
    curl "http://localhost:4566/blob/myaccount/mycontainer/hello.txt"
    ```

=== "azlocal"

    ```bash
    # Create container
    azlocal storage container create --account myaccount --name mycontainer

    # Upload blob
    azlocal storage blob upload --account myaccount --container mycontainer \
      --name hello.txt --data "Hello from miniblue!"

    # Download blob
    azlocal storage blob download --account myaccount --container mycontainer \
      --name hello.txt
    ```

Output:

```
Hello from miniblue!
```

## 5. Clean up

miniblue uses in-memory storage by default. All data is gone when the process stops:

=== "Docker"

    ```bash
    docker stop miniblue && docker rm miniblue
    ```

=== "Binary"

    Press `Ctrl+C` in the terminal running miniblue.

## What's next?

| Goal | Guide |
|------|-------|
| Use Terraform | [Terraform guide](../guides/terraform.md) |
| Explore all services | [Services overview](../services/overview.md) |
| Configure ports and TLS | [Configuration](configuration.md) |
| Use the Azure CLI | [Azure CLI guide](../guides/azure-cli.md) |
