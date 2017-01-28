package secureoperator

import (
	"net"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

// DNSQuestion represents a DNS question to be resolved by a DNS server
type DNSQuestion struct {
	Name string `json:"name,omitempty"`
	Type int32  `json:"type,omitempty"`
}

// DNSRR represents a DNS record, part of a response to a DNSQuestion
type DNSRR struct {
	Name string `json:"name,omitempty"`
	Type int32  `json:"type,omitempty"`
	TTL  int32  `json:"TTL,omitempty"`
	Data string `json:"data,omitempty"`
}

// RR transforms a DNSRR to a dns.RR. It does not support a full compliment of
// DNS records, although it will grow additional types as necessary. Currently
// it supports:
//   * A
//   * AAAA
//   * CNAME
//   * MX
//
// Any unsupported type will be translated to an RFC3597 message.
//
// Transforms from A, AAAA, and CNAME records are straightforward; the
// (DNSRR).Data fields are translated from strings to IP addresses (in the case
// of A, AAAA), or copied straight over (in the case of CNAME) to the dns.RR.
//
// MX Records expect the Data field to be in the format of "10 whatever.com",
// where "10" is the unsigned integer priority, and "whatever.com" is the server
// expected to serve the request.
func (r DNSRR) RR() dns.RR {
	var rr dns.RR

	// Build an RR header
	rrhdr := dns.RR_Header{
		Name:     r.Name,
		Rrtype:   uint16(r.Type),
		Class:    dns.ClassINET,
		Ttl:      uint32(r.TTL),
		Rdlength: uint16(len(r.Data)),
	}

	constructor, ok := dns.TypeToRR[uint16(r.Type)]
	if ok {
		// Construct a new RR
		rr = constructor()
		*(rr.Header()) = rrhdr
		switch v := rr.(type) {
		case *dns.A:
			v.A = net.ParseIP(r.Data)
		case *dns.AAAA:
			v.AAAA = net.ParseIP(r.Data)
		case *dns.CNAME:
			v.Target = r.Data
		case *dns.MX:
			c := strings.SplitN(r.Data, " ", 2)
			pref, err := strconv.ParseUint(c[0], 10, 32)
			if err != nil {
				break
			}

			v.Preference = uint16(pref)
			v.Mx = c[1]
		}
	} else {
		rr = dns.RR(&dns.RFC3597{
			Hdr:   rrhdr,
			Rdata: r.Data,
		})
	}
	return rr
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
