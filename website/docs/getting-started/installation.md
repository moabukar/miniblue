# Installation

## Homebrew (macOS / Linux)

```bash
brew tap moabukar/tap
brew install miniblue
```

This installs both `miniblue` (server) and `azlocal` (CLI).

## Docker (recommended)

Pull and run with a single command:

=== "Docker Hub"

    ```bash
    docker run -p 4566:4566 -p 4567:4567 moabukar/miniblue:latest
    ```

=== "GitHub Container Registry"

    ```bash
    docker run -p 4566:4566 -p 4567:4567 ghcr.io/moabukar/miniblue:latest
    ```

Verify it is running:

```bash
curl http://localhost:4566/health
```

```json
{
  "services": ["subscriptions","tenants","resourcegroups","blob","table","queue","keyvault","cosmosdb","servicebus","functions","network","dns","acr","eventgrid","appconfig","identity"],
  "status": "running",
  "version": "0.2.0"
}
```

## Binary

### From source (Go 1.26+)

```bash
git clone https://github.com/moabukar/miniblue.git
cd miniblue
make build
```

This produces two binaries in `bin/`:

| Binary | Purpose |
|--------|---------|
| `bin/miniblue` | The emulator server |
| `bin/azlocal` | CLI client (like `awslocal` for LocalStack) |

Start the server:

```bash
./bin/miniblue
```

### Install azlocal globally

```bash
sudo make install
```

This copies `azlocal` to `/usr/local/bin/azlocal`.

### go install

```bash
go install github.com/moabukar/miniblue/cmd/miniblue@latest
go install github.com/moabukar/miniblue/cmd/azlocal@latest
```

## Build from source

Requirements:

- Go 1.26 or later
- Make (optional)

```bash
git clone https://github.com/moabukar/miniblue.git
cd miniblue
go build -o bin/miniblue ./cmd/miniblue
go build -o bin/azlocal ./cmd/azlocal
./bin/miniblue
```

!!! tip "Docker image size"
    The miniblue Docker image is ~15MB (Alpine-based, statically compiled Go binary). Compare that to Azurite (~300MB) or LocalStack (~1GB).

## Verifying the installation

Regardless of install method, confirm everything works:

```bash
# Server health
curl http://localhost:4566/health

# Create a resource group
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourcegroups/test-rg?api-version=2020-06-01" \
  -H "Content-Type: application/json" \
  -d '{"location": "eastus"}'

# Clean up
curl -X DELETE "http://localhost:4566/subscriptions/sub1/resourcegroups/test-rg?api-version=2020-06-01"
```

## Next steps

- [Quick Start](quickstart.md) -- your first 5 minutes with miniblue
- [Configuration](configuration.md) -- ports, environment variables, TLS
- [Docker guide](../guides/docker.md) -- docker-compose, health checks
