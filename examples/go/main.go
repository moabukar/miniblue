package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	resp, err := http.NewRequest("PUT",
		baseURL+"/subscriptions/sub1/resourcegroups/go-example-rg?api-version=2020-06-01",
		bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	resp.Header.Set("Content-Type", "application/json")
	r, _ := http.DefaultClient.Do(resp)
	defer r.Body.Close()
	data, _ := io.ReadAll(r.Body)
	fmt.Printf("Resource Group: %s\n", string(data))

	// Store a secret
	secret, _ := json.Marshal(map[string]string{"value": "my-api-key-123"})
	r2, _ := http.Post(baseURL+"/keyvault/myvault/secrets/api-key",
		"application/json", bytes.NewReader(secret))
	// Note: keyvault uses PUT, not POST
	r2.Body.Close()

	req, _ := http.NewRequest("PUT", baseURL+"/keyvault/myvault/secrets/api-key",
		bytes.NewReader(secret))
	req.Header.Set("Content-Type", "application/json")
	r3, _ := http.DefaultClient.Do(req)
	defer r3.Body.Close()
	secretData, _ := io.ReadAll(r3.Body)
	fmt.Printf("Secret: %s\n", string(secretData))

	fmt.Println("\nAll calls went to miniblue, not real Azure.")
}
