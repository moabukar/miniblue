# Contributing to miniblue

Thanks for your interest in contributing! Here is how you can help.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/miniblue.git`
3. Create a branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Test: `go build ./... && go test ./...`
6. Commit and push
7. Open a Pull Request

## Adding a New Azure Service

Each service lives in its own package under `internal/services/`. Follow the existing pattern:

1. Create `internal/services/yourservice/handler.go`
2. Define a `Handler` struct with a `*store.Store` field
3. Implement `NewHandler(s *store.Store) *Handler`
4. Implement `Register(r chi.Router)` to set up routes
5. Register your handler in `internal/server/server.go`

### Example skeleton

```go
package yourservice

import (
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
    // Add your routes here
}
```

## Code Style

- Run `go vet ./...` before committing
- Use standard Go formatting (`gofmt`)
- Keep handlers focused - one service per package
- Use regular dashes (-) not em dashes
- British spelling in docs (colour, behaviour, etc.)

## Reporting Bugs

Use the bug report template when opening an issue. Include:
- Steps to reproduce
- Expected vs actual behaviour
- Your environment (OS, Go version, Docker or binary)

## Suggesting Services

Open a feature request issue with the Azure service name and links to the relevant Azure REST API docs. This helps us match the real API surface.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
