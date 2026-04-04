"""
Azure SDK for Python + miniblue example.

Install: pip install azure-mgmt-resource azure-core requests
Start miniblue: ./bin/miniblue
Trust cert: export SSL_CERT_FILE=~/.miniblue/cert.pem
Run: python example.py
"""
import os
import requests

MINIBLUE_URL = "http://localhost:4566"


def main():
    print("miniblue Python SDK example")
    print("===========================\n")

    # Create resource group
    resp = requests.put(
        f"{MINIBLUE_URL}/subscriptions/sub1/resourcegroups/python-rg",
        json={"location": "eastus", "tags": {"sdk": "python"}},
    )
    rg = resp.json()
    print(f"Resource Group: {rg['name']} ({rg['location']})")

    # List resource groups
    resp = requests.get(f"{MINIBLUE_URL}/subscriptions/sub1/resourcegroups")
    for rg in resp.json()["value"]:
        print(f"  - {rg['name']}")

    # Key Vault - store a secret
    resp = requests.put(
        f"{MINIBLUE_URL}/keyvault/myvault/secrets/db-password",
        json={"value": "super-secret-123"},
    )
    secret = resp.json()
    print(f"\nSecret stored: {secret['id']}")

    # Key Vault - read it back
    resp = requests.get(f"{MINIBLUE_URL}/keyvault/myvault/secrets/db-password")
    print(f"Secret value: {resp.json()['value']}")

    # Blob storage
    requests.put(f"{MINIBLUE_URL}/blob/myaccount/data")
    requests.put(
        f"{MINIBLUE_URL}/blob/myaccount/data/config.json",
        json={"database": "postgres://localhost:5432/mydb"},
    )
    resp = requests.get(f"{MINIBLUE_URL}/blob/myaccount/data/config.json")
    print(f"\nBlob content: {resp.json()}")

    # Cosmos DB
    requests.post(
        f"{MINIBLUE_URL}/cosmosdb/myaccount/dbs/app/colls/users/docs",
        json={"id": "user1", "name": "Mo", "role": "admin"},
    )
    resp = requests.get(
        f"{MINIBLUE_URL}/cosmosdb/myaccount/dbs/app/colls/users/docs/user1"
    )
    user = resp.json()
    print(f"\nCosmos doc: {user['name']} ({user['role']})")

    print("\nAll calls went to miniblue, not real Azure.")


if __name__ == "__main__":
    main()
