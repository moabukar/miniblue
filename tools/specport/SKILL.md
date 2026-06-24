# SKILL: porting an Azure service spec into miniblue

This is the runbook for closing gaps between miniblue's emulated service
surface and the official Azure REST API. It is meant to be followed
end-to-end whenever you (or an AI agent) implement a new Azure service or
expand an existing one.

The goal is **good-enough fidelity**, not 1:1 API parity. The bar is set
in [ROADMAP.md](../../ROADMAP.md): responses must satisfy Terraform, Bicep
and the official Azure SDKs for typical create/read/update/delete flows.
Edge cases like full RBAC, replication semantics, and exotic LRO state
machines are explicitly out of scope.

The companion `specport` CLI in this directory mechanically inventories
every Azure operation for one service and tells you which ones miniblue
already serves; this skill is the *judgment layer* on top of that
inventory.

---

## When to use this

- Adding a new service from [ROADMAP.md](../../ROADMAP.md) (e.g. App Service,
  Traffic Manager, Network Interfaces).
- Extending an existing service whose parity matrix entry says
  "Stub" / "Partial" / has known gaps in
  [website/docs/services/parity.md](../../website/docs/services/parity.md).
- Reviewing a PR that adds endpoints — the checklist diff is the source
  of truth for "did this PR close real gaps or add unspecified ones?".

## Prerequisites

- Go 1.26+ (matches [go.mod](../../go.mod)).
- Internet access for the first `extract` of a service. Subsequent runs
  can point `url:` entries at vendored copies on disk if you prefer
  reproducibility.
- Familiarity with miniblue's handler shape — read
  [CONTRIBUTING.md](../../CONTRIBUTING.md), then skim
  [internal/services/keyvault/handler.go](../../internal/services/keyvault/handler.go)
  for a small reference handler and
  [internal/services/storageaccounts/handler.go](../../internal/services/storageaccounts/handler.go)
  for a richer one.

---

## Workflow

### 1. Add (or update) the service config

Service configs live in [`tools/specport/specs/`](specs/). One YAML file
per service, named after the slug. Minimum shape:

```yaml
service: <slug>           # e.g. keyvault
display_name: <pretty name>
rp_namespace: Microsoft.Foo
sources:
  - name: arm-management
    plane: arm
    url: https://raw.githubusercontent.com/Azure/azure-rest-api-specs/main/...
  - name: data-plane-foo
    plane: data-plane
    url: https://raw.githubusercontent.com/Azure/azure-rest-api-specs/main/...
    miniblue_path_prefix: /foo/{instanceName}
match_route_filters:
  - /providers/Microsoft.Foo
  - /foo
```

Picking spec versions:

- For ARM, use the **latest stable** API version under
  `specification/<org>/resource-manager/<RPNS>/<service>/stable/<YYYY-MM-DD>/`.
- For data-plane, use the **latest stable** under
  `specification/<org>/data-plane/<service>/stable/<YYYY-MM-DD>/` (or
  `<x.y>` for legacy versioned specs).
- One `sources:` entry per swagger file. If a service has multiple ARM
  swagger files (e.g. private endpoints split out), list each one.

Picking `miniblue_path_prefix` for data-plane sources:

- Real Azure data plane lives behind a vanity host
  (`{vaultBaseUrl}`, `{accountName}.queue.core.windows.net`). miniblue
  routes everything under one HTTP host, so we prepend a prefix that
  mirrors how the existing handler is registered. See
  [internal/services/keyvault/handler.go](../../internal/services/keyvault/handler.go)
  (`/keyvault/{vaultName}/secrets/...`) for the pattern.

### 2. Extract the operation inventory

```sh
go run ./tools/specport extract <slug>
```

This writes:

- `tools/specport/checklists/<slug>.md` — human-readable, committed.
- `tools/specport/checklists/<slug>.json` — machine-readable, committed.

Every row starts as `TODO`. The next step replaces those.

### 3. Run the diff against the running router

```sh
go run ./tools/specport diff <slug>
```

This re-extracts, then boots `internal/server.New()`, walks every chi
route with `chi.Walk`, and marks each spec entry as **IMPLEMENTED**,
**MISSING**, or surfaces routes miniblue has that the spec does not as
**EXTRA**.

Commit both files. The PR review surface for "what does this PR change in
service coverage?" is now a single git diff on the markdown.

