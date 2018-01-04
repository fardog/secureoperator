package secureoperator

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
		fmt.Fprint(w, gresp)
	}))
	defer ts.Close()

	g, err := NewGDNSProvider(ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := g.Query(DNSQuestion{
		Name: name,
		Type: dns.TypeA,
	})
	if err != nil {
		t.Error(err)
	}

	a := resp.Answer[0]
	if a.Name != "example.com." {
		t.Errorf("unexpected name %v", a.Name)
	}
	if a.Type != dns.TypeA {
		t.Errorf("unexpected type %v", a.Type)
	}
	if a.Data != "93.184.216.34" {
		t.Errorf("unexpected data %v", a.Data)
	}
	if a.TTL != 78172 {
		t.Errorf("unexpected TTL %v", a.TTL)
	}

	if resp.ResponseCode != 0 {
		t.Errorf("unexpected response code %v", resp.ResponseCode)
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
		fmt.Fprint(w, gresp)
	}))
	defer ts.Close()

	g, err := NewGDNSProvider(ts.URL, &GDNSOptions{
		UseEDNSsubnetOption: true,
		EDNSSubnet:          "64.10.0.0/20",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = g.Query(DNSQuestion{
		Name: name,
		Type: dns.TypeA,
	})
	if err != nil {
		t.Error(err)
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
		fmt.Fprint(w, gresp)
	}))
	defer ts.Close()

	g, err := NewGDNSProvider(ts.URL, &GDNSOptions{
		UseEDNSsubnetOption: true,
		EDNSSubnet:          "",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = g.Query(DNSQuestion{
		Name: name,
		Type: dns.TypeA,
	})
	if err != nil {
		t.Error(err)
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
		fmt.Fprint(w, gresp)
	}))
	defer ts.Close()

	g, err := NewGDNSProvider(ts.URL, &GDNSOptions{
		EDNSSubnet: "64.10.0.0/20",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = g.Query(DNSQuestion{
		Name: name,
		Type: dns.TypeA,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestPadding(t *testing.T) {
	var expected int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := len([]byte(r.URL.String()))
		// first request, set the expectation to the length of the first URL we
		// see; if any others don't match it, our padding function fails
		if expected == 0 {
			expected = l
		}

		if l != expected {
			t.Errorf("unexpected URL length: %v, expected: %v", l, expected)
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, gresp)
	}))
	defer ts.Close()

	g, err := NewGDNSProvider(ts.URL, &GDNSOptions{Pad: true})
	if err != nil {
		t.Fatal(err)
	}

	questions := []DNSQuestion{
		DNSQuestion{Name: "whatever.yo", Type: dns.TypeA},
		// ensure differing types result in the same padded length
		DNSQuestion{Name: "sure.yep", Type: dns.TypeA},
		DNSQuestion{Name: "sure.yep", Type: dns.TypeMX},
		DNSQuestion{Name: "sure.yep", Type: dns.TypeTA},
		// ensure very long domains are fine
		DNSQuestion{Name: strings.Repeat("a", 253), Type: dns.TypeA},
		DNSQuestion{Name: strings.Repeat("a", 253), Type: dns.TypeTA},
	}

	for _, q := range questions {
		if _, err := g.Query(q); err != nil {
			t.Errorf(err.Error())
		}

	}

	// a name longer that 253 bytes should be an error
	if _, err := g.Query(DNSQuestion{Name: strings.Repeat("a", 254)}); err == nil {
		t.Errorf("expected an error for a too-long DNS name")
	}
}

func TestNameTooLong(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request should be made")
	}))
	defer ts.Close()

	g, err := NewGDNSProvider(ts.URL, &GDNSOptions{Pad: true})
	if err != nil {
		t.Fatal(err)
	}

	// a name longer that 253 bytes should be an error
	if _, err := g.Query(DNSQuestion{Name: strings.Repeat("a", 254)}); err == nil {
		t.Errorf("expected an error for a too-long DNS name")
	}
}
