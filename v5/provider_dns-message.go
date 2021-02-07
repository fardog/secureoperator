package dohProxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	// MaxBytesOfDNSName is the maximum number of bytes a DNS name may contain
	MaxBytesOfDNSName = 253
	// EDNSSentinelValue is the value that when sent to Google as the
	// EDNS value, means "do not use EDNS".
	EDNSSentinelValue    = "0.0.0.0/0"
	PaddingParameter     = "random_padding"
	ContentType          = "application/dns-message"
	MaxBytesOfDNSMessage = 512
	maxUInt16            = ^uint16(0)
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
	ipResolvers      map[string]func() ([]string, []string)
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

	JSONAPI bool

	// dns resolver for retrieve ip of DoH enpoint host
	DnsResolver string

	DnsMsgEncoder base64.Encoding
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
		Log.Errorf("config http client error: %v", err)
		return nil, err
	}

	// renew external ip every 15min.
	provider.autoSubnetGetter = provider.currentSubnetClosure(provider.opts.DnsResolver, 15*60)

	provider.ipResolvers = make(map[string]func() ([]string, []string))

	return provider, nil
}

func configHTTPClient(provider *DMProvider) error {
	// Create TLS configuration with the certificate of the server
	serverName, _, err := net.SplitHostPort(provider.url.Host)
	if err != nil {
		serverName = provider.url.Host
	}
	tlsConfig := &tls.Config{
		ServerName: serverName,
	}

	// using custom CA certificate
	if _, err := os.Stat(provider.opts.CACertFilePath); err == nil {
		caCert, err := ioutil.ReadFile(provider.opts.CACertFilePath)
		if err != nil {
			Log.Errorf("read custom CA certificate failed : %s", err)
			return err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	keepAliveTimeout := 300 * time.Second
	timeout := 15 * time.Second

	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: keepAliveTimeout,
	}

	// custom transport for supporting server name which may not match the url,
	// in cases where we request directly against an IP.
	tr := &http.Transport{
		Proxy:             nil,
		TLSClientConfig:   tlsConfig,
		ForceAttemptHTTP2: provider.opts.HTTP2,
		DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			h, p, err := net.SplitHostPort(addr)
			if len(provider.opts.EndpointIPs) > 0 {
				if err == nil {
					ip := provider.opts.EndpointIPs[rand.Intn(len(provider.opts.EndpointIPs))]
					addr = net.JoinHostPort(ip.String(), p)
					Log.Info("endpoint ip address from specified: ", addr)
				}
			} else if provider.opts.DnsResolver != "" {
				var ip4s, ip16s, ipsResolved []string
				var closure func() ([]string, []string)
				if provider.ipResolvers[h] == nil {
					// try set closure for resolving domain name.
					closure = ResolveHostToIPClosure(dns.CanonicalName(h), provider.opts.DnsResolver)
					provider.ipResolvers[h] = closure
				}
				// use the closure set for resolving domain name
				ip4s, ip16s = provider.ipResolvers[h]()
				ipsResolved = append(ip4s, ip16s...)
				if len(ipsResolved) == 0 {
					Log.Info("Can't resolve endpoint from provided dns server")
					return nil, fmt.Errorf("resolve failed during dailing")
				}
				ip := ipsResolved[rand.Intn(len(ipsResolved))]
				// only ipv4 if NoAAAA option is on.
				if provider.opts.NoAAAA {
					if len(ip4s) == 0 {
						return nil, fmt.Errorf("no ipv4 address avaiable for dailing")
					}
					ip = ipsResolved[rand.Intn(len(ip4s))]
				}
				addr = net.JoinHostPort(ip, p)
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}
	provider.client = &http.Client{Transport: tr, Timeout: timeout}
	return nil
}

func (provider *DMProvider) currentSubnetClosure(dnsResolver string, secondsBeforeRetry int64) (getter func() string) {
	expireTime := int64(0)
	subnetLastUpdated := ""
	updating := false
	renewSubnet := func() {
		updating = true
		Log.Debugf("start obtain your external ip: %v", time.Now().Unix())
		dnsS := dnsResolver
		if dnsS == "" {
			dnsS = "8.8.8.8"
		}
		ipExternal, err := provider.ObtainCurrentExternalIP(dnsS)
		if err == nil {
			ipInt := net.ParseIP(ipExternal)
			if ipInt.To4() == nil {
				subnetLastUpdated = ipExternal + "/64"
				Log.Debugf("renew subnet: %v", subnetLastUpdated)
			} else {
				subnetLastUpdated = ipExternal + "/32"
				Log.Debugf("renew subnet: %v", subnetLastUpdated)
			}
		}
		expireTime = time.Now().Unix() + secondsBeforeRetry
		updating = false
	}
	return func() string {
		if time.Now().Unix() < expireTime {
			Log.Debugf("seconds left to obtain external ip again: %v",
				time.Now().Unix()-expireTime)
			return subnetLastUpdated
		} else if subnetLastUpdated != "" {
			if !updating {
				go renewSubnet()
			}
			return subnetLastUpdated
		} else {
			renewSubnet()
		}
		return subnetLastUpdated
	}
}

// obtain external ip through some public apis.
func (provider *DMProvider) ObtainCurrentExternalIP(dnsResolver string) (string, error) {
	ip := ""
	type IPRespModel struct {
		Address string `json:"address"`
		Ip      string `json:"ip"`
	}

	apiToTry := []string{
		"https://wq.apnic.net/ip",
		"https://accountws.arin.net/public/seam/resource/rest/myip",
		"https://rdap.lacnic.net/rdap/info/myip",
		"https://api.myip.la/en?json",
		"https://api.ip.sb/jsonip",
	}

	keepAliveTimeout := 300 * time.Second
	timeout := 15 * time.Second

	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: keepAliveTimeout,
	}

	// custom transport for supporting server names which may not match the url,
	// in cases where we request directly against an IP
	tr := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			h, p, _ := net.SplitHostPort(addr)
			var ipResolved, ip4s, ip16s []string
			var closure func() ([]string, []string)
			if provider.ipResolvers[h] == nil {
				if len(provider.opts.EndpointIPs) != 0 {
					// if specified ip for endpoint, only try self query
					closure = provider.GetIPsClosure(dns.CanonicalName(h))
					provider.ipResolvers[h] = closure
					Log.Infof("using self query  as ns resolver")
				} else {
					closure = ResolveHostToIPClosure(dns.CanonicalName(h), dnsResolver)
					provider.ipResolvers[h] = closure
					Log.Infof("using %v  as ns resolver: ", dnsResolver)
				}
			}
			ip4s, ip16s = provider.ipResolvers[h]()
			ipResolved = append(ip4s, ip16s...)

			if len(ipResolved) == 0 {
				Log.Errorf("Can't resolve endpoint %v from self or provided dns server: %v", h, dnsResolver)
				return nil, fmt.Errorf("resolve failed during dailing")
			}
			ip := ipResolved[rand.Intn(len(ipResolved))]
			addr = net.JoinHostPort(ip, p)
			Log.Infof("external ip fetcher api endpoint resolved: %v", addr)
			return dialer.DialContext(ctx, network, addr)
		},
	}

	client := &http.Client{Transport: tr, Timeout: timeout}

	for _, uri := range apiToTry {
		Log.Debugf("start obtain external ip from: %v", uri)
		httpReq, err := http.NewRequest(http.MethodGet, uri, nil)
		if err != nil {
			Log.Errorf("retrieve external ip error: %v", err)
			continue
		}
		httpResp, err := client.Do(httpReq)
		if err != nil {
			Log.Errorf("http api call failed: %v", err)
			client.CloseIdleConnections()
			continue
		}
		if httpResp != nil {
			httpResp.Close = true
		}

		ipResp := new(IPRespModel)

		httpRespBytes, err := ioutil.ReadAll(httpResp.Body)
		if err != nil {
			Log.Errorf("http api call result read error: %v, %v", httpRespBytes, err)
		}
		err = json.Unmarshal(httpRespBytes, &ipResp)
		if err != nil {
			Log.Errorf("retrieve external ip error: %v", err)
			continue
		}
		if ipResp.Ip != "" {
			ip = ipResp.Ip
			Log.Infof("API result of obtain external ip: %v", ipResp)
		}
		if ipResp.Address != "" {
			ip = ipResp.Address
			Log.Infof("API result of obtain external ip: %v", ipResp)
		}
		if ip != "" {
			break
		}
	}

	if ip == "" {
		return "", errors.New("can't obtain your external ip address")
	}
	return ip, nil
}

