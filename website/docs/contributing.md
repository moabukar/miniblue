# Contributing

Contributions are welcome. This page covers project structure, how to add a new service, and development workflow.

## Getting started

```bash
git clone https://github.com/moabukar/miniblue.git
cd miniblue
make build
make test
```

Requirements:

- Go 1.25+
- Make (optional but recommended)

## Project structure

```
miniblue/
  cmd/
    miniblue/          # Server entrypoint
      main.go          # HTTP + HTTPS listeners, TLS cert generation
    azlocal/           # CLI client
      main.go          # Command routing, HTTP helpers
  internal/
    azerr/             # Azure-compatible error responses
    server/
      server.go        # Router setup, middleware, service registration
      middleware.go     # Azure headers, API version check
    services/
      acr/             # Container Registry
      appconfig/        # App Configuration
      auth/            # OAuth2 token endpoints
      blob/            # Blob Storage
      cosmosdb/        # Cosmos DB
      dns/             # DNS Zones
      eventgrid/       # Event Grid
      functions/       # Azure Functions
      identity/        # Managed Identity (IMDS)
      keyvault/        # Key Vault
      metadata/        # Cloud metadata endpoint
      network/         # Virtual Networks + Subnets
      queue/           # Queue Storage
      resourcegroups/  # Resource Groups
      servicebus/      # Service Bus
      subscriptions/   # Subscription + tenant listing
      table/           # Table Storage
    store/
      store.go         # Thread-safe in-memory key-value store
  examples/
    terraform/         # Terraform example
  scripts/
    trust-cert.sh      # Trust self-signed cert
    az-login-local.sh  # Legacy (use azlocal instead)
  website/
    docs/              # This documentation (mkdocs)
    mkdocs.yml
```

## Adding a new Azure service

Every service follows the same pattern. Here is a step-by-step guide.

### 1. Create the handler package

Create `internal/services/yourservice/handler.go`:

```go
package yourservice

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/moabukar/miniblue/internal/store"
)

type Handler struct {
    store *store.Store
}

func NewHandler(s *store.Store) *Handler {
    return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
    r.Route("/yourservice/{resourceName}", func(r chi.Router) {
        r.Put("/", h.CreateOrUpdate)
        r.Get("/", h.Get)
        r.Delete("/", h.Delete)
    })
}

func (h *Handler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
    name := chi.URLParam(r, "resourceName")

    var input map[string]interface{}
    json.NewDecoder(r.Body).Decode(&input)

    h.store.Set("yourservice:"+name, input)
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(input)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
    name := chi.URLParam(r, "resourceName")

    v, ok := h.store.Get("yourservice:" + name)
    if !ok {
        w.WriteHeader(http.StatusNotFound)
        return
    }
    json.NewEncoder(w).Encode(v)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
    name := chi.URLParam(r, "resourceName")
    h.store.Delete("yourservice:" + name)
    w.WriteHeader(http.StatusOK)
}
```

### 2. Register in the server

Edit `internal/server/server.go`:

```go
import (
    // ... existing imports ...
    "github.com/moabukar/miniblue/internal/services/yourservice"
)

func (s *Server) setupRoutes() {
    // ... existing routes ...
    yourservice.NewHandler(s.store).Register(s.router)
}
```

### 3. Add to the health check

In `internal/server/server.go`, add the service name to the `services` slice in `healthHandler`.

### 4. Add azlocal commands (optional)

Add a handler function in `cmd/azlocal/main.go` and register it in the `switch` statement.

### 5. Write tests

```bash
go test ./internal/services/yourservice/...
```

### 6. Update documentation

Add a page under `website/docs/services/` and update `website/mkdocs.yml`.

## Key patterns

### Store

The `store.Store` is a thread-safe in-memory key-value store. Every service uses string keys with a prefix:

```go
h.store.Set("rg:sub1:myRG", resourceGroup)        // Resource group
h.store.Set("kv:myvault:dbpass", secret)           // Key Vault secret
h.store.Set("blob:blob:acct:cont:file.txt", blob)  // Blob
```

Useful store methods:

| Method | Description |
|--------|-------------|
| `Set(key, value)` | Store a value |
| `Get(key)` | Retrieve a value (returns `value, bool`) |
| `Delete(key)` | Delete a key (returns `bool`) |
| `Exists(key)` | Check if a key exists |
| `ListByPrefix(prefix)` | List all values matching a prefix |
| `CountByPrefix(prefix)` | Count keys matching a prefix |
| `DeleteByPrefix(prefix)` | Delete all keys matching a prefix |

### Error responses

Use `internal/azerr` for Azure-compatible error responses:

```go
import "github.com/moabukar/miniblue/internal/azerr"

azerr.NotFound(w, "Microsoft.YourService/resources", name)
azerr.BadRequest(w, "Invalid input")
azerr.Conflict(w, "Microsoft.YourService/resources", name)
```

### ARM-style routes

For resources managed through Azure Resource Manager, use the standard ARM URL pattern:

```go
r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.YourService/resources", func(r chi.Router) {
    r.Get("/", h.List)
    r.Route("/{resourceName}", func(r chi.Router) {
        r.Put("/", h.CreateOrUpdate)
        r.Get("/", h.Get)
        r.Delete("/", h.Delete)
    })
})
```

## Code style

- Run `go vet ./...` before committing
- Use `gofmt` formatting
- One service per package under `internal/services/`
- Keep handlers focused -- no cross-service dependencies
- Use regular dashes (`-`) not em dashes

## Development workflow

```bash
# Run the server with live reload (if using air or similar)
make run

# Run all tests
make test

# Lint
make lint

# Build everything
make build

# Clean build artifacts
make clean
```

## Submitting changes

1. Fork the repository
2. Create a branch: `git checkout -b feature/my-feature`
3. Make your changes and add tests
4. Run `go build ./... && go test ./...`
5. Commit and push
6. Open a Pull Request

## Reporting bugs

Open an issue at [github.com/moabukar/miniblue/issues](https://github.com/moabukar/miniblue/issues) with:

- Steps to reproduce
- Expected vs actual behaviour
- Environment (OS, Go version, Docker or binary)

## Suggesting new services

Open a feature request issue with the Azure service name and a link to its [REST API documentation](https://learn.microsoft.com/en-us/rest/api/azure/). This helps match the real API surface.

## Licence

MIT. By contributing, you agree your contributions are licensed under the MIT licence.
