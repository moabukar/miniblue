# Blob Storage

miniblue emulates Azure Blob Storage with container and blob CRUD operations. Data is stored in memory.

## API endpoints

### Containers

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/blob/{account}/{container}` | Create container |
| `GET` | `/blob/{account}/{container}` | List blobs in container |
| `DELETE` | `/blob/{account}/{container}` | Delete container |

### Blobs

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/blob/{account}/{container}/{blob}` | Upload blob |
| `GET` | `/blob/{account}/{container}/{blob}` | Download blob |
| `DELETE` | `/blob/{account}/{container}/{blob}` | Delete blob |

## Create a container

```bash
curl -X PUT "http://localhost:4566/blob/myaccount/mycontainer"
```

Response: `201 Created`

## Upload a blob

```bash
curl -X PUT "http://localhost:4566/blob/myaccount/mycontainer/hello.txt" \
  -H "Content-Type: text/plain" \
  -d "Hello from miniblue!"
```

Response: `201 Created`

Upload a JSON file:

```bash
curl -X PUT "http://localhost:4566/blob/myaccount/mycontainer/config.json" \
  -H "Content-Type: application/json" \
  -d '{"database": "localhost", "port": 5432}'
```

Upload a binary file:

```bash
curl -X PUT "http://localhost:4566/blob/myaccount/mycontainer/image.png" \
  -H "Content-Type: image/png" \
  --data-binary @./image.png
```

## Download a blob

```bash
curl "http://localhost:4566/blob/myaccount/mycontainer/hello.txt"
```

```
Hello from miniblue!
```

The response includes Azure-compatible headers:

| Header | Example |
|--------|---------|
| `Content-Type` | `text/plain` |
| `Content-Length` | `20` |
| `ETag` | `"0x1A2B3C4D5E6F"` |

## List blobs in a container

```bash
curl "http://localhost:4566/blob/myaccount/mycontainer"
```

```json
{
  "blobs": [
    {
      "name": "hello.txt",
      "properties": {
        "contentLength": "20",
        "contentType": "text/plain",
        "etag": "\"0x1A2B3C4D5E6F\"",
        "lastModified": "Mon, 01 Jan 2026 00:00:00 UTC"
      }
    },
    {
      "name": "config.json",
      "properties": {
        "contentLength": "38",
        "contentType": "application/json",
        "etag": "\"0x1A2B3C4D5E70\"",
        "lastModified": "Mon, 01 Jan 2026 00:00:01 UTC"
      }
    }
  ]
}
```

## Delete a blob

```bash
curl -X DELETE "http://localhost:4566/blob/myaccount/mycontainer/hello.txt"
```

Response: `202 Accepted`

## Delete a container

```bash
curl -X DELETE "http://localhost:4566/blob/myaccount/mycontainer"
```

Response: `202 Accepted`

## azlocal

```bash
# Containers
azlocal storage container create --account myaccount --name mycontainer
azlocal storage container delete --account myaccount --name mycontainer

# Blobs
azlocal storage blob upload --account myaccount --container mycontainer \
  --name hello.txt --data "Hello from miniblue!"

azlocal storage blob upload --account myaccount --container mycontainer \
  --name config.json --file ./config.json

azlocal storage blob download --account myaccount --container mycontainer \
  --name hello.txt

azlocal storage blob list --account myaccount --container mycontainer

azlocal storage blob delete --account myaccount --container mycontainer \
  --name hello.txt
```

## Full example

```bash
#!/bin/bash
set -e

ACCOUNT="devaccount"
CONTAINER="uploads"

# Create container
curl -X PUT "http://localhost:4566/blob/${ACCOUNT}/${CONTAINER}"

# Upload three files
for f in file1.txt file2.txt file3.txt; do
  curl -X PUT "http://localhost:4566/blob/${ACCOUNT}/${CONTAINER}/${f}" \
    -H "Content-Type: text/plain" \
    -d "Content of ${f}"
done

# List all blobs
curl -s "http://localhost:4566/blob/${ACCOUNT}/${CONTAINER}" | jq '.blobs[].name'

# Download one
curl "http://localhost:4566/blob/${ACCOUNT}/${CONTAINER}/file1.txt"

# Clean up
curl -X DELETE "http://localhost:4566/blob/${ACCOUNT}/${CONTAINER}"
```
