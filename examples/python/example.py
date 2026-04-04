"""
Azure SDK for Python + miniblue example.

Install: pip install azure-identity azure-mgmt-resource azure-storage-blob azure-keyvault-secrets
Start miniblue: ./bin/miniblue
Run: python example.py
"""
import os

# Point Azure SDK at miniblue
os.environ["AZURE_AUTHORITY_HOST"] = "http://localhost:4566"
os.environ["AZURE_CLI_DISABLE_CONNECTION_VERIFICATION"] = "1"

from azure.identity import ClientSecretCredential
from azure.mgmt.resource import ResourceManagementClient

# Mock credentials (miniblue accepts anything)
credential = ClientSecretCredential(
    tenant_id="00000000-0000-0000-0000-000000000001",
    client_id="miniblue",
    client_secret="miniblue",
)

# Resource Management - point to miniblue
client = ResourceManagementClient(
    credential=credential,
    subscription_id="00000000-0000-0000-0000-000000000000",
    base_url="http://localhost:4566",
)

# Create a resource group
rg = client.resource_groups.create_or_update(
    "example-rg",
    {"location": "eastus", "tags": {"env": "local"}},
)
print(f"Created: {rg.name} in {rg.location}")

# List resource groups
for rg in client.resource_groups.list():
    print(f"  - {rg.name}")

print("\nDone! All calls went to miniblue, not real Azure.")
