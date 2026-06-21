# miniblue Java Example

Demonstrates four Azure services running against **[miniblue](https://github.com/moabukar/miniblue)** - a free, open-source local Azure emulator.

All calls go to `http://localhost:4566` using the built-in `java.net.http.HttpClient`. No real Azure subscription or credentials required.

## Prerequisites

| Tool | Version |
|------|---------|
| [JDK](https://adoptium.net/) | 17+ |
| [Maven](https://maven.apache.org/) | 3.8+ |
| [Docker](https://docs.docker.com/get-docker/) | any recent |

## Start miniblue

```bash
docker run -p 4566:4566 -p 4567:4567 moabukar/miniblue:latest
```

## Run the example

```bash
cd examples/java
mvn -q compile exec:java
```

Expected output:

```
miniblue Java example
=========================

Resource Group: java-rg (eastus)
All resource groups:
  - java-rg

Secret stored: db-password
Secret value:  super-secret-java-123

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
| `ResourceGroupExample.java` | ARM Resource Groups | Create, list |
| `KeyVaultExample.java` | Key Vault Secrets | Set, get |
| `BlobStorageExample.java` | Blob Storage | Create container, upload, download |
| `CosmosDbExample.java` | Cosmos DB | Insert document, read document |

## Notes

- The Azure SDK clients for Key Vault (`azure-security-keyvault-secrets`) and Blob Storage (`azure-storage-blob`) enforce HTTPS or use incompatible path-style URL parsing for custom endpoints, so this example uses `HttpClient` directly - mirroring the approach in the .NET and Python examples.
- miniblue does **not** validate authentication tokens or HMAC signatures.
