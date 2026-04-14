package tests

import (
	"net/http/httptest"
	"testing"
)

const sbSub = "sub1"
const sbRG = "rg1"
const sbNS = "mynamespace"

func sbARMBase(ts *httptest.Server) string {
	return ts.URL + "/subscriptions/" + sbSub + "/resourceGroups/" + sbRG + "/providers/Microsoft.ServiceBus/namespaces"
}

func TestServiceBusARMNamespaceCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := sbARMBase(ts) + "/" + sbNS

	// Create namespace
	resp := doRequest(t, "PUT", base, `{"location":"eastus","sku":{"name":"Standard"}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != sbNS {
		t.Fatalf("expected name=%s, got %v", sbNS, m["name"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded, got %v", props["provisioningState"])
	}
	if m["type"] != "Microsoft.ServiceBus/namespaces" {
		t.Fatalf("expected type=Microsoft.ServiceBus/namespaces, got %v", m["type"])
	}

	// Get namespace
	resp2 := doRequest(t, "GET", base, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 200)

	// Update namespace (idempotent)
	resp3 := doRequest(t, "PUT", base, `{"location":"eastus"}`)
	defer resp3.Body.Close()
	expectStatus(t, resp3, 200)
}

func TestServiceBusARMNamespaceList(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := sbARMBase(ts)

	doRequest(t, "PUT", base+"/ns-a", `{}`).Body.Close()
	doRequest(t, "PUT", base+"/ns-b", `{}`).Body.Close()

	resp := doRequest(t, "GET", base, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	items := m["value"].([]interface{})
	if len(items) < 2 {
		t.Fatalf("expected at least 2 namespaces, got %d", len(items))
	}
}

func TestServiceBusARMNamespaceNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := sbARMBase(ts) + "/nonexistent"

	resp := doRequest(t, "GET", base, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestServiceBusARMNamespaceDelete(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := sbARMBase(ts) + "/ns-to-delete"

	doRequest(t, "PUT", base, `{}`).Body.Close()

	resp := doRequest(t, "DELETE", base, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 202)

	resp2 := doRequest(t, "GET", base, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 404)
}

func TestServiceBusARMQueueCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	nsBase := sbARMBase(ts) + "/" + sbNS
	doRequest(t, "PUT", nsBase, `{}`).Body.Close()

	qBase := nsBase + "/queues/orders"

	// Create queue
	resp := doRequest(t, "PUT", qBase, `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != "orders" {
		t.Fatalf("expected name=orders, got %v", m["name"])
	}
	if m["type"] != "Microsoft.ServiceBus/namespaces/queues" {
		t.Fatalf("expected correct type, got %v", m["type"])
	}

	// Get queue
	resp2 := doRequest(t, "GET", qBase, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 200)

	// List queues
	resp3 := doRequest(t, "GET", nsBase+"/queues", "")
	defer resp3.Body.Close()
	expectStatus(t, resp3, 200)
	ml := decodeJSON(t, resp3)
	if len(ml["value"].([]interface{})) == 0 {
		t.Fatal("expected at least 1 queue in list")
	}

	// Delete queue
	resp4 := doRequest(t, "DELETE", qBase, "")
	defer resp4.Body.Close()
	expectStatus(t, resp4, 200)

	// Get deleted queue = 404
	resp5 := doRequest(t, "GET", qBase, "")
	defer resp5.Body.Close()
	expectStatus(t, resp5, 404)
}

func TestServiceBusARMTopicCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	nsBase := sbARMBase(ts) + "/" + sbNS
	doRequest(t, "PUT", nsBase, `{}`).Body.Close()

	tBase := nsBase + "/topics/events"

	// Create topic
	resp := doRequest(t, "PUT", tBase, `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != "events" {
		t.Fatalf("expected name=events, got %v", m["name"])
	}
	if m["type"] != "Microsoft.ServiceBus/namespaces/topics" {
		t.Fatalf("expected correct type, got %v", m["type"])
	}

	// Get topic
	resp2 := doRequest(t, "GET", tBase, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 200)

	// List topics
	resp3 := doRequest(t, "GET", nsBase+"/topics", "")
	defer resp3.Body.Close()
	expectStatus(t, resp3, 200)

	// Delete topic
	resp4 := doRequest(t, "DELETE", tBase, "")
	defer resp4.Body.Close()
	expectStatus(t, resp4, 200)

	// Get deleted topic = 404
	resp5 := doRequest(t, "GET", tBase, "")
	defer resp5.Body.Close()
	expectStatus(t, resp5, 404)
}

func TestServiceBusARMResponseHasID(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := sbARMBase(ts) + "/myns"

	resp := doRequest(t, "PUT", base, `{}`)
	defer resp.Body.Close()
	m := decodeJSON(t, resp)

	expectedID := "/subscriptions/" + sbSub + "/resourceGroups/" + sbRG + "/providers/Microsoft.ServiceBus/namespaces/myns"
	if m["id"] != expectedID {
		t.Fatalf("expected id=%s, got %v", expectedID, m["id"])
	}
}
