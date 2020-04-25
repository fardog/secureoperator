package main

import (
	"github.com/miekg/dns"
)

// Provider is an interface representing a servicer of DNS queries.
type Provider interface {
	Query(msg *dns.Msg) ([]byte, error)
}