using System.Net.Http.Json;
using System.Text.Json;

// ── Cosmos DB (HttpClient) ────────────────────────────────────────────────────
// Microsoft.Azure.Cosmos requires a Cosmos-compatible discovery endpoint, which
// miniblue does not expose. miniblue's Cosmos API uses simple path-based routing,
// so HttpClient is used directly here.
sealed class CosmosDbExample
{
    private const string BaseUrl = "http://localhost:4566";

    private readonly HttpClient _http = new() { BaseAddress = new Uri(BaseUrl) };

    public async Task RunAsync()
    {
        var doc = new
        {
            id = "user1",
            name = "Mo",
            role = "admin",
        };
        await _http.PostAsJsonAsync("/cosmosdb/myaccount/dbs/app/colls/users/docs", doc);
        Console.WriteLine("\nCosmos doc created: user1");

        var response = await _http.GetFromJsonAsync<JsonElement>(
            "/cosmosdb/myaccount/dbs/app/colls/users/docs/user1"
        );
        Console.WriteLine(
            $"Cosmos doc: {response.GetProperty("name")} ({response.GetProperty("role")})"
        );
    }
}
