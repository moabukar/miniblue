using System.Net.Http.Json;
using System.Text.Json;

// ── Resource Groups (HttpClient) ──────────────────────────────────────────────
// Azure.ResourceManager requires HTTPS for Bearer token auth and performs ARM
// service discovery that miniblue does not fully support. miniblue''s resource
// group API uses simple path-based routing, so HttpClient is used directly here.
sealed class ResourceGroupExample
{
    private const string BaseUrl = "http://localhost:4566";
    private const string SubId = "00000000-0000-0000-0000-000000000000";

    private readonly HttpClient _http = new() { BaseAddress = new Uri(BaseUrl) };

    public async Task RunAsync()
    {
        // Create resource group
        var resp = await _http.PutAsJsonAsync(
            $"/subscriptions/{SubId}/resourcegroups/dotnet-rg",
            new { location = "eastus", tags = new { env = "local", sdk = "dotnet" } });
        var rg = await resp.Content.ReadFromJsonAsync<JsonElement>();
        Console.WriteLine($"Resource Group: {rg.GetProperty("name")} ({rg.GetProperty("location")})");

        // List all resource groups
        var list = await _http.GetFromJsonAsync<JsonElement>(
            $"/subscriptions/{SubId}/resourcegroups");
        Console.WriteLine("All resource groups:");
        foreach (var item in list.GetProperty("value").EnumerateArray())
            Console.WriteLine($"  - {item.GetProperty("name")}");
    }
}
