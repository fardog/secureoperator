package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	// MaxBytesOfDNSName is the maximum number of bytes a DNS name may contain
	MaxBytesOfDNSName = 253
	// GoogleEDNSSentinelValue is the value that when sent to Google as the
	// EDNS value, means "do not use EDNS".
	GoogleEDNSSentinelValue = "0.0.0.0/0"
	PaddingParameter        = "random_padding"
	ContentType             = "application/dns-message"
	MaxBytesOfDNSQuestionMessage = 512
)

// GDNSOptions is a configuration object for optional GDNSProvider configuration
type GDNSOptions struct {
	// EndpointIPs is a list of IPs to be used as the GDNS endpoint, avoiding
	// DNS lookups in the case where they are provided. One is chosen randomly
	// for each request.

	EndpointIPs []net.IP
	// The EDNS subnet to send in the edns0-client-subnet option. If not
	// specified, Google determines this automatically. To specify that the
	// option should not be set, use the value "0.0.0.0/0".

	EDNSSubnet string
	// Additional headers to be sent with requests to the DNS provider
	Headers http.Header

	// Additional query parameters to be sent with requests to the DNS provider
	QueryParameters map[string][]string

	// if using http2 for query
	HTTP2 bool

	// using specific CA cert file for TLS establishment
	CACertFilePath string

	// Reply All AAAA Questions with a Empty Answer
	NoAAAA bool

	Alternative bool
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

	// Create TLS configuration with the certificate of the server
	tlsConfig := &tls.Config{
		ServerName: g.url.Host,
	}

	// using custom CA certificate
	if _, err := os.Stat(opts.CACertFilePath); err == nil {
		caCert, err := ioutil.ReadFile(opts.CACertFilePath)
		if err != nil {
			_ = fmt.Errorf("read custom CA certificate failed : %s", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	} else {
		_ = fmt.Errorf("specified CA cert don't exist.")
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	// custom transport for supporting servernames which may not match the url,
	// in cases where we request directly against an IP
	tr := &http.Transport{
		TLSClientConfig:   tlsConfig,
		ForceAttemptHTTP2: opts.HTTP2,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if len(opts.EndpointIPs) > 0 {
				_, p, err := net.SplitHostPort(addr)
				if err == nil {
					ip := opts.EndpointIPs[rand.Intn(len(opts.EndpointIPs))]
					addr = net.JoinHostPort(ip.String(), p)
				}
			}
			return dialer.DialContext(ctx, network, addr)
		},
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
	client   *http.Client
}

func (g GDNSProvider) newRequest(msg *dns.Msg) (*http.Request, error) {
	u := *g.url

	httpreq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	// set headers if provided; we don't merge these for now, as we don't set
	// any headers by default
	if g.opts.Headers != nil {
		httpreq.Header = g.opts.Headers
	}

	qry := httpreq.URL.Query()
	dnsType := fmt.Sprintf("%v", msg.Question[0].Qtype)

	l := len([]byte(msg.Question[0].Name))
	if l > MaxBytesOfDNSName {
		return nil, fmt.Errorf("name length of %v exceeds DNS name max length", l)
	}

	qry.Add("name", msg.Question[0].Name)
	qry.Add("type", dnsType)

	// add additional query parameters
	if g.opts.QueryParameters != nil {
		for k, vs := range g.opts.QueryParameters {
			for _, v := range vs {
				qry.Add(k, v)
			}
		}
	}

	edns_subnet := GoogleEDNSSentinelValue

	if edns_subnet != "" {
		qry.Add("edns_client_subnet", edns_subnet)
	}

	random_padding := strconv.FormatInt(time.Now().UnixNano(), 10)
	qry.Add(PaddingParameter, random_padding)

	qry.Add("ct", ContentType)
	httpreq.URL.RawQuery = qry.Encode()

	return httpreq, nil
}

func (g GDNSProvider) DoHTTPRequest(c_req <-chan *http.Request, c_rsp chan *http.Response) {
	httpresp, err := g.client.Do(<-c_req)

	if err != nil {
		c_rsp <- nil
		log.Errorln("HttpRequest Error", err)
	} else {
		c_rsp <- httpresp
	}
}

func(g GDNSProvider) FireDoDoHRequest(req *http.Request)(*http.Response, error){
	c_req := make(chan *http.Request)
	c_rsp := make(chan *http.Response)

	defer close(c_req)
	defer close(c_rsp)

	go g.DoHTTPRequest(c_req, c_rsp)
	c_req <- req

	httpresp := <-c_rsp

	if httpresp == nil {
		return nil, errors.New("HttpRequest Error occured")
	}else{
		return httpresp, nil
	}
}

func makeFakeAnswer(msg *dns.Msg)(msg_reply *dns.Msg){
	reply_fake := dns.Msg{
		dns.MsgHdr{
			Id: msg.Id,
		},
		msg.Compress,
		msg.Question,
		nil,
		nil,
		nil,
	}
	reply_fake_final := msg.SetReply(&reply_fake)
	return reply_fake_final
}

// UrlSimpleParamsQuery sends a DNS question to Google, and returns the response
func (g GDNSProvider) UrlSimpleParamsQuery(msg *dns.Msg) ([]byte, error) {
	// Return fake answer (SOA or empty) if NoAAAA option is on.
	var isNoAAAA bool
	var fake_answer *dns.Msg
	if g.opts.NoAAAA {
		for _, q := range msg.Question {
			if q.Qtype == dns.TypeAAAA {
				q.Qtype = dns.TypeSOA
				isNoAAAA = true
				fake_answer = makeFakeAnswer(msg)
				break
			}
		}
	}

	log_debug("Dns Question Msg: \n", msg)

	httpreq, err := g.newRequest(msg)
	if err != nil {
		return nil, err
	}

	httpresp, err := g.FireDoDoHRequest(httpreq)
	if err != nil {
		return nil, err
	}

	raw_response, err := ioutil.ReadAll(httpresp.Body)

	if err != nil {
		return nil, err
	}

	log_debug("Dns Question Msg: \n", msg)

	// dns.google/resolve return DNS Answer with no ID,
	// modify it after unpack DNS Message.
	id_original := msg.Id
	err = msg.Unpack(raw_response)
	msg.Id = id_original

	if isNoAAAA {
		if fake_answer != nil{
			fake_answer.Ns = msg.Ns
		}
		return fake_answer.Pack()
	}

	log.Debug("Dns Answer Msg: \n", msg)
	if err != nil {
		return nil, err
	}
	return msg.Pack()
}

func (g GDNSProvider) StandardQuery(msg *dns.Msg)([]byte, error){
	// Return fake answer (SOA or empty) if NoAAAA option is on.
	var isNoAAAA bool
	var fake_answer *dns.Msg
	if g.opts.NoAAAA {
		for _, q := range msg.Question {
			if q.Qtype == dns.TypeAAAA {
				q.Qtype = dns.TypeSOA
				isNoAAAA = true
				fake_answer = makeFakeAnswer(msg)
				break
			}
		}
	}

	log_debug("Dns Question Msg: \n", msg)

	httpreq, err := g.newRequest(msg)
	if err != nil {
		return nil, err
	}

	httpresp, err := g.FireDoDoHRequest(httpreq)
	if err != nil {
		return nil, err
	}

	raw_response, err := ioutil.ReadAll(httpresp.Body)

	if err != nil {
		return nil, err
	}

	log_debug("Dns Question Msg: \n", msg)

	// dns.google/resolve return DNS Answer with no ID,
	// modify it after unpack DNS Message.
	id_original := msg.Id
	err = msg.Unpack(raw_response)
	msg.Id = id_original

	if isNoAAAA {
		if fake_answer != nil{
			fake_answer.Ns = msg.Ns
		}
		return fake_answer.Pack()
	}

	log.Debug("Dns Answer Msg: \n", msg)
	if err != nil {
		return nil, err
	}
	return msg.Pack()
}

func (g GDNSProvider) Query(msg *dns.Msg)([]byte, error) {

	return g.UrlSimpleParamsQuery(msg)
}