func (provider DMProvider) Query(msg *dns.Msg) (*dns.Msg, error) {

	if len(msg.Question) == 0 {
		Log.Debugf("no questions in resolve request.")
		return nil, errors.New("should have question in resolve request")
	}

	if provider.opts.Alternative {
		return provider.urlParamsQuery(msg)
	}

	if provider.opts.JSONAPI {
		return provider.jsonQuery(msg)
	}

	return provider.dnsMessageQuery(msg)
}

// urlParamsQuery sends a DNS question to Google, and returns the response.
// endpoint: https://dns.google/resolve
func (provider DMProvider) urlParamsQuery(msg *dns.Msg) (*dns.Msg, error) {
	// Return fake answer (empty) if NoAAAA option is on.
	isAAAAQuestion := false
	if provider.opts.NoAAAA {
		for _, q := range msg.Question {
			if q.Qtype == dns.TypeAAAA {
				//msg.Question[i].Qtype = dns.TypeSOA
				isAAAAQuestion = true
				break
			}
		}
		if isAAAAQuestion {
			msgR := new(dns.Msg)
			msgR.SetReply(msg)
			return msgR, nil
		}
	}

	Log.Debugf("Dns Question Msg: \n%v", msg)

	httpReq, err := provider.parameterizedRequest(msg)
	if err != nil {
		return nil, err
	}

	httpResp, err := provider.doHTTPRequest(httpReq)
	if err != nil {
		return nil, err
	}
	httpResp.Close = true
	defer func() { _ = httpResp.Body.Close() }()
	rawResponse, err := ioutil.ReadAll(httpResp.Body)

	if err != nil {
		return nil, err
	}

	// dns.google/resolve return DNS Answer with no ID,
	// call SetReply after unpack DNS Message.
	rMsg := new(dns.Msg)
	err = rMsg.Unpack(rawResponse)
	if err != nil {
		Log.Errorf("unpack dns-message error: %v", err)
		return nil, err
	}
	rMsg.SetReply(msg)

	Log.Debugf("Dns Answer Msg: \n%v", msg)

	return rMsg, nil
}

