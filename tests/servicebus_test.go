package tests

import (
	"testing"
)

func TestServiceBusQueueConflict(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	url := ts.URL + "/servicebus/myns/queues/orders"

	// First create
	resp := doRequest(t, "PUT", url, "")
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Second create - should 409
	resp = doRequest(t, "PUT", url, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 409)

	e := decodeError(t, resp)
	if e.Error.Code != "Conflict" {
		t.Fatalf("expected Conflict, got %s", e.Error.Code)
	}
}

func TestServiceBusSendToNonexistentQueue(t *testing.T) {
	ts := setupServer()
	defer ts.Close()

	resp := doRequest(t, "POST", ts.URL+"/servicebus/myns/queues/nope/messages", `{"body":"test"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestServiceBusSendAndReceive(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/servicebus/myns/queues/orders"

	doRequest(t, "PUT", base, "").Body.Close()

	// Send 2 messages
	doRequest(t, "POST", base+"/messages", `{"body":"order-001"}`).Body.Close()
	doRequest(t, "POST", base+"/messages", `{"body":"order-002"}`).Body.Close()

	// Receive head
	resp := doRequest(t, "GET", base+"/messages/head", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["body"] == nil {
		t.Fatal("expected message body")
	}
}

func TestServiceBusDeleteAndSend(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/servicebus/myns/queues/temp"

	doRequest(t, "PUT", base, "").Body.Close()

	// Delete
	resp := doRequest(t, "DELETE", base, "")
	resp.Body.Close()
	expectStatus(t, resp, 200)

	// Send to deleted queue should 404
	resp = doRequest(t, "POST", base+"/messages", `{"body":"test"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}
