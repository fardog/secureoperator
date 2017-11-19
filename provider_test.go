package secureoperator

import (
	"testing"

	"github.com/miekg/dns"
)

func TestDNSRRTypeA(t *testing.T) {
	var r DNSRR
	var rr dns.RR

	r = DNSRR{
		Name: "who.wut.co.jp",
		Type: dns.TypeA,
		TTL:  300,
		Data: "10.10.10.1",
	}

	rr, err := r.DNSRR()
	if err != nil {
		t.Fatal(err)
	}

	v, ok := rr.(*dns.A)
	if !ok {
		t.Error("did not get expected record type")
	}

	if v.Header().Name != (r.Name + ".") {
		t.Errorf("unexpected name %v", v.Header().Name)
	}
	if v.Header().Ttl != r.TTL {
		t.Errorf("unexpected TTL %v", v.Header().Ttl)
	}
	if v.A.String() != r.Data {
		t.Errorf("unexpected record data %v", v.A.String())
	}
}

func TestDNSRRTypeMX(t *testing.T) {
	var r DNSRR
	var rr dns.RR

	r = DNSRR{
		Name: "who.wut.co.jp",
		Type: dns.TypeMX,
		TTL:  300,
		Data: "10 mail.who.wut.co.jp",
	}

	rr, err := r.DNSRR()
	if err != nil {
		t.Fatal(err)
	}

	v, ok := rr.(*dns.MX)
	if !ok {
		t.Error("did not get expected record type")
	}

	if v.Header().Name != (r.Name + ".") {
		t.Errorf("unexpected name %v", v.Header().Name)
	}
	if v.Header().Ttl != r.TTL {
		t.Errorf("unexpected TTL %v", v.Header().Ttl)
	}
	if v.Preference != uint16(10) {
		t.Errorf("unexpected preference data %v", v.Preference)
	}
	if v.Mx != ("mail.who.wut.co.jp" + ".") {
		t.Errorf("unexpected mx data %v", v.Preference)
	}
}

func TestDNSRRTypeCNAME(t *testing.T) {
	var r DNSRR
	var rr dns.RR

	r = DNSRR{
		Name: "who.wut.co.jp",
		Type: dns.TypeCNAME,
		TTL:  300,
		Data: "omg.wtf.bbq",
	}

	rr, err := r.DNSRR()
	if err != nil {
		t.Fatal(err)
	}

	v, ok := rr.(*dns.CNAME)
	if !ok {
		t.Error("did not get expected record type")
	}

	if v.Header().Name != (r.Name + ".") {
		t.Errorf("unexpected name %v", v.Header().Name)
	}
	if v.Header().Ttl != r.TTL {
		t.Errorf("unexpected TTL %v", v.Header().Ttl)
	}
	if v.Target != (r.Data + ".") {
		t.Errorf("unexpected target data %v", v.Target)
	}
}

func TestDNSRRTypeAAAA(t *testing.T) {
	var r DNSRR
	var rr dns.RR

	r = DNSRR{
		Name: "who.wut.co.jp",
		Type: dns.TypeAAAA,
		TTL:  300,
		Data: "::1",
	}

	rr, err := r.DNSRR()
	if err != nil {
		t.Fatal(err)
	}

	v, ok := rr.(*dns.AAAA)
	if !ok {
		t.Error("did not get expected record type")
	}

	if v.Header().Name != (r.Name + ".") {
		t.Errorf("unexpected name %v", v.Header().Name)
	}
	if v.Header().Ttl != r.TTL {
		t.Errorf("unexpected TTL %v", v.Header().Ttl)
	}
	if v.AAAA.String() != r.Data {
		t.Errorf("unexpected record data %v", v.AAAA.String())
	}
}