func (provider DMProvider) dnsMessageQuery(msg *dns.Msg) (*dns.Msg, error) {
	// Return fake answer (empty) if NoAAAA option is on.
	isAAAAQuestion := false
	if provider.opts.NoAAAA {
		for _, q := range msg.Question {
			if q.Qtype == dns.TypeAAAA {
				isAAAAQuestion = true
				break
			}
		}
		if isAAAAQuestion {
			msgR := new(dns.Msg)
			msgR.SetReply(msg)
			return msgR, nil
		}
	}

	Log.Debugf("Dns Question Msg: \n%v", msg)

	ednsSubnet := ""
	if provider.opts.EDNSSubnet == "no" {
		//ReplaceEDNS0Subnet(msg, nil)
		Log.Debug("will not use EDNSSubnet.")
	} else if provider.opts.EDNSSubnet == "auto" {
		ednsSubnet = provider.autoSubnetGetter()
	} else {
		ednsSubnet = provider.opts.EDNSSubnet
		Log.Debugf("will try to use EDNSSubnet you specified: %v", provider.opts.EDNSSubnet)
	}

	if ednsSubnet != "" {
		placeSubnetToMsg(ednsSubnet, msg)
	}

	pad := func(length int) {
		paddingBytes := make([]byte, length)
		for i := range paddingBytes {
			paddingBytes[i] &= 0x0
		}
		optPadding := &dns.EDNS0_PADDING{Padding: paddingBytes}

		ReplaceEDNS0Padding(msg, optPadding)
	}

	// first try padding 0, then replace padding with rational value.
	pad(0)
	bytesMsg, err := msg.Pack()
	if err != nil {
		Log.Errorf("pack message error: %v", err)
		return nil, err
	}
	lenOfBytes := len(bytesMsg)

	paddingLength := CalculatePaddingLength(lenOfBytes, 32, 16)
	if paddingLength > 0 {
		pad(paddingLength)
	}

	bytesMsg, err = msg.Pack()
	if err != nil {
		Log.Errorf("pack message error: %v", err)
		return nil, err
	}
	Log.Debugf("request msg packed size: %v", len(bytesMsg))

	// Http POST
	//httpReq, err := http.NewRequest(http.MethodPost, provider.url.String(), bytes.NewBuffer(bytesMsg))

	// Http GET
	httpReq, err := http.NewRequest(http.MethodGet, provider.url.String(), nil)

	if err != nil {
		return nil, err
	}

	// set headers if provided; we don't merge these for now, as we don't set any headers by default
	if provider.opts.Headers != nil {
		httpReq.Header = provider.opts.Headers
	}
	// Http GET
	httpReq.Header.Add("Accept", "application/dns-message")
	// HTTP POST
	//httpReq.Header.Add("Content-Type", "application/dns-message")

	// Http GET
	dnsMsgBase64Url := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytesMsg)

	httpReq.URL.RawQuery = fmt.Sprintf("dns=%v", dnsMsgBase64Url)

	lenQuery := len([]byte(httpReq.URL.RawQuery))
	if lenQuery > MaxBytesOfDNSMessage {
		Log.Errorf("GET Header is too large: %v > %v", lenQuery, MaxBytesOfDNSMessage)
	}
	Log.Debugf("http url: %v <- size: %v", httpReq.URL, len([]byte(httpReq.URL.String())))

	httpResp, err := provider.doHTTPRequest(httpReq)
	if err != nil {
		return nil, err
	}

	httpResp.Close = true
	defer func() { _ = httpResp.Body.Close() }()

	rawResponse, err := ioutil.ReadAll(httpResp.Body)

	if err != nil {
		return nil, err
	}

	err = msg.Unpack(rawResponse)
	if err != nil {
		Log.Errorf("unpack dns-message error: %v", err)
		return nil, err
	}
	Log.Debugf("Dns Answer Msg: \n%v", msg)

	return msg, nil
}

