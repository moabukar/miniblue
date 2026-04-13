package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/moabukar/miniblue/internal/server"
)

func setupMiniblue() *httptest.Server {
	srv := server.New()
	return httptest.NewServer(srv.Handler())
}

func runAzlocal(ts *httptest.Server, args ...string) (string, string, int) {
	cwd, _ := os.Getwd()
	binPath := cwd + "/../../bin/azlocal"
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), "LOCAL_AZURE_ENDPOINT="+ts.URL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err.Error(), -1
	}
	return string(output), "", 0
}

func TestStorageAccountCreate(t *testing.T) {
	ts := setupMiniblue()
	defer ts.Close()

	output, _, _ := runAzlocal(ts, "storage", "account", "create",
		"--resource-group", "myRG",
		"--name", "testacct")

	if !strings.Contains(output, "name") || !strings.Contains(output, "testacct") {
		t.Fatalf("expected account name in output, got: %s", output)
	}
}

func TestStorageAccountCreateWithFlags(t *testing.T) {
	ts := setupMiniblue()
	defer ts.Close()

	output, _, _ := runAzlocal(ts, "storage", "account", "create",
		"--resource-group", "myRG",
		"--name", "testacct2",
		"--location", "westus2",
		"--sku", "Premium_LRS")

	if !strings.Contains(output, "name") {
		t.Fatalf("expected account response, got: %s", output)
	}
}

func TestStorageAccountList(t *testing.T) {
	ts := setupMiniblue()
	defer ts.Close()

	runAzlocal(ts, "storage", "account", "create", "--resource-group", "myRG", "--name", "acct1")
	runAzlocal(ts, "storage", "account", "create", "--resource-group", "myRG", "--name", "acct2")

	output, _, _ := runAzlocal(ts, "storage", "account", "list", "--resource-group", "myRG")

	if !strings.Contains(output, "acct1") || !strings.Contains(output, "acct2") {
		t.Fatalf("expected both accounts in list, got: %s", output)
	}
}

func TestStorageAccountShow(t *testing.T) {
	ts := setupMiniblue()
	defer ts.Close()

	runAzlocal(ts, "storage", "account", "create", "--resource-group", "myRG", "--name", "showacct")

	output, _, _ := runAzlocal(ts, "storage", "account", "show",
		"--resource-group", "myRG",
		"--name", "showacct")

	if !strings.Contains(output, "showacct") || !strings.Contains(output, "Microsoft.Storage/storageAccounts") {
		t.Fatalf("expected account details in output, got: %s", output)
	}
}

func TestStorageAccountShowNotFound(t *testing.T) {
	ts := setupMiniblue()
	defer ts.Close()

	output, _, _ := runAzlocal(ts, "storage", "account", "show",
		"--resource-group", "myRG",
		"--name", "nonexistent")

	if !strings.Contains(output, "404") && !strings.Contains(output, "NotFound") {
		t.Fatalf("expected not found error in output, got: %s", output)
	}
}

func TestStorageAccountListKeys(t *testing.T) {
	ts := setupMiniblue()
	defer ts.Close()

	runAzlocal(ts, "storage", "account", "create", "--resource-group", "myRG", "--name", "keyacct")

	output, _, _ := runAzlocal(ts, "storage", "account", "list-keys",
		"--resource-group", "myRG",
		"--name", "keyacct")

	if !strings.Contains(output, "key1") || !strings.Contains(output, "key2") {
		t.Fatalf("expected keys in output, got: %s", output)
	}
}

func TestStorageAccountDelete(t *testing.T) {
	ts := setupMiniblue()
	defer ts.Close()

	runAzlocal(ts, "storage", "account", "create", "--resource-group", "myRG", "--name", "deleteacct")

	output, _, _ := runAzlocal(ts, "storage", "account", "delete",
		"--resource-group", "myRG",
		"--name", "deleteacct")

	if !strings.Contains(strings.ToLower(output), "deleted") {
		t.Fatalf("expected delete confirmation, got: %s", output)
	}

	showOutput, _, _ := runAzlocal(ts, "storage", "account", "show",
		"--resource-group", "myRG",
		"--name", "deleteacct")

	if !strings.Contains(showOutput, "404") && !strings.Contains(showOutput, "NotFound") {
		t.Fatalf("expected not found error after deletion, got: %s", showOutput)
	}
}

func TestStorageAccountMissingResourceGroup(t *testing.T) {
	ts := setupMiniblue()
	defer ts.Close()

	_, _, code := runAzlocal(ts, "storage", "account", "create",
		"--name", "testacct")

	if code == 0 {
		t.Fatal("expected error for missing --resource-group")
	}
}
