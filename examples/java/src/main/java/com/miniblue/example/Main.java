package com.miniblue.example;

/*
 * miniblue Java example
 *
 * Start miniblue: docker run -p 4566:4566 moabukar/miniblue:latest
 * Run:            mvn -q compile exec:java
 *
 * Demonstrates Resource Groups, Key Vault, Blob Storage, and Cosmos DB
 * using java.net.http.HttpClient pointed at http://localhost:4566.
 */
public final class Main {
    public static void main(String[] args) throws Exception {
        System.out.println("miniblue Java example");
        System.out.println("=========================\n");

        new ResourceGroupExample().run();
        new KeyVaultExample().run();
        new BlobStorageExample().run();
        new CosmosDbExample().run();

        System.out.println("\nAll calls went to miniblue!");
    }
}
