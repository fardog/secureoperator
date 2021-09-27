package dohProxy

import (
	"github.com/miekg/dns"
)

// Provider is an interface representing a service of DNS queries.
type Provider interface {
	Query(msg *dns.Msg) (*dns.Msg, error)
}