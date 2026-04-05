# Docker

## Docker run

=== "Docker Hub"

    ```bash
    docker run -d \
      --name miniblue \
      -p 4566:4566 \
      -p 4567:4567 \
      moabukar/miniblue:latest
    ```

=== "GitHub Container Registry"

    ```bash
    docker run -d \
      --name miniblue \
      -p 4566:4566 \
      -p 4567:4567 \
      ghcr.io/moabukar/miniblue:latest
    ```

Verify:

```bash
curl http://localhost:4566/health
```

## Docker Compose

```yaml
services:
  miniblue:
    image: moabukar/miniblue:latest
    ports:
      - "4566:4566"
      - "4567:4567"
    environment:
      - PORT=4566
      - LOG_LEVEL=info
    healthcheck:
      test: ["CMD", "wget", "--no-check-certificate", "-q", "--spider", "http://localhost:4566/health"]
      interval: 10s
      timeout: 5s
      retries: 3
```

```bash
docker compose up -d
```

### With your application

```yaml
services:
  miniblue:
    image: moabukar/miniblue:latest
    ports:
      - "4566:4566"
      - "4567:4567"
    healthcheck:
      test: ["CMD", "wget", "--no-check-certificate", "-q", "--spider", "http://localhost:4566/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  app:
    build: .
    depends_on:
      miniblue:
        condition: service_healthy
    environment:
      - AZURE_ENDPOINT=http://miniblue:4566
```

!!! note "Service-to-service networking"
    Inside a Docker Compose network, your app reaches miniblue at `http://miniblue:4566` (the service name), not `localhost`.

## Health checks

miniblue exposes a `/health` endpoint on both HTTP and HTTPS ports.

### curl

```bash
curl http://localhost:4566/health
```

### Docker healthcheck

```yaml
healthcheck:
  test: ["CMD", "wget", "--no-check-certificate", "-q", "--spider", "http://localhost:4566/health"]
  interval: 10s
  timeout: 5s
  retries: 3
```

### Wait-for script

If you need to wait for miniblue before running other commands:

```bash
#!/bin/bash
echo "Waiting for miniblue..."
until curl -sf http://localhost:4566/health > /dev/null 2>&1; do
  sleep 1
done
echo "miniblue is ready."
```

## Environment variables

Pass configuration via `-e` flags or the `environment` section:

```bash
docker run -d \
  --name miniblue \
  -p 8080:8080 \
  -p 8443:8443 \
  -e PORT=8080 \
  -e TLS_PORT=8443 \
  -e LOG_LEVEL=debug \
  moabukar/miniblue:latest
```

See [Configuration](../getting-started/configuration.md) for all available variables.

## Accessing the TLS certificate

miniblue generates a self-signed certificate inside the container. To use it on the host (e.g. for Terraform), mount the cert directory:

```bash
docker run -d \
  --name miniblue \
  -p 4566:4566 \
  -p 4567:4567 \
  -v ~/.miniblue:/root/.miniblue \
  moabukar/miniblue:latest
```

The certificate is now available at `~/.miniblue/cert.pem` on your host.

## Building the image locally

```bash
git clone https://github.com/moabukar/miniblue.git
cd miniblue
docker build -t miniblue:local .
docker run -p 4566:4566 -p 4567:4567 miniblue:local
```

Or via Make:

```bash
make docker-build
make docker-run
```

## Image details

| Property | Value |
|----------|-------|
| Base image | `alpine:3.19` |
| Size | ~15 MB |
| Entrypoint | `/miniblue` |
| Exposed port | `4566` |
| User | `root` (Alpine default) |

The image uses a two-stage build: Go compilation in `golang:1.26-alpine`, then the static binary is copied into a minimal Alpine image.

## Stopping and removing

```bash
docker stop miniblue
docker rm miniblue
```

Or with Compose:

```bash
docker compose down
```

All in-memory data is lost when the container stops.
