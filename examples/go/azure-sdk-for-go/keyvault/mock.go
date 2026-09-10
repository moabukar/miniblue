package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// EmulatorBridgePolicy handles all local emulator body quirks and response sanitization safely
type EmulatorBridgePolicy struct{}

func (p EmulatorBridgePolicy) Do(req *policy.Request) (*http.Response, error) {

	fmt.Printf("%s %s\n", req.Raw().Method, getFullURL(req.Raw()))

	// 1. REWRITE REQUEST: Fix trailing slash if present
	if strings.HasSuffix(req.Raw().URL.Path, "/") {
		req.Raw().URL.Path = strings.TrimSuffix(req.Raw().URL.Path, "/")
	}

	// 2. SAFE BODY PREPARATION: If a body exists, read it and explicitly reset it
	// before every single network try so it never empties on a challenge retry loop.
	if stream := req.Body(); stream != nil {
		bodyBytes, err := io.ReadAll(stream)
		_ = stream.Close()

		if err == nil {
			req.SetBody(streaming.NopCloser(bytes.NewReader(bodyBytes)), "application/json")
			req.Raw().GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(bodyBytes)), nil
			}
			req.Raw().Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.Raw().ContentLength = int64(len(bodyBytes))
			req.Raw().Header.Set("Content-Length", strconv.Itoa(len(bodyBytes)))
			req.Raw().Header.Set("Content-Type", "application/json")
		}
	}

	// 3. PROCEED TO WIRE
	resp, err := req.Next()
	if err != nil {
		return nil, err
	}

	// 4. SANITIZE RESPONSE: Inject precisely structured fake challenge header if missing
	if resp.Header.Get("WWW-Authenticate") == "" {
		mockHeader := `Bearer authorization="https://microsoftonline.com", resource="https://vault.azure.net"`
		resp.Header.Set("WWW-Authenticate", mockHeader)
	}

	return resp, nil
}

func getFullURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	// Combines scheme, host, path, and query parameters
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.RequestURI())
}

// FakeChallengePolicy injects a mock WWW-Authenticate header if missing
type FakeChallengePolicy struct{}

func (p FakeChallengePolicy) Do(req *policy.Request) (*http.Response, error) {
	resp, err := req.Next()
	if err != nil {
		return nil, err
	}

	// If the server returns a 401 or a status without the challenge header, inject a fake one
	if resp.Header.Get("WWW-Authenticate") == "" {
		mockHeader := `Bearer authorization="https://microsoftonline.com/00000000-0000-0000-0000-000000000000", resource="https://azure.net"`
		resp.Header.Set("WWW-Authenticate", mockHeader)
	}
	return resp, nil
}

// FakeTokenCredential creates a dummy token credential for local mock environments
type FakeTokenCredential struct{}

func (f FakeTokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{
		Token:     "mock-token-payload",
		ExpiresOn: time.Now().Add(1 * time.Hour),
	}, nil
}
