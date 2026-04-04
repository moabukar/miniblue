# Service Bus

miniblue emulates Azure Service Bus with queues, topics, and basic messaging. Create queues, send messages, and consume them locally.

## API endpoints

### Queues

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/servicebus/{namespace}/queues/{queue}` | Create queue |
| `GET` | `/servicebus/{namespace}/queues/{queue}` | Get queue |
| `DELETE` | `/servicebus/{namespace}/queues/{queue}` | Delete queue |
| `GET` | `/servicebus/{namespace}/queues` | List queues |
| `POST` | `/servicebus/{namespace}/queues/{queue}/messages` | Send message |
| `GET` | `/servicebus/{namespace}/queues/{queue}/messages/head` | Receive message |

### Topics

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/servicebus/{namespace}/topics/{topic}` | Create topic |
| `DELETE` | `/servicebus/{namespace}/topics/{topic}` | Delete topic |
| `POST` | `/servicebus/{namespace}/topics/{topic}/messages` | Publish message |

## Queues

### Create a queue

```bash
curl -X PUT "http://localhost:4566/servicebus/my-namespace/queues/my-queue"
```

Response (`201 Created`):

```json
{
  "name": "my-queue",
  "properties": {
    "status": "Active",
    "messageCount": 0,
    "maxDeliveryCount": 10,
    "defaultMessageTimeToLive": "P14D",
    "lockDuration": "PT1M",
    "createdAt": "2026-01-01T00:00:00Z"
  }
}
```

Creating a queue that already exists returns `409 Conflict`.

### Get queue details

```bash
curl "http://localhost:4566/servicebus/my-namespace/queues/my-queue"
```

### List all queues

```bash
curl "http://localhost:4566/servicebus/my-namespace/queues"
```

### Send a message

```bash
curl -X POST "http://localhost:4566/servicebus/my-namespace/queues/my-queue/messages" \
  -H "Content-Type: application/json" \
  -d '{"body": "Hello from Service Bus!"}'
```

Response (`201 Created`):

```json
{
  "messageId": "1",
  "body": "Hello from Service Bus!",
  "enqueuedTime": "2026-01-01T00:00:00Z"
}
```

### Receive a message

```bash
curl "http://localhost:4566/servicebus/my-namespace/queues/my-queue/messages/head"
```

Returns the oldest message in the queue. Returns `204 No Content` if the queue is empty.

!!! note
    Receiving a message does not delete it from the queue. Messages persist until the queue is deleted.

### Delete a queue

```bash
curl -X DELETE "http://localhost:4566/servicebus/my-namespace/queues/my-queue"
```

Deleting a queue also deletes all its messages.

## Topics

### Create a topic

```bash
curl -X PUT "http://localhost:4566/servicebus/my-namespace/topics/my-topic"
```

Response (`201 Created`):

```json
{
  "name": "my-topic",
  "properties": {
    "status": "Active"
  }
}
```

### Publish a message to a topic

```bash
curl -X POST "http://localhost:4566/servicebus/my-namespace/topics/my-topic/messages" \
  -H "Content-Type: application/json" \
  -d '{"body": "Broadcast event"}'
```

### Delete a topic

```bash
curl -X DELETE "http://localhost:4566/servicebus/my-namespace/topics/my-topic"
```

## azlocal

### Queues

```bash
# Create
azlocal servicebus queue create --namespace my-ns --name my-queue

# Send
azlocal servicebus queue send --namespace my-ns --name my-queue \
  --body "Hello from Service Bus!"

# Receive
azlocal servicebus queue receive --namespace my-ns --name my-queue

# Delete
azlocal servicebus queue delete --namespace my-ns --name my-queue
```

### Topics

```bash
# Create
azlocal servicebus topic create --namespace my-ns --name my-topic

# Publish
azlocal servicebus topic send --namespace my-ns --name my-topic \
  --body "Broadcast event"

# Delete
azlocal servicebus topic delete --namespace my-ns --name my-topic
```

## Full example -- task queue

```bash
#!/bin/bash
set -e

NS="task-system"
QUEUE="tasks"

# Create queue
curl -X PUT "http://localhost:4566/servicebus/${NS}/queues/${QUEUE}"

# Enqueue tasks
for i in 1 2 3 4 5; do
  curl -X POST "http://localhost:4566/servicebus/${NS}/queues/${QUEUE}/messages" \
    -H "Content-Type: application/json" \
    -d "{\"body\": \"Process order #${i}\"}"
done

# Peek at the first message
curl -s "http://localhost:4566/servicebus/${NS}/queues/${QUEUE}/messages/head" | jq .body

# Check queue status
curl -s "http://localhost:4566/servicebus/${NS}/queues/${QUEUE}" | jq .properties
```

## Limitations

- Topic subscriptions are not yet implemented (topics accept messages but there is no subscription/consumer model)
- No dead-letter queue
- No message scheduling or deferral
- No message sessions or duplicate detection
- Receive is a peek operation -- messages are not removed from the queue
