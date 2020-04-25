package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/miekg/dns"
)

var gresp = `{"Status": 0,"TC": false,"RD": true,"RA": true,"AD": true,"CD": false,"Question":[ {"name": "example.com.","type": 1}],"Answer":[ {"name": "example.com.","type": 1,"TTL": 78172,"data": "93.184.216.34"}]}`

func TestQuery(t *testing.T) {
	name := "example.com"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if n := q.Get("name"); n != name {
			t.Errorf("unexpected name in query: %v", n)
		}
		if tp := q.Get("type"); tp != strconv.Itoa(int(dns.TypeA)) {
			t.Errorf("unexpected type in query: %v", tp)
		}

		if ed := q.Get("edns_client_subnet"); ed != GoogleEDNSSentinelValue {
			t.Errorf("expected EDNS to be set to Google sentinel value, was: %v", ed)
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, gresp)
	}))
	defer ts.Close()

	_, err := NewGDNSProvider(ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEDNS(t *testing.T) {
	name := "example.com"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if n := q.Get("name"); n != name {
			t.Errorf("unexpected name in query: %v", n)
		}
		if tp := q.Get("type"); tp != strconv.Itoa(int(dns.TypeA)) {
			t.Errorf("unexpected type in query: %v", tp)
		}
		if e := q.Get("edns_client_subnet"); e != "64.10.0.0/20" {
			t.Errorf("did not use edns_client_subnet option specified")
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, gresp)
	}))
	defer ts.Close()

	_, err := NewGDNSProvider(ts.URL, &GDNSOptions{
		EDNSSubnet:          "64.10.0.0/20",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEDNSOmittedWhenBlank(t *testing.T) {
	name := "example.com"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if n := q.Get("name"); n != name {
			t.Errorf("unexpected name in query: %v", n)
		}
		if tp := q.Get("type"); tp != strconv.Itoa(int(dns.TypeA)) {
			t.Errorf("unexpected type in query: %v", tp)
		}
		if strings.Contains(r.URL.RawQuery, "edns_client_subnet") {
			t.Errorf("edns_client_subnet should be omitted")
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, gresp)
	}))
	defer ts.Close()

	_, err := NewGDNSProvider(ts.URL, &GDNSOptions{
		EDNSSubnet:          "",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEDNSIgnoredByDefault(t *testing.T) {
	// Deprecated: remove test in v4
	name := "example.com"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if n := q.Get("name"); n != name {
			t.Errorf("unexpected name in query: %v", n)
		}
		if tp := q.Get("type"); tp != strconv.Itoa(int(dns.TypeA)) {
			t.Errorf("unexpected type in query: %v", tp)
		}
		if e := q.Get("edns_client_subnet"); e != "0.0.0.0/0" {
			t.Errorf("did not use edns_client_subnet option specified")
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, gresp)
	}))
	defer ts.Close()

	_, err := NewGDNSProvider(ts.URL, &GDNSOptions{
		EDNSSubnet: "64.10.0.0/20",
	})
	if err != nil {
		t.Fatal(err)
	}
}
