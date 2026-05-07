package com.miniblue.example;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpRequest.BodyPublishers;
import java.net.http.HttpResponse.BodyHandlers;
import java.util.Map;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;

// Cosmos DB (HttpClient)
// The Azure Cosmos SDK requires a Cosmos-compatible discovery endpoint, which
// miniblue does not expose. miniblue's Cosmos API uses simple path-based
// routing, so HttpClient is used directly here.
final class CosmosDbExample {
    private static final String BASE_URL = "http://localhost:4566";

    private final HttpClient http = HttpClient.newHttpClient();
    private final ObjectMapper json = new ObjectMapper();

    void run() throws Exception {
        String body = json.writeValueAsString(Map.of(
                "id", "user1",
                "name", "Mo",
                "role", "admin"
        ));
        HttpRequest insert = HttpRequest.newBuilder()
                .uri(URI.create(BASE_URL + "/cosmosdb/myaccount/dbs/app/colls/users/docs"))
                .header("Content-Type", "application/json")
                .POST(BodyPublishers.ofString(body))
                .build();
        http.send(insert, BodyHandlers.discarding());
        System.out.println("\nCosmos doc created: user1");

        HttpRequest read = HttpRequest.newBuilder()
                .uri(URI.create(BASE_URL + "/cosmosdb/myaccount/dbs/app/colls/users/docs/user1"))
                .GET()
                .build();
        String responseBody = http.send(read, BodyHandlers.ofString()).body();
        JsonNode doc = json.readTree(responseBody);
        System.out.printf("Cosmos doc: %s (%s)%n",
                doc.get("name").asText(), doc.get("role").asText());
    }
}