func (provider DMProvider) parameterizedRequest(msg *dns.Msg) (*http.Request, error) {
	httpReq, err := http.NewRequest(http.MethodGet, provider.url.String(), nil)
	if err != nil {
		return nil, err
	}

	// set headers if provided; we don't merge these for now, as we don't set
	// any headers by default
	if provider.opts.Headers != nil {
		httpReq.Header = provider.opts.Headers
	}

	qry := httpReq.URL.Query()
	dnsType := fmt.Sprintf("%v", msg.Question[0].Qtype)

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
		//ReplaceEDNS0Subnet(msg, nil)
		Log.Debug("will not use EDNSSubnet.")
	} else if provider.opts.EDNSSubnet == "auto" {
		ednsSubnet = provider.autoSubnetGetter()
	} else {
		_, _, err := net.ParseCIDR(provider.opts.EDNSSubnet)
		if err != nil {
			Log.Debugf("specified subnet is not OK: %v", provider.opts.EDNSSubnet)
		}
		Log.Debugf("will use EDNSSubnet you specified: %v", provider.opts.EDNSSubnet)
		ednsSubnet = provider.opts.EDNSSubnet
	}

	if ednsSubnet != "" {
		qry.Add("edns_client_subnet", ednsSubnet)
	}

	qry.Add("ct", ContentType)
	httpReq.URL.RawQuery = qry.Encode()

	lengthOfUrlPreAllocated := len([]byte((httpReq.URL.String()) + PaddingParameter + "&="))

	paddingLength := CalculatePaddingLength(lengthOfUrlPreAllocated, 32, 16)
	// block length padding 128x total length.
	if paddingLength > 0 {
		qry.Add(PaddingParameter, GenerateUrlSafeString(paddingLength))
	}

	httpReq.URL.RawQuery = qry.Encode()

	lenQName := len([]byte(msg.Question[0].Name))
	if lenQName > MaxBytesOfDNSName {
		return nil, fmt.Errorf("name length of %v exceeds DNS name max length", lenQName)
	}
	Log.Debugf("http url: %v <- size %v", httpReq.URL, len([]byte(httpReq.URL.String())))
	return httpReq, nil
}

