package auth

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"time"
)

type AnonymousMockCredential struct{}

func (c *AnonymousMockCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "mock-token-value", ExpiresOn: time.Now().Add(24 * time.Hour)}, nil
}
