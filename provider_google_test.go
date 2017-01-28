package secureoperator

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, gresp)
	}))

	g := GDNSProvider{
		Endpoint: ts.URL,
		Pad:      false,
	}

	resp, err := g.Query(DNSQuestion{
		Name: name,
		Type: int32(dns.TypeA),
	})
	if err != nil {
		t.Error(err)
	}

	a := resp.Answer[0]
	if a.Name != "example.com." {
		t.Errorf("unexpected name %v", a.Name)
	}
	if a.Type != 1 {
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
