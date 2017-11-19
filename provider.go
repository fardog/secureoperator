package secureoperator

import (
	"github.com/miekg/dns"
)

// DNSQuestion represents a DNS question to be resolved by a DNS server
type DNSQuestion struct {
	Name string `json:"name,omitempty"`
	Type uint16 `json:"type,omitempty"`
}

// DNSRR represents a DNS record, part of a response to a DNSQuestion
type DNSRR struct {
	Name string `json:"name,omitempty"`
	Type uint16 `json:"type,omitempty"`
	TTL  uint32 `json:"TTL,omitempty"`
	Data string `json:"data,omitempty"`
}

// RR is deprecated as of 2.2.0, use DNSRR instead
func (r DNSRR) RR() dns.RR {
	hdr := dns.RR_Header{Name: r.Name, Rrtype: r.Type, Class: dns.ClassINET, Ttl: r.TTL}
	str := hdr.String() + r.Data
	rr, _ := dns.NewRR(str)
	return rr
}

// DNSRR transforms a DNSRR to a dns.RR; returns `nil` if an RR could not be
// created from the record.
func (r DNSRR) DNSRR() (dns.RR, error) {
	hdr := dns.RR_Header{Name: r.Name, Rrtype: r.Type, Class: dns.ClassINET, Ttl: r.TTL}
	str := hdr.String() + r.Data
	return dns.NewRR(str)
}

func (r DNSRR) String() string {
	hdr := dns.RR_Header{Name: r.Name, Rrtype: r.Type, Class: dns.ClassINET, Ttl: r.TTL}
	return hdr.String()
}

// DNSResponse represents a complete DNS server response, to be served by the
// DNS server handler.
type DNSResponse struct {
	Question           []DNSQuestion
	Answer             []DNSRR
	Authority          []DNSRR
	Extra              []DNSRR
	Truncated          bool
	RecursionDesired   bool
	RecursionAvailable bool
	AuthenticatedData  bool
	CheckingDisabled   bool
	ResponseCode       int
}

// Provider is an interface representing a servicer of DNS queries.
type Provider interface {
	Query(DNSQuestion) (*DNSResponse, error)
}
