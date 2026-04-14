# Azure Functions

miniblue emulates Azure Functions via ARM endpoints as a stub. Function apps can be created, listed and deleted but no function execution is performed.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Web/sites/{name}` | Create or update function app |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Web/sites/{name}` | Get function app |
| `DELETE` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Web/sites/{name}` | Delete function app |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Web/sites` | List function apps |

## Limitations

- Stub only: no function execution or triggers
- No deployment slots
- No app settings management
- No consumption plan or premium plan configuration
