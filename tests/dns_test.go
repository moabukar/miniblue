package tests

import (
	"testing"
)

func TestDNSZoneAndRecord(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/dnsZones"
	av := "?api-version=2023-07-01-preview"

	// Create zone
	resp := doRequest(t, "PUT", base+"/example.com"+av, `{"location":"global"}`)
	resp.Body.Close()
	expectStatus(t, resp, 201)

	// Create A record
	resp = doRequest(t, "PUT", base+"/example.com/A/www"+av,
		`{"properties":{"TTL":300,"ARecords":[{"ipv4Address":"1.2.3.4"}]}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 201)

	// Delete zone cascades records
	doRequest(t, "DELETE", base+"/example.com"+av, "").Body.Close()
	resp = doRequest(t, "GET", base+"/example.com/A/www"+av, "")
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}

func TestDNSRecordOnNonexistentZone(t *testing.T) {
	ts := setupServer()
	defer ts.Close()
	base := ts.URL + "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/dnsZones"

	resp := doRequest(t, "PUT", base+"/nope.com/A/www?api-version=2023-07-01",
		`{"properties":{"TTL":60}}`)
	defer resp.Body.Close()
	expectStatus(t, resp, 404)
}
