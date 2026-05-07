package com.miniblue.example;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpRequest.BodyPublishers;
import java.net.http.HttpResponse.BodyHandlers;

// Blob Storage (HttpClient)
// The Azure Blob Storage SDK interprets path-style URIs as
// http://{host}/{accountName}/... - "blob" would be parsed as the account name
// when the service URI is http://localhost:4566/blob/javaaccount, which generates
// wrong blob URLs. miniblue's blob API uses simple path-based routing, so
// HttpClient is used directly here.
final class BlobStorageExample {
    private static final String BASE_URL = "http://localhost:4566";

    private final HttpClient http = HttpClient.newHttpClient();

    void run() throws Exception {
        HttpRequest createContainer = HttpRequest.newBuilder()
                .uri(URI.create(BASE_URL + "/blob/javaaccount/data"))
                .PUT(BodyPublishers.noBody())
                .build();
        http.send(createContainer, BodyHandlers.discarding());
        System.out.println("\nContainer created: data");

        String payload = "{\"database\":\"postgres://localhost:5432/mydb\",\"env\":\"local\"}";
        HttpRequest uploadBlob = HttpRequest.newBuilder()
                .uri(URI.create(BASE_URL + "/blob/javaaccount/data/config.json"))
                .header("Content-Type", "application/json")
                .PUT(BodyPublishers.ofString(payload))
                .build();
        http.send(uploadBlob, BodyHandlers.discarding());
        System.out.println("Blob uploaded: config.json");

        HttpRequest getBlob = HttpRequest.newBuilder()
                .uri(URI.create(BASE_URL + "/blob/javaaccount/data/config.json"))
                .GET()
                .build();
        String body = http.send(getBlob, BodyHandlers.ofString()).body();
        System.out.println("Blob content:  " + body);
    }
}
