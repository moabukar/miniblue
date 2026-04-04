package tests

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestBlobStorageCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/blob/myaccount"

	// Create container
	resp := doRequest(t, "PUT", base+"/mycontainer", "")
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Upload blob
	resp = doRequest(t, "PUT", base+"/mycontainer/hello.txt", "Hello World!")
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Download
	resp = doRequest(t, "GET", base+"/mycontainer/hello.txt", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	if buf.String() != "Hello World!" {
		t.Fatalf("expected 'Hello World!', got '%s'", buf.String())
	}

	// Verify content-length header
	if resp.Header.Get("Content-Length") != "12" {
		t.Fatalf("expected Content-Length=12, got %s", resp.Header.Get("Content-Length"))
	}
}

func TestBlobListContentLength(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/blob/myaccount/mycontainer"

	doRequest(t, "PUT", base, "").Body.Close()
	doRequest(t, "PUT", base+"/test.txt", "abcdef").Body.Close()

	resp := doRequest(t, "GET", base, "")
	defer resp.Body.Close()

	var result struct {
		Blobs []struct {
			Name       string `json:"name"`
			Properties struct {
				ContentLength string `json:"contentLength"`
			} `json:"properties"`
		} `json:"blobs"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Blobs) != 1 {
		t.Fatalf("expected 1 blob, got %d", len(result.Blobs))
	}
	if result.Blobs[0].Properties.ContentLength != "6" {
		t.Fatalf("expected contentLength=6, got %s", result.Blobs[0].Properties.ContentLength)
	}
}

func TestBlobNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "GET", ts.URL+"/blob/acct/container/nope.txt", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestBlobDelete(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/blob/myaccount/mycontainer"

	doRequest(t, "PUT", base, "").Body.Close()
	doRequest(t, "PUT", base+"/file.txt", "data").Body.Close()

	resp := doRequest(t, "DELETE", base+"/file.txt", "")
	resp.Body.Close()
	expectStatus(t, resp, 202)

	resp = doRequest(t, "GET", base+"/file.txt", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}
