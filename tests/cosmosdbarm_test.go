package tests

import (
	"net/http/httptest"
	"testing"
)

const cosmosSub = "sub1"
const cosmosRG = "rg1"
const cosmosAcct = "mycosmosacct"

func cosmosARMBase(ts *httptest.Server) string {
	return ts.URL + "/subscriptions/" + cosmosSub + "/resourceGroups/" + cosmosRG + "/providers/Microsoft.DocumentDB/databaseAccounts"
}

func TestCosmosDBARMAccountCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := cosmosARMBase(ts) + "/" + cosmosAcct

	// Create account
	resp := doRequest(t, "PUT", base, `{"location":"eastus","kind":"GlobalDocumentDB"}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != cosmosAcct {
		t.Fatalf("expected name=%s, got %v", cosmosAcct, m["name"])
	}
	if m["type"] != "Microsoft.DocumentDB/databaseAccounts" {
		t.Fatalf("expected correct type, got %v", m["type"])
	}
	props := m["properties"].(map[string]interface{})
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState=Succeeded, got %v", props["provisioningState"])
	}
	if props["documentEndpoint"] == nil {
		t.Fatal("expected documentEndpoint in properties")
	}

	// Get account
	resp2 := doRequest(t, "GET", base, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 200)

	// Update account (idempotent)
	resp3 := doRequest(t, "PUT", base, `{"location":"eastus"}`)
	defer resp3.Body.Close()
	expectStatus(t, resp3, 200)
}

func TestCosmosDBARMAccountList(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := cosmosARMBase(ts)

	doRequest(t, "PUT", base+"/acct-a", `{}`).Body.Close()
	doRequest(t, "PUT", base+"/acct-b", `{}`).Body.Close()

	resp := doRequest(t, "GET", base, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 200)

	m := decodeJSON(t, resp)
	items := m["value"].([]interface{})
	if len(items) < 2 {
		t.Fatalf("expected at least 2 accounts, got %d", len(items))
	}
}

func TestCosmosDBARMAccountNotFound(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	resp := doRequest(t, "GET", cosmosARMBase(ts)+"/nonexistent", "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestCosmosDBARMAccountDelete(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := cosmosARMBase(ts) + "/acct-del"

	doRequest(t, "PUT", base, `{}`).Body.Close()

	resp := doRequest(t, "DELETE", base, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 202)

	resp2 := doRequest(t, "GET", base, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 404)
}

func TestCosmosDBARMSQLDatabaseCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctBase := cosmosARMBase(ts) + "/" + cosmosAcct
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()

	dbBase := acctBase + "/sqlDatabases/mydb"

	// Create database
	resp := doRequest(t, "PUT", dbBase, `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != "mydb" {
		t.Fatalf("expected name=mydb, got %v", m["name"])
	}
	if m["type"] != "Microsoft.DocumentDB/databaseAccounts/sqlDatabases" {
		t.Fatalf("expected correct type, got %v", m["type"])
	}

	// Get database
	resp2 := doRequest(t, "GET", dbBase, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 200)

	// List databases
	resp3 := doRequest(t, "GET", acctBase+"/sqlDatabases", "")
	defer resp3.Body.Close()
	expectStatus(t, resp3, 200)
	ml := decodeJSON(t, resp3)
	if len(ml["value"].([]interface{})) == 0 {
		t.Fatal("expected at least 1 database in list")
	}

	// Delete database
	resp4 := doRequest(t, "DELETE", dbBase, "")
	defer resp4.Body.Close()
	expectStatus(t, resp4, 202)

	resp5 := doRequest(t, "GET", dbBase, "")
	defer resp5.Body.Close()
	expectStatus(t, resp5, 404)
}

func TestCosmosDBARMContainerCRUD(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	acctBase := cosmosARMBase(ts) + "/" + cosmosAcct
	doRequest(t, "PUT", acctBase, `{}`).Body.Close()
	doRequest(t, "PUT", acctBase+"/sqlDatabases/mydb", `{}`).Body.Close()

	cBase := acctBase + "/sqlDatabases/mydb/containers/users"

	// Create container
	resp := doRequest(t, "PUT", cBase, `{}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	m := decodeJSON(t, resp)
	if m["name"] != "users" {
		t.Fatalf("expected name=users, got %v", m["name"])
	}
	if m["type"] != "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers" {
		t.Fatalf("expected correct type, got %v", m["type"])
	}

	// Get container
	resp2 := doRequest(t, "GET", cBase, "")
	defer resp2.Body.Close()
	expectStatus(t, resp2, 200)

	// List containers
	resp3 := doRequest(t, "GET", acctBase+"/sqlDatabases/mydb/containers", "")
	defer resp3.Body.Close()
	expectStatus(t, resp3, 200)
	ml := decodeJSON(t, resp3)
	if len(ml["value"].([]interface{})) == 0 {
		t.Fatal("expected at least 1 container in list")
	}

	// Delete container
	resp4 := doRequest(t, "DELETE", cBase, "")
	defer resp4.Body.Close()
	expectStatus(t, resp4, 202)

	resp5 := doRequest(t, "GET", cBase, "")
	defer resp5.Body.Close()
	expectStatus(t, resp5, 404)
}

func TestCosmosDBARMResponseHasID(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := cosmosARMBase(ts) + "/myacct"

	resp := doRequest(t, "PUT", base, `{}`)
	defer resp.Body.Close()
	m := decodeJSON(t, resp)

	expectedID := "/subscriptions/" + cosmosSub + "/resourceGroups/" + cosmosRG + "/providers/Microsoft.DocumentDB/databaseAccounts/myacct"
	if m["id"] != expectedID {
		t.Fatalf("expected id=%s, got %v", expectedID, m["id"])
	}
}
