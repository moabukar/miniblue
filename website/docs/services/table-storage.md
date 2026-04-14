# Table Storage

miniblue emulates Azure Table Storage via data plane endpoints. Supports table creation, entity upsert, get, query and delete with partition and row key support.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/table/{account}/Tables` | Create table |
| `POST` | `/table/{account}/{table}` | Upsert entity |
| `GET` | `/table/{account}/{table}(PartitionKey='{pk}',RowKey='{rk}')` | Get entity |
| `GET` | `/table/{account}/{table}()` | Query entities |
| `DELETE` | `/table/{account}/{table}(PartitionKey='{pk}',RowKey='{rk}')` | Delete entity |

## Limitations

- No OData query filtering
- No batch operations
- No continuation tokens for large result sets
