package storageauth

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestSignAndVerifySharedKeyFull(t *testing.T) {
	key, _ := base64.StdEncoding.DecodeString(DeterministicAccountKey("sub1", "rg1", "acct1", "1"))
	u, _ := url.Parse("http://localhost:4566/blob/acct1?comp=properties&restype=service")
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("x-ms-date", time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC).Format(http.TimeFormat))
	req.Header.Set("x-ms-version", "2020-10-02")

	if err := SignBlobSharedKey(req, "acct1", key, false); err != nil {
		t.Fatal(err)
	}
	if !VerifyBlobSharedKey(req, "acct1", key, key) {
		t.Fatal("expected signature to verify")
	}
}

func TestVerifyWrongAccountName(t *testing.T) {
	key, _ := base64.StdEncoding.DecodeString(DeterministicAccountKey("sub1", "rg1", "acct1", "1"))
	u, _ := url.Parse("http://localhost:4566/blob/acct1?comp=list")
	req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
	req.Header.Set("x-ms-date", time.Now().UTC().Format(http.TimeFormat))
	req.Header.Set("x-ms-version", "2020-10-02")
	_ = SignBlobSharedKey(req, "acct1", key, false)
	if VerifyBlobSharedKey(req, "otheracct", key, key) {
		t.Fatal("expected verify to fail for wrong account")
	}
}
