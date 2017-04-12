package secureoperator

import (
	"net"
	"testing"
	"time"
)

func TestParseEndpoint(t *testing.T) {
	type Case struct {
		c    string
		p    uint16
		ip   string
		port uint16
	}

	cases := []Case{
		Case{"8.8.8.8", 53, "8.8.8.8", 53},
		Case{"8.8.4.4", 54, "8.8.4.4", 54},
		Case{"8.8.8.8:8053", 53, "8.8.8.8", 8053},
		Case{"8.8.4.4:8053", 53, "8.8.4.4", 8053},
	}

	for i, c := range cases {
		e, err := ParseEndpoint(c.c, c.p)
		if err != nil {
			t.Fatalf("%v: %v", i, err)
		}

		if e.IP.String() != c.ip {
			t.Errorf("%v: expected %v, got %v", i, e.IP, c.ip)
		}
		if e.Port != c.port {
			t.Errorf("%v: expected %v, got %v", i, e.Port, c.port)
		}
	}
}

func TestParseEndpointErrors(t *testing.T) {
	_, err := ParseEndpoint("8.8.8.8:53:54", 53)
	if err != ErrInvalidEndpointString {
		t.Fatal("expected ErrInvalidEndpointString")
	}

	_, err = ParseEndpoint("abc:53", 53)
	if err != ErrFailedParsingIP {
		t.Fatal("expected ErrFailedParsingIP")
	}
}

func TestDNSCache(t *testing.T) {
	d := newDNSCache()

	_, ok := d.Get("wut")
	if ok {
		t.Error("expected to retrieve no record, but got one")
	}

	d.Set("wut", dnsCacheRecord{
		msg:     nil,
		ips:     []net.IP{net.ParseIP("8.8.8.8")},
		expires: time.Now().Add(time.Minute * 5),
	})

	r, ok := d.Get("wut")
	if !ok {
		t.Fatal("expected to retrieve a record, but did not get one")
	}

	if len(r.ips) != 1 {
		t.Fatalf("expected one IP, but got none")
	}

	if r.ips[0].String() != "8.8.8.8" {
		t.Errorf("got unexpected IP: %v", r.ips[0].String())
	}

	d.Set("cool", dnsCacheRecord{
		msg:     nil,
		ips:     []net.IP{net.ParseIP("8.8.4.4")},
		expires: time.Now().Add(time.Minute * 5),
	})

	r, ok = d.Get("cool")
	if !ok {
		t.Fatal("expected to retrieve a record, but did not get one")
	}

	if len(r.ips) != 1 {
		t.Fatalf("expected one IP, but got none")
	}

	if r.ips[0].String() != "8.8.4.4" {
		t.Errorf("got unexpected IP: %v", r.ips[0].String())
	}

	r, ok = d.Get("nope")
	if ok {
		t.Error("expected to retrieve no record, but got one")
	}
}