func (provider DMProvider) doHTTPRequest(req *http.Request) (rsp *http.Response, err error) {

	httpResp, err := provider.client.Do(req)

	if err != nil {
		Log.Errorf("HttpRequest Error: %v", err)
		provider.client.CloseIdleConnections()
		return nil, fmt.Errorf("HttpRequest Error: %v", err)
	} else {
		logHttpResp := func() {
			headerKV := httpResp.Header
			bodyBytes, _ := ioutil.ReadAll(httpResp.Body)
			Log.Errorf("Error Header:\n%v\nError Body:\n%v", headerKV, string(bodyBytes))
		}
		switch httpResp.StatusCode {
		case 301:
			// follow 301 redirect once.
			Log.Warnf("301 Moved Permanently")
			newLocation := httpResp.Header.Get("Location")
			logHttpResp()
			newUrl, err := url.Parse(newLocation)
			if err != nil {
				Log.Warnf("parse 301 location error: %v", err)
				return nil, err
			}
			// if no dns parameter, give up.
			// refer: https://developers.google.com/speed/public-dns/docs/doh
			dnsQ := newUrl.Query().Get("dns")
			if dnsQ == "" {
				Log.Warnf("301 location invalid")
				return nil, fmt.Errorf("301 location invalid")
			}
			req.URL = newUrl
			Log.Debugf("will try follow redirect url: %v", newUrl)
			return provider.doHTTPRequest(req)
		case 400:
			errStr := "400 Bad Request: may be invalid DNS request"
			Log.Errorf(errStr)
			logHttpResp()
			return nil, fmt.Errorf(errStr)
		case 413:
			errStr := "413 Payload Too Large"
			Log.Errorf(errStr)
			logHttpResp()
			return nil, fmt.Errorf(errStr)
		case 414:
			errStr := "414 URI Too Long"
			Log.Errorf(errStr)
			logHttpResp()
			return nil, fmt.Errorf(errStr)
		case 415:
			errStr := "415 Unsupported Media Type: " +
				"The POST body did not have an application/dns-message Content-Type header"
			Log.Errorf(errStr)
			logHttpResp()
			return nil, fmt.Errorf(errStr)
		case 429:
			errStr := "429 Too Many Requests: The client has sent too many requests in a given amount of time"
			Log.Errorf(errStr)
			logHttpResp()
			return nil, fmt.Errorf(errStr)
		case 500:
			errStr := "500 Internal Server Error"
			Log.Errorf(errStr)
			logHttpResp()
			return nil, fmt.Errorf(errStr)
		case 501:
			errStr := "501 Not Implemented: " +
				"Only GET and POST methods are implemented, other methods get this error"
			Log.Errorf(errStr)
			logHttpResp()
			return nil, fmt.Errorf(errStr)
		case 502:
			errStr := "502 Bad Gateway: The DoH service could not contact DNS resolvers"
			Log.Errorf(errStr)
			logHttpResp()
			return nil, fmt.Errorf(errStr)
		default:
			return httpResp, nil
		}
	}
}

