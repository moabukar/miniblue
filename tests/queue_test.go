package tests

import (
	"testing"
)

func TestStorageQueueConflict(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	url := ts.URL + "/queue/myaccount/myqueue"

	resp := doRequest(t, "PUT", url, "")
	resp.Body.Close()
	expectStatus(t, resp, 201)

	resp = doRequest(t, "PUT", url, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 409)
}

func TestStorageQueueMessageLifecycle(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/queue/myaccount/q1"

	doRequest(t, "PUT", base, "").Body.Close()

	// Send
	resp := doRequest(t, "POST", base+"/messages", `{"messageText":"hello"}`)
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Receive
	resp = doRequest(t, "GET", base+"/messages", "")
	defer resp.Body.Close()
	m := decodeJSON(t, resp)
	msgs := m["messages"].([]interface{})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	// Clear
	resp = doRequest(t, "DELETE", base+"/messages", "")
	resp.Body.Close()
	expectStatus(t, resp, 204)

	// Receive again - empty
	resp = doRequest(t, "GET", base+"/messages", "")
	m = decodeJSON(t, resp)
	msgs2, _ := m["messages"].([]interface{})
	if len(msgs2) != 0 {
		t.Fatalf("expected 0 messages after clear, got %d", len(msgs2))
	}
}
