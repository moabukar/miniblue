# Event Grid

miniblue emulates Azure Event Grid via ARM and data plane endpoints. Supports topic creation, deletion and event publishing.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.EventGrid/topics/{name}` | Create or update topic |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.EventGrid/topics/{name}` | Get topic |
| `DELETE` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.EventGrid/topics/{name}` | Delete topic |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.EventGrid/topics` | List topics |
| `POST` | `/eventgrid/{topic}/events` | Publish events |

## Limitations

- No event subscriptions or delivery
- No dead-lettering
- No event filtering
- Published events are accepted but not forwarded
