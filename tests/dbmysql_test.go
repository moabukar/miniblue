package tests

import (
	"testing"
)

func TestMySQLServerCreate(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	url := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver"

	resp := doRequest(t, "PUT", url, `{"location": "eastus"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != "myserver" {
		t.Fatalf("expected name=myserver, got %v", m["name"])
	}
	if m["type"] != "Microsoft.DBforMySQL/flexibleServers" {
		t.Fatalf("expected type=Microsoft.DBforMySQL/flexibleServers, got %v", m["type"])
	}
}

func TestMySQLServerCreateAgain(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	url := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver"

	resp := doRequest(t, "PUT", url, `{"location": "westus"}`)
	resp.Body.Close()
	expectStatus(t, resp, 201)

	resp = doRequest(t, "PUT", url, `{"location": "eastus"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
}

func TestMySQLServerGet(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	url := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver"

	doRequest(t, "PUT", url, `{"location": "eastus"}`).Body.Close()

	resp := doRequest(t, "GET", url, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["name"] != "myserver" {
		t.Fatalf("expected name=myserver, got %v", m["name"])
	}
	if m["location"] != "eastus" {
		t.Fatalf("expected location=eastus, got %v", m["location"])
	}

	props := m["properties"].(map[string]interface{})
	if props["state"] != "Ready" {
		t.Fatalf("expected state=Ready, got %v", props["state"])
	}
	if props["version"] != "8.0" {
		t.Fatalf("expected version=8.0, got %v", props["version"])
	}
}

func TestMySQLServerList(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	url := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers"

	doRequest(t, "PUT", ts.URL+"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/server1", `{}`).Body.Close()
	doRequest(t, "PUT", ts.URL+"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/server2", `{}`).Body.Close()

	resp := doRequest(t, "GET", url, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	value := m["value"].([]interface{})
	if len(value) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(value))
	}
}

func TestMySQLServerDelete(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	url := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver"

	doRequest(t, "PUT", url, `{}`).Body.Close()

	resp := doRequest(t, "DELETE", url, "")
	resp.Body.Close()
	expectStatus(t, resp, 202)
}

func TestMySQLServerGetAfterDelete(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	url := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver"

	doRequest(t, "PUT", url, `{}`).Body.Close()
	doRequest(t, "DELETE", url, "").Body.Close()

	resp := doRequest(t, "GET", url, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestMySQLServerInvalidName(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	url := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/my-server!"

	resp := doRequest(t, "PUT", url, `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 400)
}

func TestMySQLDatabaseCreate(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	serverURL := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver"
	dbURL := serverURL + "/databases/mydb"

	doRequest(t, "PUT", serverURL, `{}`).Body.Close()

	resp := doRequest(t, "PUT", dbURL, `{"properties": {"charset": "utf8mb4"}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != "mydb" {
		t.Fatalf("expected name=mydb, got %v", m["name"])
	}
	if m["type"] != "Microsoft.DBforMySQL/flexibleServers/databases" {
		t.Fatalf("expected type=Microsoft.DBforMySQL/flexibleServers/databases, got %v", m["type"])
	}
}

func TestMySQLDatabaseCreateAgain(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	serverURL := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver"
	dbURL := serverURL + "/databases/mydb"

	doRequest(t, "PUT", serverURL, `{}`).Body.Close()

	resp := doRequest(t, "PUT", dbURL, `{}`)
	resp.Body.Close()
	expectStatus(t, resp, 201)

	resp = doRequest(t, "PUT", dbURL, `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 200)
}

func TestMySQLDatabaseGet(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	serverURL := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver"
	dbURL := serverURL + "/databases/mydb"

	doRequest(t, "PUT", serverURL, `{}`).Body.Close()
	doRequest(t, "PUT", dbURL, `{"properties": {"charset": "utf8mb4", "collation": "utf8mb4_general_ci"}}`).Body.Close()

	resp := doRequest(t, "GET", dbURL, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	if m["name"] != "mydb" {
		t.Fatalf("expected name=mydb, got %v", m["name"])
	}

	props := m["properties"].(map[string]interface{})
	if props["charset"] != "utf8mb4" {
		t.Fatalf("expected charset=utf8mb4, got %v", props["charset"])
	}
	if props["collation"] != "utf8mb4_general_ci" {
		t.Fatalf("expected collation=utf8mb4_general_ci, got %v", props["collation"])
	}
}

func TestMySQLDatabaseList(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	serverURL := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver"
	dbListURL := serverURL + "/databases"

	doRequest(t, "PUT", serverURL, `{}`).Body.Close()
	doRequest(t, "PUT", serverURL+"/databases/db1", `{}`).Body.Close()
	doRequest(t, "PUT", serverURL+"/databases/db2", `{}`).Body.Close()

	resp := doRequest(t, "GET", dbListURL, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	value := m["value"].([]interface{})
	if len(value) != 2 {
		t.Fatalf("expected 2 databases, got %d", len(value))
	}
}

func TestMySQLDatabaseDelete(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	serverURL := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver"
	dbURL := serverURL + "/databases/mydb"

	doRequest(t, "PUT", serverURL, `{}`).Body.Close()
	doRequest(t, "PUT", dbURL, `{}`).Body.Close()

	resp := doRequest(t, "DELETE", dbURL, "")
	resp.Body.Close()
	expectStatus(t, resp, 202)
}

func TestMySQLDatabaseGetAfterDelete(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	serverURL := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver"
	dbURL := serverURL + "/databases/mydb"

	doRequest(t, "PUT", serverURL, `{}`).Body.Close()
	doRequest(t, "PUT", dbURL, `{}`).Body.Close()
	doRequest(t, "DELETE", dbURL, "").Body.Close()

	resp := doRequest(t, "GET", dbURL, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestMySQLDatabaseWithoutServer(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	dbURL := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver/databases/mydb"

	resp := doRequest(t, "PUT", dbURL, `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestMySQLDatabaseInvalidName(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	serverURL := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers/myserver"
	dbURL := serverURL + "/databases/my-db!"

	doRequest(t, "PUT", serverURL, `{}`).Body.Close()

	resp := doRequest(t, "PUT", dbURL, `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 400)
}
