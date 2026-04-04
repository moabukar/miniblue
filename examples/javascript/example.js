/**
 * Azure SDK for JavaScript + miniblue example.
 *
 * Install: npm install @azure/identity @azure/arm-resources @azure/keyvault-secrets
 * Start miniblue: ./bin/miniblue
 * Run: node example.js
 */

const BASE_URL = "http://localhost:4566";

async function main() {
  console.log("miniblue JavaScript example");
  console.log("===========================\n");

  // Create resource group
  const rgResp = await fetch(
    `${BASE_URL}/subscriptions/sub1/resourcegroups/js-example-rg?api-version=2020-06-01`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ location: "eastus", tags: { env: "local" } }),
    }
  );
  const rg = await rgResp.json();
  console.log(`Resource Group: ${rg.name} (${rg.location})`);

  // Store a secret
  const secretResp = await fetch(
    `${BASE_URL}/keyvault/myvault/secrets/js-secret`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ value: "super-secret-from-js" }),
    }
  );
  const secret = await secretResp.json();
  console.log(`Secret: ${secret.id}`);

  // Read it back
  const getResp = await fetch(
    `${BASE_URL}/keyvault/myvault/secrets/js-secret`
  );
  const got = await getResp.json();
  console.log(`Secret value: ${got.value}`);

  // Upload a blob
  await fetch(`${BASE_URL}/blob/myaccount/mycontainer`, { method: "PUT" });
  await fetch(`${BASE_URL}/blob/myaccount/mycontainer/hello.txt`, {
    method: "PUT",
    headers: { "Content-Type": "text/plain" },
    body: "Hello from JavaScript!",
  });
  const blobResp = await fetch(
    `${BASE_URL}/blob/myaccount/mycontainer/hello.txt`
  );
  console.log(`Blob: ${await blobResp.text()}`);

  console.log("\nAll calls went to miniblue, not real Azure.");
}

main().catch(console.error);
