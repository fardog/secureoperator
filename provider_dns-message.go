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
	// EDNSSentinelValue is the value that when sent to Google as the
	// EDNS value, means "do not use EDNS".
	EDNSSentinelValue            = "0.0.0.0/0"
	PaddingParameter             = "random_padding"
	ContentType                  = "application/dns-message"
	MaxBytesOfDNSQuestionMessage = 512
)

// DMProvider is the Google DNS-over-HTTPS provider; it implements the
// Provider interface, the abbreviation "DM" stands for dns-message.
type DMProvider struct {
	endpoint         string
	url              *url.URL
	host             string
	opts             *DMProviderOptions
	client           *http.Client
	autoSubnetGetter func() (ip string)
}

// DMProviderOptions is a configuration object for optional DMProvider configuration
type DMProviderOptions struct {
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

	// use https://dns.google/resolve like endpoint
	Alternative bool

	// dns resolver for retrieve ip of DoH enpoint host
	DnsResolver string
}

// NewDMProvider creates a DMProvider
func NewDMProvider(endpoint string, opts *DMProviderOptions) (*DMProvider, error) {
	if opts == nil {
		opts = &DMProviderOptions{}
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	provider := &DMProvider{
		endpoint: endpoint,
		url:      u,
		host:     u.Host,
		opts:     opts,
	}
	err = configHTTPClient(provider)
	if err != nil {
		log.Errorf("config http client error: %v", err)
		return nil, err
	}

	provider.autoSubnetGetter = func(secondsBeforeRetry int64) func() string {
		timeLastTryGetSubnet := int64(0)
		subnetLastUpdated := ""
		renewSubnet := func(){
			timeLastTryGetSubnet = time.Now().Unix()
			log.Debugf("start obtain your external ip: %v", timeLastTryGetSubnet)
			dnsS := provider.opts.DnsResolver
			if dnsS == "" {
				dnsS = "8.8.8.8"
			}
			ipExternal, err := ObtainCurrentExternalIP(dnsS)
			if err == nil {
				ipInt := net.ParseIP(ipExternal)
				if ipInt.To4() == nil {
					subnetLastUpdated = ipExternal + "/64"
					log.Debugf("renew subnet: %v", subnetLastUpdated)
				} else {
					subnetLastUpdated = ipExternal + "/32"
					log.Debugf("renew subnet: %v", subnetLastUpdated)
				}
			}
		}
		return func() string {
			if time.Now().Unix() < timeLastTryGetSubnet + secondsBeforeRetry && time.Now().Unix() > timeLastTryGetSubnet{
				log.Debugf("seconds left to obtain external ip again: %v",
					timeLastTryGetSubnet + secondsBeforeRetry -time.Now().Unix())
				return subnetLastUpdated
			}else if subnetLastUpdated != ""{
				go renewSubnet()
				return subnetLastUpdated
			}else{
				renewSubnet()
			}
			return subnetLastUpdated
		}
	}(15*60) // renew external ip every 15min.

	return provider, nil
}

func configHTTPClient(provider *DMProvider) error {
	// Create TLS configuration with the certificate of the server
	tlsConfig := &tls.Config{
		ServerName: provider.url.Host,
	}

	// using custom CA certificate
	if _, err := os.Stat(provider.opts.CACertFilePath); err == nil {
		caCert, err := ioutil.ReadFile(provider.opts.CACertFilePath)
		if err != nil {
			log.Errorf("read custom CA certificate failed : %s", err)
			return err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// custom transport for supporting servernames which may not match the url,
	// in cases where we request directly against an IP
	tr := &http.Transport{
		TLSClientConfig:   tlsConfig,
		ForceAttemptHTTP2: provider.opts.HTTP2,
		DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			h, p, err := net.SplitHostPort(addr)
			if len(provider.opts.EndpointIPs) > 0 {
				if err == nil {
					ip := provider.opts.EndpointIPs[rand.Intn(len(provider.opts.EndpointIPs))]
					addr = net.JoinHostPort(ip.String(), p)
					log.Info("endpoint ip address from specified: ", addr)
				}
			} else if provider.opts.DnsResolver != "" {
				var ipResolved []string
				ipResolved = ResolveHostToIP(h+".", provider.opts.DnsResolver)

				if ipResolved == nil {
					log.Info("Can't resolve endpoint from provided dns server")
					return dialer.DialContext(ctx, network, addr)
				} else if len(ipResolved) == 0 {
					log.Info("Resolve answder of endpoint is empty from provided dns server")
					return dialer.DialContext(ctx, network, addr)
				}
				ip := ipResolved[rand.Intn(len(ipResolved))]
				addr = net.JoinHostPort(ip, p)
				log.Info("endpoint ip address from dns-resolver: ", addr)
			}

			return dialer.DialContext(ctx, network, addr)
		},
	}
	provider.client = &http.Client{Transport: tr}
	return nil
}

func (provider DMProvider) Query(msg *dns.Msg) (*dns.Msg, error) {

	if provider.opts.Alternative {
		return provider.urlParamsQuery(msg)
	}

	return provider.dnsMessageQuery(msg)
}

// urlParamsQuery sends a DNS question to Google, and returns the response
func (provider DMProvider) urlParamsQuery(msg *dns.Msg) (*dns.Msg, error) {
	// Return fake answer (SOA or empty) if NoAAAA option is on.
	if provider.opts.NoAAAA {
		for _, q := range msg.Question {
			if q.Qtype == dns.TypeAAAA {
				q.Qtype = dns.TypeSOA
				break
			}
		}
	}

	log.Debugf("Dns Question Msg: \n%v", msg)

	httpreq, err := provider.parameterizedRequest(msg)
	if err != nil {
		return nil, err
	}

	httpresp, err := provider.fireDoHRequest(httpreq)
	if err != nil {
		return nil, err
	}

	rawResponse, err := ioutil.ReadAll(httpresp.Body)

	if err != nil {
		return nil, err
	}

	// dns.google/resolve return DNS Answer with no ID,
	// modify it after unpack DNS Message.
	idOriginal := msg.Id
	err = msg.Unpack(rawResponse)
	msg.Id = idOriginal

	log.Debugf("Dns Answer Msg: \n%v", msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (provider DMProvider) dnsMessageQuery(msg *dns.Msg) (*dns.Msg, error) {

	return msg, nil
}

func (provider DMProvider) parameterizedRequest(msg *dns.Msg) (*http.Request, error) {
	u := *provider.url

	httpreq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	// set headers if provided; we don't merge these for now, as we don't set
	// any headers by default
	if provider.opts.Headers != nil {
		httpreq.Header = provider.opts.Headers
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
	if provider.opts.QueryParameters != nil {
		for k, vs := range provider.opts.QueryParameters {
			for _, v := range vs {
				qry.Add(k, v)
			}
		}
	}

	ednsSubnet := ""
	if provider.opts.EDNSSubnet == "no" {
		log.Debug("will not use EDNSSubnet.")
	} else if provider.opts.EDNSSubnet == "auto" {
		ednsSubnet = provider.autoSubnetGetter()
	} else {
		_, _, err := net.ParseCIDR(provider.opts.EDNSSubnet)
		if err != nil {
			log.Debugf("specified subnet is not OK: %v", provider.opts.EDNSSubnet)
		}
		log.Debugf("will use EDNSSubnet you specified: %v", provider.opts.EDNSSubnet)
		ednsSubnet = provider.opts.EDNSSubnet
	}

	if ednsSubnet != "" {
		qry.Add("edns_client_subnet", ednsSubnet)
	}

	randomPadding := strconv.FormatInt(time.Now().UnixNano(), 10)
	qry.Add(PaddingParameter, randomPadding)

	qry.Add("ct", ContentType)
	httpreq.URL.RawQuery = qry.Encode()

	return httpreq, nil
}

func (provider DMProvider) doHTTPRequest(cReq <-chan *http.Request, cRsp chan *http.Response) {
	httpresp, err := provider.client.Do(<-cReq)

	if err != nil {
		cRsp <- nil
		log.Errorln("HttpRequest Error", err)
	} else {
		cRsp <- httpresp
	}
}

func (provider DMProvider) fireDoHRequest(req *http.Request) (*http.Response, error) {
	cReq := make(chan *http.Request)
	cRsp := make(chan *http.Response)

	defer close(cReq)
	defer close(cRsp)

	go provider.doHTTPRequest(cReq, cRsp)
	cReq <- req

	httpresp := <-cRsp

	if httpresp == nil {
		return nil, errors.New("HttpRequest Error occured")
	} else {
		return httpresp, nil
	}
}
