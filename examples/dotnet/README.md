# miniblue .NET Example

Demonstrates four Azure services running against **[miniblue](https://github.com/moabukar/miniblue)** — a free, open-source local Azure emulator.

All calls go to `http://localhost:4566` using plain `HttpClient`. No real Azure subscription or credentials required.

## Prerequisites

| Tool | Version |
|------|---------|
| [.NET SDK](https://dotnet.microsoft.com/download) | 10.0+ |
| [Docker](https://docs.docker.com/get-docker/) | any recent |

## Start miniblue

```bash
docker run -p 4566:4566 -p 4567:4567 moabukar/miniblue:latest
```

## Run the example

```bash
cd examples/dotnet
dotnet run
```

Expected output:

```
miniblue .NET SDK example
=========================

Resource Group: dotnet-rg (eastus)
All resource groups:
  - dotnet-rg

Secret stored: db-password
Secret value:  super-secret-dotnet-123

Container created: data
Blob uploaded: config.json
Blob content:  {"database":"postgres://localhost:5432/mydb","env":"local"}

Cosmos doc created: user1
Cosmos doc: Mo (admin)

All calls went to miniblue!
```

## What each file does

| File | Service | Operations |
|------|---------|------------|
| `ResourceGroupExample.cs` | ARM Resource Groups | Create, list |
| `KeyVaultExample.cs` | Key Vault Secrets | Set, get |
| `BlobStorageExample.cs` | Blob Storage | Create container, upload, download |
| `CosmosDbExample.cs` | Cosmos DB | Insert document, read document |

## Notes

- The Azure SDK clients for Key Vault (`Azure.Security.KeyVault.Secrets`) and Blob Storage (`Azure.Storage.Blobs`) enforce HTTPS or use incompatible path-style URL parsing for custom endpoints, so this example uses `HttpClient` directly — mirroring the approach in the Python example.
- miniblue does **not** validate authentication tokens or HMAC signatures.
