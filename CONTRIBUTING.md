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

## Prerequisites for Python Examples

On Debian/Ubuntu you need `python3-venv` (or the version-specific variant) before creating a virtualenv:

```bash
# Debian/Ubuntu
sudo apt install python3.12-venv   # or python3-venv for the system default

# Then create a venv as normal
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

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

## Closing gaps against Azure's REST specs

When implementing a new service, or filling holes in an existing one, follow [`tools/specport/SKILL.md`](tools/specport/SKILL.md). The `specport` CLI extracts the inventory of operations Azure's official spec defines and diffs it against miniblue's chi router so you can see exactly which endpoints are missing or extra:

```bash
go run ./tools/specport list
go run ./tools/specport extract keyvault
go run ./tools/specport diff keyvault
```

The committed checklists under [`tools/specport/checklists/`](tools/specport/checklists/) are the source of truth for "what's covered" — review them in PRs and re-run `diff` whenever you add routes.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
