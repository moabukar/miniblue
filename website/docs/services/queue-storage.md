# Queue Storage

miniblue emulates Azure Queue Storage via data plane endpoints. Supports queue creation, message send, receive, peek, clear and delete with dequeue count tracking.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/queue/{account}/{queue}` | Create queue |
| `POST` | `/queue/{account}/{queue}/messages` | Send message |
| `GET` | `/queue/{account}/{queue}/messages` | Receive messages |
| `GET` | `/queue/{account}/{queue}/messages?peekonly=true` | Peek messages |
| `DELETE` | `/queue/{account}/{queue}/messages` | Clear messages |
| `DELETE` | `/queue/{account}/{queue}` | Delete queue |

## Limitations

- No visibility timeout on receive
- No message TTL
- No poison queue support
