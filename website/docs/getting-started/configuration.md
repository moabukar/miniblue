# Configuration

miniblue works with zero configuration out of the box. This page covers everything you can customise.

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `4566` | HTTP server port |
| `TLS_PORT` | `4567` | HTTPS server port (self-signed certificate) |
| `LOG_LEVEL` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `LOCAL_AZURE_ENDPOINT` | `http://localhost:4566` | Endpoint used by the `azlocal` CLI |
| `LOCAL_AZURE_CERT_DIR` | `~/.miniblue` | Directory for TLS certificate and key |
| `DATABASE_URL` | _(unset)_ | PostgreSQL connection string for persistent storage |

Example -- run on custom ports:

```bash
PORT=8080 TLS_PORT=8443 ./bin/miniblue
```

## Ports

| Port | Protocol | Used by |
|------|----------|---------|
| **4566** | HTTP | curl, SDKs, Terraform `endpoint`, azlocal, REST calls |
| **4567** | HTTPS | Terraform `metadata_host`, Azure CLI (MSAL), auth endpoints |

!!! note "Why two ports?"
    The HTTP port (4566) handles all resource management APIs. The HTTPS port (4567) is required because Terraform's `azurerm` provider fetches cloud metadata over HTTPS, and the Azure CLI (MSAL) requires TLS for auth endpoints. If you only use curl or azlocal, you only need port 4566.

## TLS certificate

On first start, miniblue generates a self-signed certificate at:

```
~/.miniblue/cert.pem
~/.miniblue/key.pem
```

The certificate is valid for 1 year. miniblue reuses an existing certificate if it has more than 24 hours until expiry.

### Trusting the certificate

You need to trust the certificate for Terraform, Go SDKs, and other tools that verify TLS.

=== "Quick (current session)"

    ```bash
    export SSL_CERT_FILE=~/.miniblue/cert.pem
    ```

=== "macOS (permanent)"

    ```bash
    sudo security add-trusted-cert -d -r trustRoot \
      -k /Library/Keychains/System.keychain ~/.miniblue/cert.pem
    ```

=== "Debian / Ubuntu (permanent)"

    ```bash
    sudo cp ~/.miniblue/cert.pem /usr/local/share/ca-certificates/miniblue.crt
    sudo update-ca-certificates
    ```

=== "RHEL / Fedora (permanent)"

    ```bash
    sudo cp ~/.miniblue/cert.pem /etc/pki/ca-trust/source/anchors/miniblue.crt
    sudo update-ca-trust
    ```

=== "Script"

    ```bash
    bash scripts/trust-cert.sh
    ```

    This auto-detects your OS and runs the appropriate command.

### Custom certificate directory

```bash
LOCAL_AZURE_CERT_DIR=/tmp/my-certs ./bin/miniblue
```

### Persistent storage

By default miniblue stores everything in memory (lost on restart). For persistence, set `DATABASE_URL`:

```bash
DATABASE_URL=postgres://user:pass@localhost:5432/miniblue ./bin/miniblue
```

Or use the included Postgres docker-compose:

```bash
docker-compose -f docker-compose.postgres.yml up
```

## Docker configuration

Pass environment variables with `-e`:

```bash
docker run -p 8080:8080 -p 8443:8443 \
  -e PORT=8080 \
  -e TLS_PORT=8443 \
  -e LOG_LEVEL=debug \
  moabukar/miniblue:latest
```

To access the generated certificate from the host, mount the cert directory:

```bash
docker run -p 4566:4566 -p 4567:4567 \
  -v ~/.miniblue:/root/.miniblue \
  moabukar/miniblue:latest
```

## Health check

```bash
curl http://localhost:4566/health
```

```json
{
  "services": [
    "subscriptions", "tenants", "resourcegroups", "blob", "table",
    "queue", "keyvault", "cosmosdb", "servicebus", "functions",
    "network", "dns", "acr", "eventgrid", "appconfig", "identity",
    "dbpostgres", "redis", "sqldb", "dbmysql", "publicip", "nsg"
  ],
  "status": "running",
  "version": "0.2.5"
}
```

## Storage

miniblue uses **in-memory storage** by default. All data is lost when the process exits. This is intentional -- it keeps tests fast and repeatable.

!!! info "Persistent storage"
    File-backed persistent storage is on the [roadmap](https://github.com/moabukar/miniblue). Contributions welcome.
