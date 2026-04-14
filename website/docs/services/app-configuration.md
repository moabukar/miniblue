# App Configuration

miniblue emulates Azure App Configuration via data plane endpoints. Supports key-value CRUD operations.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/appconfig/{store}/kv/{key}` | Set key-value |
| `GET` | `/appconfig/{store}/kv/{key}` | Get key-value |
| `GET` | `/appconfig/{store}/kv` | List key-values |
| `DELETE` | `/appconfig/{store}/kv/{key}` | Delete key-value |

## Limitations

- No labels or feature flags
- No configuration snapshots
- No event notifications
