using System.Net.Http.Json;
using System.Text.Json;

// -- Blob Storage (HttpClient) ------------------------------------------------
// Azure.Storage.Blobs v12 interprets path-style URIs as
// http://{host}/{accountName}/... — "blob" would be parsed as the account name
// when the service URI is http://localhost:4566/blob/dotnetaccount, which
// generates wrong blob URLs. miniblue's blob API uses simple path-based routing,
// so HttpClient is used directly here.
sealed class BlobStorageExample
{
    private const string BaseUrl = "http://localhost:4566";

    private readonly HttpClient _http = new() { BaseAddress = new Uri(BaseUrl) };

    public async Task RunAsync()
    {
        await _http.PutAsync("/blob/dotnetaccount/data", null);
        Console.WriteLine("\nContainer created: data");

        var payload = """{"database":"postgres://localhost:5432/mydb","env":"local"}""";
        await _http.PutAsync(
            "/blob/dotnetaccount/data/config.json",
            new StringContent(payload, System.Text.Encoding.UTF8, "application/json")
        );
        Console.WriteLine("Blob uploaded: config.json");

        var response = await _http.GetFromJsonAsync<JsonElement>(
            "/blob/dotnetaccount/data/config.json"
        );
        Console.WriteLine($"Blob content:  {response}");
    }
}
