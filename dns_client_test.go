package secureoperator

import (
	"errors"
	"net"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/miekg/dns"
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

	_, err = ParseEndpoint("8.8.8.8:abc", 53)
	if err != ErrFailedParsingPort {
		t.Fatal("expected ErrFailedParsingPort")
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

func TestSimpleDNSClient(t *testing.T) {
	exch := exchange
	level := log.GetLevel()
	defer func() {
		exchange = exch
		log.SetLevel(level)
	}()

	var callCount int

	log.SetLevel(log.FatalLevel)
	exchange = func(m *dns.Msg, a string) (*dns.Msg, error) {
		callCount++

		if len(m.Question) != 1 {
			t.Fatal("expected only one question")
		}

		if q := m.Question[0].String(); q != ";google.com.	IN	 A" {
			t.Errorf("unexpected question: %v", q)
		}

		r := dns.Msg{
			Answer: []dns.RR{
				&dns.A{
					A:   net.ParseIP("1.2.3.4"),
					Hdr: dns.RR_Header{Ttl: 300},
				},
			},
		}
		r.SetReply(m)

		return &r, nil
	}

	// test first call, should hit resolver
	client, err := NewSimpleDNSClient(Endpoints{
		Endpoint{net.ParseIP("8.8.8.8"), 53},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	ips, err := client.LookupIP("google.com")
	if err != nil {
		t.Fatal(err)
	}

	if callCount != 1 {
		t.Error("expected call to exchange")
	}

	if len(ips) != 1 {
		t.Fatal("expected only one answer")
	}

	if ip := ips[0].String(); ip != "1.2.3.4" {
		t.Errorf("unexpected response: %v", ip)
	}

	ips, err = client.LookupIP("google.com")
	if err != nil {
		t.Fatal(err)
	}

	if callCount != 1 {
		t.Error("expected no additional call to exchange")
	}

	if len(ips) != 1 {
		t.Fatal("expected only one answer")
	}

	if ip := ips[0].String(); ip != "1.2.3.4" {
		t.Errorf("unexpected response: %v", ip)
	}
}

func TestSimpleDNSClientError(t *testing.T) {
	exch := exchange
	level := log.GetLevel()
	defer func() {
		exchange = exch
		log.SetLevel(level)
	}()

	log.SetLevel(log.FatalLevel)
	exchange = func(m *dns.Msg, a string) (*dns.Msg, error) {
		return nil, errors.New("whoopsie daisy")
	}

	client, err := NewSimpleDNSClient(Endpoints{
		Endpoint{net.ParseIP("8.8.8.8"), 53},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = client.LookupIP("who.wut")
	if err == nil {
		t.Fatal("expected error, got none")
	}

	if err.Error() != "whoopsie daisy" {
		t.Error("got unexpected error message")
	}
}
