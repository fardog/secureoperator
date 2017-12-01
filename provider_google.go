package secureoperator

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	// DNSNameMaxBytes is the maximum number of bytes a DNS name may contain
	DNSNameMaxBytes = 253
	// max number of characters in a 16-bit uint integer, converted to string
	extraPad         = 5
	paddingParameter = "random_padding"
)

// GDNSQuestion represents a question response item from Google's DNS service
// This is currently the same as DNSQuestion, our internal implementation, but
// since Google's API is in flux, we keep them separate
type GDNSQuestion DNSQuestion

// DNSQuestion transforms a GDNSQuestion to a DNSQuestion and returns it.
func (r GDNSQuestion) DNSQuestion() DNSQuestion {
	return DNSQuestion{
		Name: r.Name,
		Type: r.Type,
	}
}

// GDNSQuestions is a array of GDNSQuestion objects
type GDNSQuestions []GDNSQuestion

// DNSQuestions transforms an array of GDNSQuestion objects to an array of
// DNSQuestion objects
func (rs GDNSQuestions) DNSQuestions() (rqs []DNSQuestion) {
	for _, r := range rs {
		rqs = append(rqs, r.DNSQuestion())
	}

	return
}

// GDNSRR represents a dns response record item from Google's DNS service.
// This is currently the same as DNSRR, our internal implementation, but since
// Google's API is in flux, we keep them separate
type GDNSRR DNSRR

// DNSRR transforms a GDNSRR to a DNSRR
func (r GDNSRR) DNSRR() DNSRR {
	return DNSRR{
		Name: r.Name,
		Type: r.Type,
		TTL:  r.TTL,
		Data: r.Data,
	}
}

// GDNSRRs represents an array of GDNSRR objects
type GDNSRRs []GDNSRR

// DNSRRs transforms an array of GDNSRR objects to an array of DNSRR objects
func (rs GDNSRRs) DNSRRs() (rrs []DNSRR) {
	for _, r := range rs {
		rrs = append(rrs, r.DNSRR())
	}

	return
}

// GDNSResponse represents a response from the Google DNS-over-HTTPS servers
type GDNSResponse struct {
	Status           int32         `json:"Status"`
	TC               bool          `json:"TC"`
	RD               bool          `json:"RD"`
	RA               bool          `json:"RA"`
	AD               bool          `json:"AD"`
	CD               bool          `json:"CD"`
	Question         GDNSQuestions `json:"Question,omitempty"`
	Answer           GDNSRRs       `json:"Answer,omitempty"`
	Authority        GDNSRRs       `json:"Authority,omitempty"`
	Additional       GDNSRRs       `json:"Additional,omitempty"`
	EDNSClientSubnet string        `json:"edns_client_subnet,omitempty"`
	Comment          string        `json:"Comment,omitempty"`
}

// GDNSOptions is a configuration object for optional GDNSProvider configuration
type GDNSOptions struct {
	// Pad specifies if a DNS request should be padded to a fixed length
	Pad bool
	// EndpointIPs is a list of IPs to be used as the GDNS endpoint, avoiding
	// DNS lookups in the case where they are provided. One is chosen randomly
	// for each request.
	EndpointIPs []net.IP
	// DNSServers is a list of Endpoints to be used as DNS servers when looking
	// up the endpoint; if not provided, the system DNS resolver is used.
	DNSServers Endpoints
}

// NewGDNSProvider creates a GDNSProvider
func NewGDNSProvider(endpoint string, opts *GDNSOptions) (*GDNSProvider, error) {
	if opts == nil {
		opts = &GDNSOptions{}
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	g := &GDNSProvider{
		endpoint: endpoint,
		url:      u,
		host:     u.Host,
		opts:     opts,
	}

	if len(opts.DNSServers) > 0 {
		d, err := NewSimpleDNSClient(opts.DNSServers)
		if err != nil {
			return nil, err
		}

		g.dns = d
	}

	// custom transport for supporting servernames which may not match the url,
	// in cases where we request directly against an IP
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{ServerName: g.url.Host},
	}
	g.client = &http.Client{Transport: tr}

	return g, nil
}

// GDNSProvider is the Google DNS-over-HTTPS provider; it implements the
// Provider interface.
type GDNSProvider struct {
	endpoint string
	url      *url.URL
	host     string
	opts     *GDNSOptions
	dns      *SimpleDNSClient
	client   *http.Client
}

func (g GDNSProvider) newRequest(q DNSQuestion) (*http.Request, error) {
	u := *g.url

	var mustSendHost bool

	if l := len(g.opts.EndpointIPs); l > 0 {
		// if endpointIPs are provided, use one of those
		u.Host = g.opts.EndpointIPs[rand.Intn(l)].String()
		mustSendHost = true
	} else if g.dns != nil {
		ips, err := g.dns.LookupIP(u.Host)
		if err != nil {
			return nil, err
		}

		if l := len(ips); l > 0 {
			u.Host = ips[rand.Intn(l)].String()
		} else {
			return nil, fmt.Errorf("lookup for Google DNS host %v failed", u.Host)
		}
		mustSendHost = true
	}

	httpreq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	qry := httpreq.URL.Query()
	dnsType := fmt.Sprintf("%v", q.Type)

	l := len([]byte(q.Name))
	if l > DNSNameMaxBytes {
		return nil, fmt.Errorf("name length of %v exceeds DNS name max length", l)
	}

	qry.Add("name", q.Name)
	qry.Add("type", dnsType)
	qry.Add("edns_client_subnet", "0.0.0.0/0")

	httpreq.URL.RawQuery = qry.Encode()

	if g.opts.Pad {
		// pad to the maximum size a valid request could be. we add `1` because
		// Google's DNS service ignores a trailing period, increasing the
		// possible size of a name by 1
		pad := randSeq(DNSNameMaxBytes + extraPad - l - len(dnsType) + 1)
		qry.Add(paddingParameter, pad)

		httpreq.URL.RawQuery = qry.Encode()
	}

	if mustSendHost {
		httpreq.Host = g.url.Host
	}

	return httpreq, nil
}

// Query sends a DNS question to Google, and returns the response
func (g GDNSProvider) Query(q DNSQuestion) (*DNSResponse, error) {
	httpreq, err := g.newRequest(q)
	if err != nil {
		return nil, err
	}

	httpresp, err := g.client.Do(httpreq)
	if err != nil {
		return nil, err
	}
	defer httpresp.Body.Close()

	dnsResp := new(GDNSResponse)
	decoder := json.NewDecoder(httpresp.Body)
	err = decoder.Decode(&dnsResp)
	if err != nil {
		return nil, err
	}

	return &DNSResponse{
		Question:           dnsResp.Question.DNSQuestions(),
		Answer:             dnsResp.Answer.DNSRRs(),
		Authority:          dnsResp.Authority.DNSRRs(),
		Extra:              dnsResp.Additional.DNSRRs(),
		Truncated:          dnsResp.TC,
		RecursionDesired:   dnsResp.RD,
		RecursionAvailable: dnsResp.RA,
		AuthenticatedData:  dnsResp.AD,
		CheckingDisabled:   dnsResp.CD,
		ResponseCode:       int(dnsResp.Status),
	}, nil
}