Use `--fail-on-missing` in CI to fail builds when MISSING regresses.

### 4. Close the MISSING rows

For each `MISSING` operation, add a chi route in
`internal/services/<slug>/handler.go` (or a new file in the same package).

The handler skeleton is fixed across the codebase — copy from any
existing service:

```go
func (h *Handler) Register(r chi.Router) {
    r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Foo/bars",
        func(r chi.Router) {
            r.Get("/", h.ListBars)
            r.Route("/{barName}", func(r chi.Router) {
                r.Put("/", h.CreateOrUpdateBar)
                r.Get("/", h.GetBar)
                r.Delete("/", h.DeleteBar)
            })
        })
}
```

Body decoding rule: **always** use `map[string]interface{}` rather than
typed structs. The whole codebase does this so that minor schema drift
in Azure's spec does not break miniblue.

Response building rule: copy the smallest happy-path example from the
spec's `examples/` sibling folder, then trim it down to the fields the
azurerm Terraform provider and the Azure Go SDK actually read. Keep
nested objects (`properties`, `sku`, `identity`) so providers don't
panic on missing fields. Always include:

- `id` — full ARM path (`/subscriptions/{}/resourceGroups/{}/providers/...`)
- `name` — the resource name
- `type` — e.g. `Microsoft.Foo/bars`
- `location` — copied from input or defaulted to `eastus`
- `properties.provisioningState: "Succeeded"`

### 5. Long-running operations

For any spec row with `LRO yes`, the simplest contract that satisfies
SDKs is:

1. Return `202 Accepted` with `Azure-AsyncOperation` and
   `Location` headers pointing at a synthesized endpoint.
2. Persist the resource immediately into the store as if the operation
   already succeeded.
3. Have the synthesized endpoint return `{"status": "Succeeded"}` on
   the first poll.

Don't model intermediate states (`Creating`, `Updating`) unless a
specific test or example needs them. The roadmap's `provisioningState`
guidance (always-`Succeeded` for stub services) applies.

### 6. Status codes

Azure has subtle rules:

- **PUT** create returns `201 Created`, PUT update returns `200 OK`.
  See [internal/services/storageaccounts/handler.go](../../internal/services/storageaccounts/handler.go)
  (which deviates: storage RP uses 200 for both because the Go SDK
  rejects 201) and [internal/services/aks/handler.go](../../internal/services/aks/handler.go)
  (which uses 201/200 split).
- **DELETE** returns `200 OK` for sync deletes, `202 Accepted` if the
  spec marks it LRO, `204 No Content` if the resource didn't exist.
- **GET** missing → `404` via `azerr.NotFound`.

When in doubt, copy the status-code logic from the closest existing
handler.

### 7. Errors

Use the helpers in [internal/azerr/errors.go](../../internal/azerr/errors.go).
Never write raw `http.Error(...)` — Azure clients parse the JSON shape
of the error body and will fail if you return text.

### 8. Tests

Add a `handler_test.go` modelled on
[internal/services/aks/handler_test.go](../../internal/services/aks/handler_test.go).
At minimum:

- Create returns `201` first time, `200` on update.
- GET after create returns the same body.
- DELETE removes the resource; subsequent GET is `404`.
- List returns the right number of items at both subscription and
  resource-group scope (where applicable).

### 9. Re-run the diff and update the parity matrix

```sh
go run ./tools/specport diff <slug>
```

Coverage should rise. Commit the updated checklist and update the row
in [website/docs/services/parity.md](../../website/docs/services/parity.md)
to reflect the new state.

### 10. Register the handler

If the service is brand new, add the registration in
[internal/server/server.go](../../internal/server/server.go) inside the
`services` slice. Existing services are already registered.

---

## Out of scope for any single PR

- Generating Go structs from the spec (against the project's design;
  see [ROADMAP.md](../../ROADMAP.md) "Non-goals").
- Compiling TypeSpec — Azure ships emitted Swagger 2.0 alongside the
  `.tsp` sources; we read the JSON, not the TypeSpec.
- Real auth enforcement — miniblue accepts any credential by design.
- Multi-region / replication semantics — regions are cosmetic.

## Reference checklists

- Pilot: [`checklists/keyvault.md`](checklists/keyvault.md).
- Subsequent services follow the same shape.
