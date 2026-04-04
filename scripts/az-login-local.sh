#!/bin/bash
# Helper script to configure az CLI for miniblue
# This bypasses MSAL authentication by writing a mock token directly

set -e

PORT="${LOCAL_AZURE_PORT:-4566}"
BASE="http://localhost:${PORT}"

echo "Configuring az CLI for miniblue on ${BASE}..."

# Register the cloud (if not already)
az cloud set --name AzureCloud 2>/dev/null || true
az cloud unregister --name miniblue 2>/dev/null || true
az cloud register --name miniblue \
  --endpoint-resource-manager "${BASE}" \
  --endpoint-active-directory "${BASE}" \
  --endpoint-active-directory-resource-id "${BASE}" \
  --endpoint-active-directory-graph-resource-id "${BASE}" 2>/dev/null

az cloud set --name miniblue

# Write a mock access token directly into az CLI's token cache
AZURE_DIR="${HOME}/.azure"
mkdir -p "${AZURE_DIR}"

# Create a mock profile
cat > "${AZURE_DIR}/azureProfile.json" << PROFILE
{
  "installationId": "miniblue-mock",
  "subscriptions": [
    {
      "id": "00000000-0000-0000-0000-000000000000",
      "name": "miniblue",
      "state": "Enabled",
      "user": {
        "name": "miniblue@localhost",
        "type": "servicePrincipal"
      },
      "isDefault": true,
      "tenantId": "00000000-0000-0000-0000-000000000001",
      "environmentName": "miniblue",
      "homeTenantId": "00000000-0000-0000-0000-000000000001",
      "managedByTenants": []
    }
  ]
}
PROFILE

echo ""
echo "Done! az CLI is now configured for miniblue."
echo ""
echo "Test it:"
echo "  az group create --name myRG --location eastus"
echo "  az group list"
echo ""
echo "To switch back to real Azure:"
echo "  az cloud set --name AzureCloud"
echo "  az login"
