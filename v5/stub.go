package dohProxy

import (
	"fmt"
	"github.com/miekg/dns"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Stub struct {
	ListenAddr       string
	UpstreamAddr     string
	UpstreamProtocol string // tcp or udp
	UseCache         bool
}

var (
	client          *dns.Client
	conns           []*dns.Conn
	connsAddrs      []string
	nextConnIdx     int = 0
	cache           *Cache
	subnetException map[string]bool
)

func (stub Stub) addSubnetException(){
	subnetException = make(map[string]bool)
	subnetException["cloudfront.net"] = true
	subnetException["recaptcha.net"] = true
	subnetException["gstatic.com"] = true
	subnetException["google-analytics.com"] = true
	subnetException["googlesyndication.com"] = true
	subnetException["googletagmanager.com"] = true
	subnetException["doubleclick.net"] = true
	subnetException["google.com"] = true
	subnetException["googletagservices.com"] = true
	subnetException["googleapis.com"] = true
	subnetException["googleusercontent.com"] = true
	subnetException["ggpht.com"] = true
	subnetException["ytimg.com"] = true
	subnetException["youtube-nocookie.com"] = true
	subnetException["youtube.com"] = true
	subnetException["googlevideo.com"] = true
}

func (stub Stub) ifUseSubnet(qName string) bool {
	name := strings.TrimRight(qName, ".")
	if name == ""{
		return true
	}
	_, ok := subnetException[name]
	if ok {
		return false
	}
	idxFirstDot := strings.Index(name, ".")
	if idxFirstDot != -1 {
		tname := strings.TrimLeft(name[idxFirstDot+1:], ".")
		_, ok = subnetException[tname]
		if ok {
			return false
		}
	}
	return true
}

func (stub Stub) ensureConn() (*dns.Conn, error) {
	if client == nil {
		client = &dns.Client{
			Net:          stub.UpstreamProtocol,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			DialTimeout:  5 * time.Second,
		}
	}
	c, err := turnAddrRoundRobin()
	if err != nil {
		Log.Errorf("connect to upstream server %v://%v failed: %v",
			stub.UpstreamProtocol, connsAddrs[nextConnIdx], err)
		return nil, err
	}
	return c, nil
}

func turnAddrRoundRobin() (*dns.Conn, error) {
	nextConnIdx += 1
	if nextConnIdx >= len(connsAddrs) {
		nextConnIdx = 0
	}

	nextConnAddr := connsAddrs[nextConnIdx]
	var nextConn *dns.Conn
	if len(conns) >= nextConnIdx+1 {
		if conns[nextConnIdx] != nil {
			nextConn = conns[nextConnIdx]
		} else {
			nextConn_, err := client.Dial(nextConnAddr)
			if err != nil {
				return nil, err
			} else {
				nextConn = nextConn_
			}
		}
	} else {
		nextConn_, err := client.Dial(nextConnAddr)
		if err != nil {
			return nil, err
		} else {
			nextConn = nextConn_
			conns = append(conns, nextConn)
		}
	}
	return nextConn, nil
}

func (stub Stub) answer(w http.ResponseWriter, r *http.Request) {
	accept_in_req := r.Header.Get("Accept")
	if accept_in_req != "" && accept_in_req != "*/*" && accept_in_req != ContentType {
		Log.Errorf("request content type not supported: %v", accept_in_req)
		w.Header().Add("content-type", "text/plain")
		w.WriteHeader(http.StatusForbidden)
		_, err := w.Write([]byte("request content type not supported."))
		if err != nil {
			Log.Errorf("write message failed: %v", err)
			return
		}
		return
	}
	q, err := stub.generateMsgFromReq(r)
	if err != nil {
		Log.Errorf("get message from request failed: %v", err)
		w.Header().Add("content-type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, err := w.Write([]byte("get message from request failed."))
		if err != nil {
			Log.Errorf("write response failed: %v", err)
			return
		}
		return
	}

	if stub.UseCache {
		rMsg := cache.Get(q)
		if rMsg != nil {
			rMsg.Id = q.Id
			Log.Infof("resolved from cache")
			stub.writeAnswer(rMsg, w)
			return
		}
	}

	rMsg, err := stub.relay(q, false)
	if err != nil {
		Log.Errorf("error when querying upstream: %v", err)
		w.Header().Add("content-type", "text/plain")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, err := w.Write([]byte("error when querying upstream."))
		if err != nil {
			Log.Errorf("write response failed: %v", err)
			return
		}
		return
	}
	stub.writeAnswer(rMsg, w)
}

func (stub Stub) writeAnswer(rMsg *dns.Msg, w http.ResponseWriter) {
	bytes_4_write, err := rMsg.Pack()
	if err != nil {
		Log.Errorf("error when querying upstream: %v", err)
		w.Header().Add("content-type", "text/plain")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, err := w.Write([]byte("error when querying upstream."))
		if err != nil {
			Log.Errorf("write response failed: %v", err)
			return
		}
		return
	}
	w.WriteHeader(200)
	_, err = w.Write(bytes_4_write)
	if err != nil {
		Log.Errorf("error when writing response: %v", err)
		return
	}
}

func (stub Stub) relay(msg *dns.Msg, is_retry bool) (*dns.Msg, error) {
	c, err := stub.ensureConn()
	if err != nil {
		client = nil
		conns = nil
		return nil, fmt.Errorf("client connecting error")
	}
	rMsg, _, err := client.ExchangeWithConn(msg, c)
	if err != nil {
		client = nil
		conns = nil
		// retry once.
		if is_retry {
			Log.Errorf("error when relaying query: %v", err)
			return nil, err
		}
		Log.Infof("retrying query: %v", msg.Question[0].String())
		return stub.relay(msg, true)
	}
	if stub.UseCache {
		msgch := make(chan *dns.Msg)
		defer close(msgch)
		go cache.Insert(msgch)
		msgch <- rMsg
	}
	Log.Debugf("upstream answer: %v", rMsg)
	Log.Infof("resolved from upstream for: %v", rMsg.Question[0].String())
	return rMsg, nil
}

func (stub Stub) generateMsgFromReq(r *http.Request) (*dns.Msg, error) {
	qMsg := new(dns.Msg)
	qMsg.Id = dns.Id()
	qMsg.Response = false
	qMsg.Opcode = dns.OpcodeQuery
	qMsg.Authoritative = false
	qMsg.Truncated = false
	qMsg.RecursionAvailable = false
	qMsg.RecursionDesired = true
	qMsg.AuthenticatedData = true
	qMsg.CheckingDisabled = false

	qURL := r.URL.Query()
	qCT := qURL.Get("ct")
	if qCT != "" && qCT != ContentType {
		Log.Errorf("content type not supported: %v", qCT)
		return nil, fmt.Errorf("content type not supported")
	}

	qName := qURL.Get("name")
	qName = dns.CanonicalName(qName)
	if qName == "" {
		Log.Errorf("question name invalid: %v", qName)
		return nil, fmt.Errorf("question name invalid: %v", qName)
	}

	qType := qURL.Get("type")
	itype, err := strconv.Atoi(qType)
	if err != nil {
		Log.Errorf("question type invalid: %v", itype)
		return nil, fmt.Errorf("question type invalid")
	}
	qMsg = qMsg.SetQuestion(qName, uint16(itype))

	qSubnet := qURL.Get("edns_client_subnet")
	ip, ipnet, err := net.ParseCIDR(qSubnet)
	if ip == nil {
		ip = net.ParseIP(qSubnet)
	}
	if err != nil || ip == nil || !stub.ifUseSubnet(qName) {
		Log.Debugf("question subnet skipped, name: %v, subnet: %v",qName, qSubnet)
	} else {
		subnet := new(dns.EDNS0_SUBNET)
		subnet.Family = 0
		subnet.Code = dns.EDNS0SUBNET
		subnet.SourceScope = 0
		subnet.Address = ip
		is_ip4 := ip.To4() != nil
		ones_ := 32
		if is_ip4 {
			subnet.Family = 1
		} else {
			subnet.Family = 2
		}
		if ipnet != nil {
			ones_, _ = ipnet.Mask.Size()
			subnet.SourceNetmask = uint8(ones_)
		} else {
			if is_ip4 {
				subnet.SourceNetmask = 32
			} else {
				subnet.SourceNetmask = 128
			}
		}
		ReplaceEDNS0Subnet(qMsg, subnet)
	}

	Log.Infof("will query name: %v, type: %v, client_subnet: %v", qName, qType, qSubnet)

	return qMsg, nil
}

func (stub Stub) Run() {
	if stub.UseCache {
		cache = NewCache()
	}

	stub.addSubnetException()

	// get connection addresses.
	addrs := strings.Split(stub.UpstreamAddr, ",")
	for _, addr := range addrs {
		if addr == "" {
			continue
		}
		_, _, err := net.SplitHostPort(addr)
		if err == nil {
			connsAddrs = append(connsAddrs, addr)
		} else {
			ip := net.ParseIP(addr)
			if ip == nil {
				continue
			}
			connsAddrs = append(connsAddrs, net.JoinHostPort(ip.String(), "53"))
		}
	}

	if len(connsAddrs) == 0 {
		Log.Error("no valid upstream address.")
		return
	}

	http.HandleFunc("/resolve", stub.answer)
	Log.Infof("running stub server http://%v <--> %v://%v ...",
		stub.ListenAddr, stub.UpstreamProtocol, stub.UpstreamAddr)

	err := http.ListenAndServe(stub.ListenAddr, nil)
	if err != nil {
		Log.Fatalf("stub server running into error: %v", err)
	}
	for _, c := range conns {
		_ = c.Close()
	}
	Log.Info("stopping stub server...")
}

