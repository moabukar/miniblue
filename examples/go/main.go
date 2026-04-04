package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const baseURL = "http://localhost:4566"

func main() {
	fmt.Println("miniblue Go SDK example")
	fmt.Println("=======================")

	// Create resource group
	body, _ := json.Marshal(map[string]interface{}{
		"location": "eastus",
		"tags":     map[string]string{"env": "local"},
	})
	req, err := http.NewRequest("PUT",
		baseURL+"/subscriptions/sub1/resourcegroups/go-example-rg",
		bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Body.Close()
	data, _ := io.ReadAll(r.Body)
	fmt.Printf("Resource Group: %s\n", string(data))

	// Store a secret
	secret, _ := json.Marshal(map[string]string{"value": "my-api-key-123"})
	kvReq, err := http.NewRequest("PUT", baseURL+"/keyvault/myvault/secrets/api-key",
		bytes.NewReader(secret))
	if err != nil {
		log.Fatal(err)
	}
	kvReq.Header.Set("Content-Type", "application/json")
	r2, err := http.DefaultClient.Do(kvReq)
	if err != nil {
		log.Fatal(err)
	}
	defer r2.Body.Close()
	secretData, _ := io.ReadAll(r2.Body)
	fmt.Printf("Secret: %s\n", string(secretData))

	fmt.Println("\nAll calls went to miniblue, not real Azure.")
}
