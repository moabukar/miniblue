package storage

import (
	"az-storage-mock/internal/auth"
	"az-storage-mock/internal/mock"
	"crypto/tls"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v4"
	"net/http"
	"net/url"
)

type Clients struct {
	Factory        *armstorage.ClientFactory
	BlobContainers *armstorage.BlobContainersClient
	Accounts       *armstorage.AccountsClient
	BlobServices   *armstorage.BlobServicesClient
}

func NewClients(subscriptionID, endpoint string) (*Clients, error) {

	u, e := url.Parse(endpoint)

	if e != nil {
		return nil, e
	}

	cred := &auth.AnonymousMockCredential{}

	transport := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// Prevents the SDK from validating that the host ends with 'vault.azure.net'
	opts := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			// Allow HTTP connections if your emulator doesn't use SSL/TLS
			InsecureAllowCredentialWithHTTP: true,
			Transport:                       transport,
			PerRetryPolicies:                []policy.Policy{mock.FakeChallengePolicy{}},
			PerCallPolicies: []policy.Policy{
				&mock.LocalMockPolicy{
					TargetHost: u.Host,
					UseHTTPS:   u.Scheme == "https",
				},
			},
		},
	}

	f, e := armstorage.NewClientFactory(subscriptionID, cred, opts)
	if e != nil {
		return nil, e
	}

	return &Clients{
		f,
		f.NewBlobContainersClient(),
		f.NewAccountsClient(),
		f.NewBlobServicesClient(),
	}, nil
}
