package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/moabukar/miniblue/internal/server"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

var (
	mysqlContainer *mysql.MySQLContainer
	mysqlURL       string
	serverURL      string
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	c, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithUsername("root"),
		mysql.WithPassword("test"),
	)
	if err != nil {
		log.Fatalf("failed to start MySQL container: %v", err)
	}
	mysqlContainer = c

	connStr, err := c.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}
	mysqlURL = connStr

	os.Setenv("MYSQL_URL", mysqlURL)

	srv := server.New()
	ts := httptest.NewServer(srv.Handler())
	serverURL = ts.URL

	code := m.Run()

	ts.Close()
	os.Unsetenv("MYSQL_URL")
	if err := mysqlContainer.Terminate(ctx); err != nil {
		log.Printf("failed to terminate MySQL container: %v", err)
	}

	os.Exit(code)
}

func doRequest(method, url, body string) (*http.Response, error) {
	var req *http.Request
	if body != "" {
		req, _ = http.NewRequest(method, url, nil)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	return http.DefaultClient.Do(req)
}

func dbExistsInMySQL(dbName string) (bool, error) {
	db, err := sql.Open("mysql", mysqlURL)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?)", dbName).Scan(&exists)
	return exists, err
}

func TestMySQLRealDatabaseCreation(t *testing.T) {
	serverURL := serverURL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers"

	resp, err := doRequest("PUT", serverURL+"/myserver", "")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected server create to succeed, got %d", resp.StatusCode)
	}

	dbURL := serverURL + "/myserver/databases/testdb"
	resp, err = doRequest("PUT", dbURL, `{"properties": {"charset": "utf8mb4"}}`)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected database create to succeed, got %d", resp.StatusCode)
	}

	exists, err := dbExistsInMySQL("testdb")
	if err != nil {
		t.Fatalf("failed to check database existence: %v", err)
	}
	if !exists {
		t.Fatal("expected database 'testdb' to exist in real MySQL")
	}
}

func TestMySQLRealDatabaseDeletion(t *testing.T) {
	serverURL := serverURL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers"

	resp, err := doRequest("PUT", serverURL+"/delserver", "")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	dbURL := serverURL + "/delserver/databases/deltest"
	resp, err = doRequest("PUT", dbURL, "")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	exists, _ := dbExistsInMySQL("deltest")
	if !exists {
		t.Fatalf("database 'deltest' should exist before deletion")
	}

	resp, err = doRequest("DELETE", dbURL, "")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected database delete to succeed, got %d", resp.StatusCode)
	}

	exists, err = dbExistsInMySQL("deltest")
	if err != nil {
		t.Fatalf("failed to check database existence: %v", err)
	}
	if exists {
		t.Fatal("expected database 'deltest' to be deleted from real MySQL")
	}
}

func TestMySQLRealDatabaseServerDeletion(t *testing.T) {
	serverURL := serverURL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers"

	resp, err := doRequest("PUT", serverURL+"/serversub1", "")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	dbURL := serverURL + "/serversub1/databases/srvtest1"
	resp, err = doRequest("PUT", dbURL, "")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	dbURL2 := serverURL + "/serversub1/databases/srvtest2"
	resp, err = doRequest("PUT", dbURL2, "")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	exists1, _ := dbExistsInMySQL("srvtest1")
	exists2, _ := dbExistsInMySQL("srvtest2")
	if !exists1 || !exists2 {
		t.Fatal("expected databases to exist before server deletion")
	}

	resp, err = doRequest("DELETE", serverURL+"/serversub1", "")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected server delete to succeed, got %d", resp.StatusCode)
	}

	exists1, err = dbExistsInMySQL("srvtest1")
	exists2, err = dbExistsInMySQL("srvtest2")
	if err != nil {
		t.Fatalf("failed to check database existence: %v", err)
	}
	if exists1 || exists2 {
		t.Fatal("expected databases to be deleted when server is deleted")
	}
}

func TestMySQLRealDatabasePreservesExisting(t *testing.T) {
	serverURL := serverURL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.DBforMySQL/flexibleServers"

	existsBefore, _ := dbExistsInMySQL("existingdb")
	if existsBefore {
		t.Skip("database 'existingdb' already exists, skipping test")
	}

	db, err := sql.Open("mysql", mysqlURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE DATABASE `existingdb`")
	if err != nil {
		t.Fatalf("failed to create existing database: %v", err)
	}
	defer db.Exec("DROP DATABASE `existingdb`")

	resp, err := doRequest("PUT", serverURL+"/preserveserver", "")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	dbURL := serverURL + "/preserveserver/databases/existingdb"
	resp, err = doRequest("PUT", dbURL, "")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected database create to succeed, got %d", resp.StatusCode)
	}

	exists, _ := dbExistsInMySQL("existingdb")
	if !exists {
		t.Fatal("expected database 'existingdb' to still exist (preserved)")
	}
}

func TestMySQLRealConnection(t *testing.T) {
	if mysqlURL == "" {
		t.Fatal("MYSQL_URL should be set")
	}

	db, err := sql.Open("mysql", mysqlURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping MySQL: %v", err)
	}
}

func Example() {
	fmt.Println("MySQL integration tests require Docker to be running")
}
