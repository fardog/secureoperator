package secureoperator

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// ParseEndpoint parses a string into an Endpoint object, where the endpoint
// string is in the format of "ip:port". If a port is not present in the string,
// the defaultPort is used.
func ParseEndpoint(endpoint string, defaultPort uint16) (ep Endpoint, err error) {
	e := strings.Split(endpoint, ":")

	if len(e) > 2 {
		return ep, errors.New("invalid format")
	}

	ip := net.ParseIP(e[0])
	if ip == nil {
		return ep, fmt.Errorf("unable to parse IP from string %s", e[0])
	}

	ep.IP = ip
	ep.Port = defaultPort

	if len(e) > 1 {
		i, err := strconv.ParseUint(e[1], 10, 16)
		if err != nil {
			return ep, err
		}

		ep.Port = uint16(i)
	}

	return ep, err
}

// Endpoint represents a host/port combo
type Endpoint struct {
	IP   net.IP
	Port uint16
}

func (e Endpoint) String() string {
	return net.JoinHostPort(e.IP.String(), string(e.Port))
}

// Endpoints is a list of Endpoint objects
type Endpoints []Endpoint

// Random retrieves a random Endpoint from a list of Endpoints
func (e Endpoints) Random() Endpoint {
	return e[rand.Intn(len(e))]
}

type dnsCacheRecord struct {
	msg     *dns.Msg
	ips     []net.IP
	expires time.Time
}

type dnsCache map[string]dnsCacheRecord

// NewSimpleDNSClient creates a SimpleDNSClient
func NewSimpleDNSClient(servers Endpoints) (*SimpleDNSClient, error) {
	if len(servers) < 1 {
		return nil, fmt.Errorf("at least one endpoint server is required")
	}
	return &SimpleDNSClient{
		servers: servers,
		cache:   dnsCache{},
	}, nil
}

// SimpleDNSClient is a DNS client, primarily for internal use in secure
// operator.
//
// It provides an in-memory cache, but was optimized to look up one address
// at a time only.
type SimpleDNSClient struct {
	servers Endpoints
	cache   dnsCache
}

// LookupIP looks up a single IP against the client's configured DNS servers,
// returning a value from cache if its still valid.
func (c *SimpleDNSClient) LookupIP(host string) ([]net.IP, error) {
	// see if cache has the entry; if it's still good, return it
	entry, ok := c.cache[host]
	if ok && entry.expires.After(time.Now()) {
		return entry.ips, nil
	}

	// we need to look it up
	server := c.servers.Random()
	msg := dns.Msg{}
	msg.SetQuestion(dns.Fqdn(host), dns.TypeA)

	r, err := dns.Exchange(&msg, server.String())
	if err != nil {
		return []net.IP{}, err
	}

	rec := dnsCacheRecord{
		msg: r,
	}

	var shortestTTL uint32

	for _, ans := range r.Answer {
		h := ans.Header()

		if shortestTTL == 0 || h.Ttl < shortestTTL {
			shortestTTL = h.Ttl
		}

		if t, ok := ans.(*dns.A); ok {
			rec.ips = append(rec.ips, t.A)
		}
	}

	// set the expiry
	rec.expires = time.Now().Add(time.Second * time.Duration(shortestTTL))

	// cache the record
	c.cache[host] = rec

	return entry.ips, nil
}
