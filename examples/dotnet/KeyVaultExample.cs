using System.Net.Http.Json;
using System.Text.Json;

// -- Key Vault (HttpClient) ---------------------------------------------------
// Azure.Security.KeyVault.Secrets enforces HTTPS at the transport level and
// cannot be used with an http:// endpoint. miniblue's Key Vault API uses simple
// path-based routing, so HttpClient is used directly here.
sealed class KeyVaultExample
{
    private const string BaseUrl = "http://localhost:4566";

    private readonly HttpClient _http = new() { BaseAddress = new Uri(BaseUrl) };

    public async Task RunAsync()
    {
        await _http.PutAsJsonAsync(
            "/keyvault/myvault/secrets/db-password",
            new { value = "super-secret-dotnet-123" }
        );
        Console.WriteLine("\nSecret stored: db-password");

        var response = await _http.GetFromJsonAsync<JsonElement>(
            "/keyvault/myvault/secrets/db-password"
        );
        Console.WriteLine($"Secret value:  {response.GetProperty("value")}");
    }
}
