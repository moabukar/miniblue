# Cosmos DB

miniblue emulates Azure Cosmos DB with a SQL API-compatible document store. Create, read, update, delete, and list JSON documents.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/cosmosdb/{account}/dbs/{db}/colls/{coll}/docs` | Create document |
| `GET` | `/cosmosdb/{account}/dbs/{db}/colls/{coll}/docs` | List documents |
| `GET` | `/cosmosdb/{account}/dbs/{db}/colls/{coll}/docs/{id}` | Get document |
| `PUT` | `/cosmosdb/{account}/dbs/{db}/colls/{coll}/docs/{id}` | Replace document |
| `DELETE` | `/cosmosdb/{account}/dbs/{db}/colls/{coll}/docs/{id}` | Delete document |

!!! note
    Accounts, databases, and collections are created implicitly. You do not need to create them before inserting documents.

## Create a document

Every document must contain an `id` property.

```bash
curl -X POST "http://localhost:4566/cosmosdb/myaccount/dbs/mydb/colls/users/docs" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "user-1",
    "name": "Alice",
    "email": "alice@example.com",
    "age": 30
  }'
```

Response (`201 Created`):

```json
{
  "id": "user-1",
  "name": "Alice",
  "email": "alice@example.com",
  "age": 30
}
```

## Get a document

```bash
curl "http://localhost:4566/cosmosdb/myaccount/dbs/mydb/colls/users/docs/user-1"
```

Returns `404` if the document does not exist.

## Replace (update) a document

```bash
curl -X PUT "http://localhost:4566/cosmosdb/myaccount/dbs/mydb/colls/users/docs/user-1" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Alice Smith",
    "email": "alice.smith@example.com",
    "age": 31
  }'
```

The `id` in the URL takes precedence -- the document's `id` field is set to `user-1` regardless of the request body.

## List documents

```bash
curl "http://localhost:4566/cosmosdb/myaccount/dbs/mydb/colls/users/docs"
```

Response:

```json
{
  "Documents": [
    {
      "id": "user-1",
      "name": "Alice Smith",
      "email": "alice.smith@example.com",
      "age": 31
    },
    {
      "id": "user-2",
      "name": "Bob",
      "email": "bob@example.com",
      "age": 25
    }
  ],
  "_count": 2
}
```

## Delete a document

```bash
curl -X DELETE "http://localhost:4566/cosmosdb/myaccount/dbs/mydb/colls/users/docs/user-1"
```

Response: `204 No Content`

## azlocal

```bash
# Create
azlocal cosmosdb doc create --account myaccount --database mydb \
  --collection users --id user-1 --data '{"name":"Alice","age":30}'

# Get
azlocal cosmosdb doc show --account myaccount --database mydb \
  --collection users --id user-1

# List
azlocal cosmosdb doc list --account myaccount --database mydb \
  --collection users

# Delete
azlocal cosmosdb doc delete --account myaccount --database mydb \
  --collection users --id user-1
```

## Full example

```bash
#!/bin/bash
set -e

ACCOUNT="shopdb"
DB="ecommerce"
COLL="products"
BASE="http://localhost:4566/cosmosdb/${ACCOUNT}/dbs/${DB}/colls/${COLL}/docs"

# Insert products
curl -X POST "${BASE}" -H "Content-Type: application/json" \
  -d '{"id": "prod-1", "name": "Widget", "price": 9.99, "stock": 100}'

curl -X POST "${BASE}" -H "Content-Type: application/json" \
  -d '{"id": "prod-2", "name": "Gadget", "price": 24.99, "stock": 50}'

curl -X POST "${BASE}" -H "Content-Type: application/json" \
  -d '{"id": "prod-3", "name": "Gizmo", "price": 14.99, "stock": 75}'

# List all products
curl -s "${BASE}" | jq '.Documents[] | {id, name, price}'

# Update stock
curl -X PUT "${BASE}/prod-1" -H "Content-Type: application/json" \
  -d '{"name": "Widget", "price": 9.99, "stock": 99}'

# Delete a product
curl -X DELETE "${BASE}/prod-3"

# Verify
curl -s "${BASE}" | jq '._count'
```

## Limitations

- No SQL query language support -- the list endpoint returns all documents in the collection
- No partition key support
- No indexing or stored procedures
- Documents are stored in memory and lost when miniblue restarts