// resolve domain name to ips (ipv4 and ipv6) using Dns over HTTPS.
func (provider *DMProvider) GetIPsClosure(name string) (closure func() (ip4s []string, ip16s []string)) {
	// hijack the EDNSSubnet option with a special msg id.
	var ip4s, ip16s []string
	m4 := new(dns.Msg)
	m6 := new(dns.Msg)
	m6.Id = m4.Id - 1
	ttl := uint32(0)
	expireTime := time.Now().Unix()
	qName := dns.CanonicalName(name)
	resolve := func() {
		opts := &DMProviderOptions{
			EndpointIPs:     provider.opts.EndpointIPs,
			EDNSSubnet:      "no",
			QueryParameters: provider.opts.QueryParameters,
			Headers:         provider.opts.Headers,
			HTTP2:           provider.opts.HTTP2,
			CACertFilePath:  provider.opts.CACertFilePath,
			NoAAAA:          provider.opts.NoAAAA,
			Alternative:     provider.opts.Alternative,
			JSONAPI:         provider.opts.JSONAPI,
			DnsResolver:     provider.opts.DnsResolver,
		}
		providerTmp, err := NewDMProvider(provider.endpoint, opts)
		if err != nil {
			Log.Errorf("can't get new provider: %v", err)
			return
		}
		if providerTmp == nil {
			Log.Errorf("temporary provider is nil")
			return
		}
		m4.SetQuestion(qName, dns.TypeA)
		m6.SetQuestion(qName, dns.TypeAAAA)

		m4.Id = dns.Id()
		m4r, err := providerTmp.Query(m4)
		if err == nil && len(m4r.Answer) != 0 {
			for _, answer := range m4r.Answer {
				switch answer.(type) {
				case *dns.A:
					ipv := answer.(*dns.A)
					if ipv != nil {
						ip4s = append(ip4s, ipv.A.String())
					}
				}
			}
		}

		// set id and push into channel for hijacking
		m6.Id = dns.Id()
		m6r, err := providerTmp.Query(m6)
		if err == nil && len(m6r.Answer) != 0 {
			for _, answer := range m6r.Answer {
				switch answer.(type) {
				case *dns.AAAA:
					ipv := answer.(*dns.AAAA)
					if ipv != nil {
						ip16s = append(ip16s, ipv.AAAA.String())
					}
				}
			}
		}
		ttl4 := GetMinTTLFromDnsMsg(m4r)
		ttl6 := GetMinTTLFromDnsMsg(m6r)
		if ttl4 < ttl6 {
			ttl = ttl4
		} else {
			ttl = ttl6
		}

		expireTime = time.Now().Unix() + int64(ttl)
		// set to nil to let the temp provider in GC's sight.
		providerTmp = nil
	}
	return func() ([]string, []string) {
		if len(ip4s) == 0 && len(ip16s) == 0 {
			resolve()
		} else if time.Now().Unix() > expireTime {
			go resolve()
		}
		// if empty result, ttl reset to 1
		if len(ip4s) == 0 && len(ip16s) == 0 {
			expireTime = time.Now().Unix() + 1
		}
		return ip4s, ip16s
	}
}

