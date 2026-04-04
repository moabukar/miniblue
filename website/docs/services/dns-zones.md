# DNS Zones

miniblue emulates Azure DNS zones and record sets. Creating a zone automatically generates SOA and NS records, matching Azure's behaviour.

## API endpoints

### Zones

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/dnsZones/{zone}` | Create or update |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/dnsZones/{zone}` | Get |
| `DELETE` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/dnsZones/{zone}` | Delete |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/dnsZones` | List |

### Record Sets

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `.../dnsZones/{zone}/{recordType}/{name}` | Create or update |
| `GET` | `.../dnsZones/{zone}/{recordType}/{name}` | Get |
| `DELETE` | `.../dnsZones/{zone}/{recordType}/{name}` | Delete |

## Create a DNS zone

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/dnsZones/example.com?api-version=2020-06-01" \
  -H "Content-Type: application/json" \
  -d '{}'
```

Response (`201 Created`):

```json
{
  "id": "/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/dnsZones/example.com?api-version=2020-06-01",
  "name": "example.com",
  "type": "Microsoft.Network/dnsZones",
  "location": "global",
  "properties": {
    "numberOfRecordSets": 2,
    "nameServers": [
      "ns1-01.azure-dns.com.",
      "ns2-01.azure-dns.net."
    ]
  }
}
```

!!! note
    miniblue automatically creates SOA and NS records for new zones, just like Azure does.

## Get a DNS zone

```bash
curl "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/dnsZones/example.com?api-version=2020-06-01"
```

## List DNS zones

```bash
curl "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/dnsZones?api-version=2020-06-01"
```

## Create a record set

### A record

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/dnsZones/example.com/A/www?api-version=2020-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "properties": {
      "TTL": 300,
      "ARecords": [
        {"ipv4Address": "1.2.3.4"},
        {"ipv4Address": "5.6.7.8"}
      ]
    }
  }'
```

Response (`201 Created`):

```json
{
  "id": "/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/dnsZones/example.com/A/www?api-version=2020-06-01",
  "name": "www",
  "type": "Microsoft.Network/dnsZones/A",
  "properties": {
    "TTL": 300,
    "ARecords": [
      {"ipv4Address": "1.2.3.4"},
      {"ipv4Address": "5.6.7.8"}
    ]
  }
}
```

### CNAME record

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/dnsZones/example.com/CNAME/docs?api-version=2020-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "properties": {
      "TTL": 3600,
      "CNAMERecord": {
        "cname": "docs.example.com"
      }
    }
  }'
```

### MX record

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/dnsZones/example.com/MX/@?api-version=2020-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "properties": {
      "TTL": 3600,
      "MXRecords": [
        {"preference": 10, "exchange": "mail.example.com."}
      ]
    }
  }'
```

### TXT record

```bash
curl -X PUT "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/dnsZones/example.com/TXT/@?api-version=2020-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "properties": {
      "TTL": 3600,
      "TXTRecords": [
        {"value": ["v=spf1 include:example.com ~all"]}
      ]
    }
  }'
```

## Get a record set

```bash
curl "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/dnsZones/example.com/A/www?api-version=2020-06-01"
```

## Delete a record set

```bash
curl -X DELETE "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/dnsZones/example.com/A/www?api-version=2020-06-01"
```

## Delete a DNS zone

```bash
curl -X DELETE "http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.Network/dnsZones/example.com?api-version=2020-06-01"
```

Deleting a zone also deletes all its record sets.

## Supported record types

miniblue accepts any record type in the URL path. The `properties` payload is stored as-is. Common types:

| Type | Example property |
|------|-----------------|
| `A` | `ARecords: [{ipv4Address: "1.2.3.4"}]` |
| `AAAA` | `AAAARecords: [{ipv6Address: "::1"}]` |
| `CNAME` | `CNAMERecord: {cname: "target.example.com"}` |
| `MX` | `MXRecords: [{preference: 10, exchange: "mail.example.com."}]` |
| `TXT` | `TXTRecords: [{value: ["v=spf1 ..."]}]` |
| `NS` | `NSRecords: [{nsdname: "ns1.example.com."}]` |
| `SOA` | Auto-created on zone creation |

## Terraform

```hcl
resource "azurerm_dns_zone" "example" {
  name                = "example.local"
  resource_group_name = azurerm_resource_group.example.name
}
```

See the [Terraform guide](../guides/terraform.md) for full provider configuration.
