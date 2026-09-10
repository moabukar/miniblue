package mock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"io"
	"log"
	"net/http"
)

type LocalMockPolicy struct {
	TargetHost string
	UseHTTPS   bool
}

func (p *LocalMockPolicy) Do(req *policy.Request) (*http.Response, error) {
	fmt.Printf("[Pipeline Interceptor] Original Destination: %s\n", req.Raw().URL.String())
	if p.UseHTTPS {
		req.Raw().URL.Scheme = "https"
	} else {
		req.Raw().URL.Scheme = "http"
	}
	req.Raw().URL.Host = p.TargetHost
	req.Raw().Host = p.TargetHost
	fmt.Printf("[Pipeline Interceptor] Rerouted Destination: %s\n", req.Raw().URL.String())
	return req.Next()
}

// FakeChallengePolicy injects a mock WWW-Authenticate header if missing
type FakeChallengePolicy struct{}

func (p FakeChallengePolicy) Do(req *policy.Request) (*http.Response, error) {

	resp, err := req.Next()
	if err != nil {
		return nil, err
	}

	// 2. SANITIZE RESPONSE BODY: Fix string timestamp to integer Unix epoch transformation
	if resp.Body != nil && resp.StatusCode == http.StatusOK {

		bodyBytes, err := io.ReadAll(resp.Body)
		//fmt.Printf("%s", bodyBytes)

		_ = resp.Body.Close() // Close original body
		if err == nil {

			var prettyJSON bytes.Buffer

			e := json.Indent(&prettyJSON, bodyBytes, "", "    ")
			if e != nil {
				log.Fatalf("Error indenting: %s", e)
			}

			fmt.Println(prettyJSON.String())

			// Restore original body if unmarshaling failed
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	return resp, nil
}
