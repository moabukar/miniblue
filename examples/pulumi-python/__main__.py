"""
miniblue Pulumi Python example using dynamic providers.

Uses Pulumi dynamic providers to make HTTP calls directly to miniblue's
REST API (http://localhost:4566) without needing real Azure credentials.

Resources created:
- ResourceGroup
- KeyVault secret
- Blob container + blob
- CosmosDB document
"""

import requests
import pulumi
from pulumi import Input, Output
from pulumi.dynamic import Resource, ResourceProvider, CreateResult

MINIBLUE_URL = "http://localhost:4566"
SUBSCRIPTION = "sub1"


class ResourceGroupProvider(ResourceProvider):
    def create(self, props):
        name = props["name"]
        location = props.get("location", "eastus")
        tags = props.get("tags", {})
        resp = requests.put(
            f"{MINIBLUE_URL}/subscriptions/{SUBSCRIPTION}/resourcegroups/{name}",
            json={"location": location, "tags": tags},
        )
        resp.raise_for_status()
        data = resp.json()
        outs = dict(props)
        outs["provisioning_state"] = data.get("properties", {}).get("provisioningState", "Succeeded")
        return CreateResult(id_=name, outs=outs)

    def delete(self, id, props):
        requests.delete(
            f"{MINIBLUE_URL}/subscriptions/{SUBSCRIPTION}/resourcegroups/{id}"
        )


class ResourceGroup(Resource):
    def __init__(self, resource_name, name: Input[str], location: Input[str] = "eastus", tags: Input[dict] = None, opts=None):
        props = {
            "name": name,
            "location": location,
            "tags": tags or {},
            "provisioning_state": None,
        }
        super().__init__(ResourceGroupProvider(), resource_name, props, opts)
        self.name = self.__dict__.get("name") or Output.from_input(name)
        self.location = self.__dict__.get("location") or Output.from_input(location)
        self.provisioning_state = self.__dict__.get("provisioning_state")


class KeyVaultSecretProvider(ResourceProvider):
    def create(self, props):
        vault = props["vault"]
        secret_name = props["secret_name"]
        value = props["value"]
        resp = requests.put(
            f"{MINIBLUE_URL}/keyvault/{vault}/secrets/{secret_name}",
            json={"value": value},
        )
        resp.raise_for_status()
        data = resp.json()
        secret_id = data.get("id", f"https://{vault}.vault.azure.net/secrets/{secret_name}")
        outs = dict(props)
        outs["secret_id"] = secret_id
        return CreateResult(id_=f"{vault}/{secret_name}", outs=outs)

    def delete(self, id, props):
        vault = props["vault"]
        secret_name = props["secret_name"]
        requests.delete(f"{MINIBLUE_URL}/keyvault/{vault}/secrets/{secret_name}")


class KeyVaultSecret(Resource):
    def __init__(self, resource_name, vault: Input[str], secret_name: Input[str], value: Input[str], opts=None):
        props = {
            "vault": vault,
            "secret_name": secret_name,
            "value": value,
            "secret_id": None,
        }
        super().__init__(KeyVaultSecretProvider(), resource_name, props, opts)
        self.secret_id = self.__dict__.get("secret_id")


class BlobContainerProvider(ResourceProvider):
    def create(self, props):
        account = props["account"]
        container = props["container"]
        resp = requests.put(f"{MINIBLUE_URL}/blob/{account}/{container}")
        resp.raise_for_status()
        return CreateResult(id_=f"{account}/{container}", outs=dict(props))

    def delete(self, id, props):
        account = props["account"]
        container = props["container"]
        requests.delete(f"{MINIBLUE_URL}/blob/{account}/{container}")


class BlobContainer(Resource):
    def __init__(self, resource_name, account: Input[str], container: Input[str], opts=None):
        props = {"account": account, "container": container}
        super().__init__(BlobContainerProvider(), resource_name, props, opts)


