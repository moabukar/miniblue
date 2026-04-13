/*
 * miniblue .NET example
 *
 * Start miniblue: docker run -p 4566:4566 moabukar/miniblue:latest
 * Run:            dotnet run
 *
 * Demonstrates Resource Groups, Key Vault, Blob Storage, and Cosmos DB
 * using plain HttpClient pointed at http://localhost:4566.
 */

Console.WriteLine("miniblue .NET example");
Console.WriteLine("=========================\n");

await new ResourceGroupExample().RunAsync();
await new KeyVaultExample().RunAsync();
await new BlobStorageExample().RunAsync();
await new CosmosDbExample().RunAsync();

Console.WriteLine("\nAll calls went to miniblue!");
