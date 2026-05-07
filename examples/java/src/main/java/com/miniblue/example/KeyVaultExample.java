package com.miniblue.example;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpRequest.BodyPublishers;
import java.net.http.HttpResponse.BodyHandlers;
import java.util.Map;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;

// Key Vault (HttpClient)
// The Azure Key Vault SDK enforces HTTPS at the transport level and cannot be
// used with an http:// endpoint. miniblue's Key Vault API uses simple path-based
// routing, so HttpClient is used directly here.
final class KeyVaultExample {
    private static final String BASE_URL = "http://localhost:4566";

    private final HttpClient http = HttpClient.newHttpClient();
    private final ObjectMapper json = new ObjectMapper();

    void run() throws Exception {
        String body = json.writeValueAsString(Map.of("value", "super-secret-java-123"));
        HttpRequest setReq = HttpRequest.newBuilder()
                .uri(URI.create(BASE_URL + "/keyvault/myvault/secrets/db-password"))
                .header("Content-Type", "application/json")
                .PUT(BodyPublishers.ofString(body))
                .build();
        http.send(setReq, BodyHandlers.discarding());
        System.out.println("\nSecret stored: db-password");

        HttpRequest getReq = HttpRequest.newBuilder()
                .uri(URI.create(BASE_URL + "/keyvault/myvault/secrets/db-password"))
                .GET()
                .build();
        String response = http.send(getReq, BodyHandlers.ofString()).body();
        JsonNode secret = json.readTree(response);
        System.out.println("Secret value:  " + secret.get("value").asText());
    }
}