class BlobProvider(ResourceProvider):
    def create(self, props):
        account = props["account"]
        container = props["container"]
        blob_name = props["blob_name"]
        content = props.get("content", {})
        resp = requests.put(
            f"{MINIBLUE_URL}/blob/{account}/{container}/{blob_name}",
            json=content,
        )
        resp.raise_for_status()
        return CreateResult(id_=f"{account}/{container}/{blob_name}", outs=dict(props))

    def delete(self, id, props):
        account = props["account"]
        container = props["container"]
        blob_name = props["blob_name"]
        requests.delete(f"{MINIBLUE_URL}/blob/{account}/{container}/{blob_name}")


class Blob(Resource):
    def __init__(self, resource_name, account: Input[str], container: Input[str], blob_name: Input[str], content: Input[dict] = None, opts=None):
        props = {
            "account": account,
            "container": container,
            "blob_name": blob_name,
            "content": content or {},
        }
        super().__init__(BlobProvider(), resource_name, props, opts)
        self.blob_name_out = self.__dict__.get("blob_name") or Output.from_input(blob_name)


class CosmosDBDocumentProvider(ResourceProvider):
    def create(self, props):
        account = props["account"]
        db = props["db"]
        collection = props["collection"]
        document = props["document"]
        resp = requests.post(
            f"{MINIBLUE_URL}/cosmosdb/{account}/dbs/{db}/colls/{collection}/docs",
            json=document,
        )
        resp.raise_for_status()
        data = resp.json()
        doc_id = document.get("id", data.get("id", "unknown"))
        outs = dict(props)
        outs["doc_id"] = doc_id
        return CreateResult(id_=f"{account}/{db}/{collection}/{doc_id}", outs=outs)

    def delete(self, id, props):
        account = props["account"]
        db = props["db"]
        collection = props["collection"]
        doc_id = props.get("doc_id", props["document"].get("id"))
        requests.delete(
            f"{MINIBLUE_URL}/cosmosdb/{account}/dbs/{db}/colls/{collection}/docs/{doc_id}"
        )


class CosmosDBDocument(Resource):
    def __init__(self, resource_name, account: Input[str], db: Input[str], collection: Input[str], document: Input[dict], opts=None):
        props = {
            "account": account,
            "db": db,
            "collection": collection,
            "document": document,
            "doc_id": None,
        }
        super().__init__(CosmosDBDocumentProvider(), resource_name, props, opts)
        self.doc_id = self.__dict__.get("doc_id")


# ------- Stack resources -------

rg = ResourceGroup(
    "example-rg",
    name="pulumi-rg",
    location="eastus",
    tags={"managed-by": "pulumi", "env": "dev"},
)

secret = KeyVaultSecret(
    "db-password",
    vault="pulumi-vault",
    secret_name="db-password",
    value="pulumi-secret-42",
    opts=pulumi.ResourceOptions(depends_on=[rg]),
)

container = BlobContainer(
    "app-container",
    account="pulumiaccount",
    container="appdata",
    opts=pulumi.ResourceOptions(depends_on=[rg]),
)

blob = Blob(
    "config-blob",
    account="pulumiaccount",
    container="appdata",
    blob_name="config.json",
    content={"env": "dev", "version": "1.0", "managed_by": "pulumi"},
    opts=pulumi.ResourceOptions(depends_on=[container]),
)

cosmos_doc = CosmosDBDocument(
    "user-doc",
    account="pulumiaccount",
    db="appdb",
    collection="users",
    document={"id": "user1", "name": "Pulumi User", "role": "admin"},
    opts=pulumi.ResourceOptions(depends_on=[rg]),
)

# Export outputs
pulumi.export("resource_group", "pulumi-rg")
pulumi.export("secret_id", "https://pulumi-vault.vault.azure.net/secrets/db-password")
pulumi.export("blob_id", "config.json")
pulumi.export("cosmos_doc_id", "user1")
pulumi.export("miniblue_url", MINIBLUE_URL)