func (provider DMProvider) jsonQuery(msg *dns.Msg) (*dns.Msg, error) {
	// Return fake answer (empty) if NoAAAA option is on.
	isAAAAQuestion := false
	if provider.opts.NoAAAA {
		for _, q := range msg.Question {
			if q.Qtype == dns.TypeAAAA {
				//msg.Question[i].Qtype = dns.TypeSOA
				isAAAAQuestion = true
				break
			}
		}
		if isAAAAQuestion {
			msgR := new(dns.Msg)
			msgR.SetReply(msg)
			return msgR, nil
		}
	}

	Log.Debugf("Dns Question Msg: \n%v", msg)

	httpReq, err := provider.parameterizedRequest2(msg)
	if err != nil {
		return nil, err
	}

	httpResp, err := provider.doHTTPRequest(httpReq)
	if err != nil {
		return nil, err
	}
	httpResp.Close = true
	defer func() { _ = httpResp.Body.Close() }()

	// dns.google/resolve return DNS Answer with no ID,
	// call SetReply after unpack DNS Message.
	json_ := new(JSONDNSResponse)
	decoder := json.NewDecoder(httpResp.Body)
	err = decoder.Decode(json_)
	if err != nil && json_.Answer == nil && json_.Authority == nil {
		headerKV := httpResp.Header
		bodyBytes, _ := ioutil.ReadAll(httpResp.Body)
		Log.Errorf("Error json decoding Header:\n%v\nError Body:\n%v", headerKV, string(bodyBytes))
		return nil, fmt.Errorf("json decoding error")
	}

	rMsg := provider.obtainDMFromJSON(json_, msg)

	Log.Debugf("Dns Answer Msg: \n%v", rMsg)

	return rMsg, nil
}

// DNSRR represents a DNS record, part of a response to a DNSQuestion
type DNSRR struct {
	Name string `json:"name,omitempty"`
	Type uint16 `json:"type,omitempty"`
	TTL  uint32 `json:"TTL,omitempty"`
	Data string `json:"data,omitempty"`
}

// GDNSRRs represents an array of GDNSRR objects
type DNSRRs []DNSRR

// RR transforms a DNSRR to a dns.RR
func (r DNSRR) RR() (dns.RR, error) {
	hdr := dns.RR_Header{Name: r.Name, Rrtype: r.Type, Class: dns.ClassINET, Ttl: r.TTL}
	str := hdr.String() + r.Data
	return dns.NewRR(str)
}

// DNSQuestion represents a DNS question to be resolved by a DNS server
type DNSQuestion struct {
	Name string `json:"name,omitempty"`
	Type uint16 `json:"type,omitempty"`
}
type DNSQuestions []DNSQuestion

// JSONDNSResponse represents a response from the Google DNS-over-HTTPS servers
type JSONDNSResponse struct {
	Status           int32        `json:"Status"`
	TC               bool         `json:"TC"`
	RD               bool         `json:"RD"`
	RA               bool         `json:"RA"`
	AD               bool         `json:"AD"`
	CD               bool         `json:"CD"`
	Question         DNSQuestions `json:"Question,omitempty"`
	Answer           DNSRRs       `json:"Answer,omitempty"`
	Authority        DNSRRs       `json:"Authority,omitempty"`
	Additional       DNSRRs       `json:"Additional,omitempty"`
	EDNSClientSubnet string       `json:"edns_client_subnet,omitempty"`
	Comment          string       `json:"Comment,omitempty"`
}

