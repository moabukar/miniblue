package com.miniblue.example;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpRequest.BodyPublishers;
import java.net.http.HttpResponse.BodyHandlers;
import java.util.Map;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;

// Resource Groups (HttpClient)
// The Azure ResourceManager SDK requires HTTPS for Bearer token auth and
// performs ARM service discovery that miniblue does not fully support. miniblue's
// resource group API uses simple path-based routing, so HttpClient is used
// directly here.
final class ResourceGroupExample {
    private static final String BASE_URL = "http://localhost:4566";
    private static final String SUB_ID = "00000000-0000-0000-0000-000000000000";

    private final HttpClient http = HttpClient.newHttpClient();
    private final ObjectMapper json = new ObjectMapper();

    void run() throws Exception {
        String body = json.writeValueAsString(Map.of(
                "location", "eastus",
                "tags", Map.of("env", "local", "sdk", "java")
        ));
        HttpRequest createReq = HttpRequest.newBuilder()
                .uri(URI.create(BASE_URL + "/subscriptions/" + SUB_ID + "/resourcegroups/java-rg"))
                .header("Content-Type", "application/json")
                .PUT(BodyPublishers.ofString(body))
                .build();
        String createBody = http.send(createReq, BodyHandlers.ofString()).body();
        JsonNode rg = json.readTree(createBody);
        System.out.printf("Resource Group: %s (%s)%n",
                rg.get("name").asText(), rg.get("location").asText());

        HttpRequest listReq = HttpRequest.newBuilder()
                .uri(URI.create(BASE_URL + "/subscriptions/" + SUB_ID + "/resourcegroups"))
                .GET()
                .build();
        String listBody = http.send(listReq, BodyHandlers.ofString()).body();
        JsonNode list = json.readTree(listBody);
        System.out.println("All resource groups:");
        for (JsonNode item : list.get("value")) {
            System.out.println("  - " + item.get("name").asText());
        }
    }
}
