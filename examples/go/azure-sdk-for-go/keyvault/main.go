package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

func main() {

	keyVaultName := "myvault"
	secretName := "db-password"
	secretValue := "secretValue"

	fmt.Printf("Data:\n Key vault: %s\n Secret name: %s\n Value: %s\n\n", keyVaultName, secretName, secretValue)

	dialer := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}

	cred := FakeTokenCredential{}
	options := &azsecrets.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport:                       &http.Client{Transport: dialer},
			InsecureAllowCredentialWithHTTP: true,
			PerCallPolicies:                 []policy.Policy{EmulatorBridgePolicy{}},
			PerRetryPolicies:                []policy.Policy{FakeChallengePolicy{}},
		},
		DisableChallengeResourceVerification: true,
	}

	emulatorURL := fmt.Sprintf("https://localhost:4567/keyvault/%s/", keyVaultName)
	client, err := azsecrets.NewClient(emulatorURL, cred, options)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	// List secrets to prime the authentication challenge cache
	fmt.Println("Listing secrets...")
	pager := client.NewListSecretsPager(nil)
	for pager.More() {
		page, err := pager.NextPage(context.TODO())
		if err != nil {
			log.Fatal(err)
		}
		for _, secret := range page.Value {
			fmt.Printf("Secret ID: %s\n", *secret.ID)
		}
	}
	fmt.Println("")

	// Create
	params := azsecrets.SetSecretParameters{
		Value: to.Ptr(secretValue),
	}

	fmt.Printf("Creating secret %s\n", secretName)
	resp, err := client.SetSecret(context.TODO(), secretName, params, nil)
	if err != nil {
		log.Fatalf("failed to create a secret: %v", err)
	}

	fmt.Println("Secret created successfully!")
	fmt.Printf(" Secret Value: %v\n", *resp.Value)
	fmt.Printf(" Created: %v\n", *resp.Attributes.Created)
	fmt.Printf(" Updated: %v\n", *resp.Attributes.Updated)
	fmt.Printf(" Enabled: %v\n\n", *resp.Attributes.Enabled)

	// Retrieve
	fmt.Println("Retrieving secret...")
	resp1, err := client.GetSecret(context.TODO(), secretName, "", nil)
	if err != nil {
		log.Fatalf("Secret retrieval failed: %v", err)
	}
	fmt.Printf("Success! secretValue: %s\n", *resp1.Value)
	fmt.Printf(" Created: %v\n", *resp1.Attributes.Created)
	fmt.Printf(" Updated: %v\n", *resp1.Attributes.Updated)
	fmt.Printf(" Enabled: %v\n\n", *resp1.Attributes.Enabled)

	// Delete
	_, err = client.DeleteSecret(context.TODO(), secretName, nil)
	if err != nil {
		log.Fatalf("failed to delete secret: %v", err)
	}

	fmt.Printf("Secret %s deletion initiated.\n", secretName)
}