func (provider DMProvider) parameterizedRequest2(msg *dns.Msg) (*http.Request, error) {
	httpReq, err := http.NewRequest(http.MethodGet, provider.url.String(), nil)
	if err != nil {
		return nil, err
	}

	// set headers if provided; we don't merge these for now, as we don't set
	// any headers by default
	if provider.opts.Headers != nil {
		httpReq.Header = provider.opts.Headers
	}

	httpReq.Header.Add("Accept", "application/json")

	qry := httpReq.URL.Query()
	dnsType := fmt.Sprintf("%v", msg.Question[0].Qtype)

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
		//ReplaceEDNS0Subnet(msg, nil)
		Log.Debug("will not use EDNSSubnet.")
	} else if provider.opts.EDNSSubnet == "auto" {
		ednsSubnet = provider.autoSubnetGetter()
	} else {
		_, _, err := net.ParseCIDR(provider.opts.EDNSSubnet)
		if err != nil {
			Log.Debugf("specified subnet is not OK: %v", provider.opts.EDNSSubnet)
		}
		Log.Debugf("will use EDNSSubnet you specified: %v", provider.opts.EDNSSubnet)
		ednsSubnet = provider.opts.EDNSSubnet
	}

	if ednsSubnet != "" {
		qry.Add("edns_client_subnet", ednsSubnet)
	}

	httpReq.URL.RawQuery = qry.Encode()

	lengthOfUrlPreAllocated := len([]byte((httpReq.URL.String()) + PaddingParameter + "&="))

	paddingLength := CalculatePaddingLength(lengthOfUrlPreAllocated, 32, 16)
	// block length padding 128x total length.
	if paddingLength > 0 {
		qry.Add(PaddingParameter, GenerateUrlSafeString(paddingLength))
	}

	httpReq.URL.RawQuery = qry.Encode()

	lenQName := len([]byte(msg.Question[0].Name))
	if lenQName > MaxBytesOfDNSName {
		return nil, fmt.Errorf("name length of %v exceeds DNS name max length", lenQName)
	}
	Log.Debugf("http url: %v <- size %v", httpReq.URL, len([]byte(httpReq.URL.String())))
	return httpReq, nil
}

func (provider DMProvider) obtainDMFromJSON(json_ *JSONDNSResponse, qMsg *dns.Msg) *dns.Msg {
	rMsg := new (dns.Msg)
	rMsg.Truncated = json_.TC
	rMsg.RecursionDesired = json_.RD
	rMsg.RecursionAvailable = json_.AD
	rMsg.AuthenticatedData = json_.AD
	rMsg.CheckingDisabled = json_.CD
	rMsg.Rcode = int(json_.Status)

	answers := transformRR(json_.Answer, "answer")
	authorities := transformRR(json_.Authority, "authority")

	if json_.Comment != "" {
		Log.Infof(json_.Comment)
	}

	rMsg.Answer = answers
	rMsg.Ns = authorities

	msg_ := rMsg.SetReply(qMsg)
	return msg_
}

// for a given []DNSRR, transform to dns.RR, logging if any errors occur
func transformRR(rrs []DNSRR, logType string) []dns.RR {
	var t []dns.RR

	for _, r := range rrs {
		if rr, err := r.RR(); err != nil {
			Log.Errorln("unable to translate record rr", logType, r, err)
		} else {
			t = append(t, rr)
		}
	}

	return t
}

func placeSubnetToMsg(subnet string, msg *dns.Msg) {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		Log.Debugf("subnet is not OK: %v", subnet)
	} else {
		mask := ipNet.Mask
		// mask bits count.
		ones, bits := mask.Size()
		if ones <= 0 {
			ones = bits
		}
		maskUint := uint8(ones)

		// 1 for IP, 2 for IP6, must be 0 when sourceNetmask is 0
		family := uint16(1)

		if maskUint == 0 {
			family = 0
		} else if ipNet.IP.To4() == nil {
			family = 2
		}
		subnet := &dns.EDNS0_SUBNET{Code: dns.EDNS0SUBNET, SourceScope: 0,
			Address: ipNet.IP, SourceNetmask: maskUint, Family: family}
		ReplaceEDNS0Subnet(msg, subnet)
	}
}

func CalculatePaddingLength(preAllocatedLen int, least int, gain int) int {
	paddingLength := 0
	for i := 0; ; i++ {
		nextDesireTotalLen := least + i*gain
		if nextDesireTotalLen >= preAllocatedLen {
			paddingLength = nextDesireTotalLen - preAllocatedLen
			Log.Debugf("padding length: %v", paddingLength)
			break
		}
	}
	return paddingLength
}
