package dohProxy

import (
	"fmt"
	"github.com/miekg/dns"
	"net"
	"path/filepath"
	"runtime"
	"strings"
)

type HostsFileProvider struct {
	// using net.lookupStaticHost
	resolver HostsFileResolver
}

func NewHostsFileProvider() *HostsFileProvider {
	provider := new(HostsFileProvider)
	if runtime.GOOS == "windows" {
		provider.resolver.path = filepath.FromSlash("C:/Windows/System32/drivers/etc/hosts")
		Log.Debugf("set %v hosts path: %v", runtime.GOOS, provider.resolver.path)
	} else {
		provider.resolver.path = filepath.FromSlash("/etc/hosts")
		Log.Debugf("set %v hosts path: %v", runtime.GOOS, provider.resolver.path)
	}
	return provider
}

func (provider *HostsFileProvider) Query(msg *dns.Msg) (*dns.Msg, error) {
	// sounds ridiculous, some program on macOS resolve domain with port,e.g. localhost:1080.
	// so, remove port in dns question and pin the localhost resolve when using hosts file resolver.
	localhostName := "localhost"
	localhostIP4 := "127.0.0.1"
	localhostIP16 := "::1"

	qName := dns.CanonicalName(msg.Question[0].Name)
	qType := msg.Question[0].Qtype

	host := strings.TrimSuffix(qName, ".")

	rMsg := new(dns.Msg)
	rMsg.SetReply(msg)

	if qType == dns.TypeA {
		if qName == localhostName {
			rMsg.Answer = make([]dns.RR, 1)
			rMsg.Answer[0] = genAnswerFromIP(qType, qName, net.ParseIP(localhostIP4))
		}else{
			ips := provider.resolver.LookupStaticHost(strings.ReplaceAll(host,"\\",""))
			for _, ip := range ips {
				ip_n := net.ParseIP(ip)
				if ip_n.To4() == nil {
					continue
				}
				rMsg.Answer = append(rMsg.Answer, genAnswerFromIP(qType, qName, ip_n))
			}
		}
	}else if qType == dns.TypeAAAA{
		if qName == localhostName {
			rMsg.Answer = make([]dns.RR, 1)
			rMsg.Answer[0] = genAnswerFromIP(qType, qName, net.ParseIP(localhostIP16))
		}else{
			ips := provider.resolver.LookupStaticHost(strings.ReplaceAll(host,"\\",""))
			for _, ip := range ips {
				ip_n := net.ParseIP(ip)
				if ip_n.To4() != nil {
					continue
				}
				rMsg.Answer = append(rMsg.Answer, genAnswerFromIP(qType, qName, ip_n))
			}
		}
	}else{
		return nil, fmt.Errorf("can't resolve Qtype other than A, AAAA")
	}
	if len(rMsg.Answer) == 0 {
		return nil, fmt.Errorf("no answer form hostsfile")
	}
	Log.Debugf("hosts resolved:\n %v", rMsg)
	return rMsg, nil
}

func genAnswerFromIP(t uint16, name string, ip net.IP) dns.RR {
	if t == dns.TypeA {
		r := &dns.A{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: ip,
		}
		return r
	} else if t == dns.TypeAAAA {
		r := &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			AAAA: ip,
		}
		return r
	} else {
		return nil
	}
}
