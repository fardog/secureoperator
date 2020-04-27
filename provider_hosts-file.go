package main

import (
	"errors"
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
		log.Debugf("set %v hosts path: %v", runtime.GOOS, provider.resolver.path)
	} else {
		provider.resolver.path = filepath.FromSlash("/etc/hosts")
		log.Debugf("set %v hosts path: %v", runtime.GOOS, provider.resolver.path)
	}
	return provider
}

func (provider *HostsFileProvider) Query(msg *dns.Msg) (*dns.Msg, error) {
	localhostName := "localhost"
	localhostIP4 := "127.0.0.1"
	localhostIP16 := "::1"

	qName := msg.Question[0].Name
	qType := msg.Question[0].Qtype

	host, _, _ := net.SplitHostPort(strings.TrimSuffix(qName, "."))

	if host == "" {
		host = strings.TrimSuffix(qName, ".")
	}

	ipHost := net.ParseIP(host)
	isHostIP := func() bool {
		return ipHost != nil
	}()

	rMsg := new(dns.Msg)
	rMsg.SetReply(msg)
	if isHostIP {
		isIP4 := func() bool {
			return ipHost.To4() != nil
		}
		if (qType == dns.TypeAAAA && isIP4()) || (qType == dns.TypeA && !isIP4()) {
			return nil, errors.New(fmt.Sprintf("IP %v mismatch the question type %v",
				ipHost, dns.Type(qType).String()))
		}
		rMsg.Answer = make([]dns.RR, 1)
		rMsg.Answer[0] = genAnswerFromIP(qType, qName, host)
	} else if qType == dns.TypeA && host == localhostName {
		rMsg.Answer = make([]dns.RR, 1)
		rMsg.Answer[0] = genAnswerFromIP(qType, qName, localhostIP4)
	} else if qType == dns.TypeAAAA && host == localhostName {
		rMsg.Answer = make([]dns.RR, 1)
		rMsg.Answer[0] = genAnswerFromIP(qType, qName, localhostIP16)
	} else {
		// remove \ from host so that host will keep orginal
		ips := provider.resolver.LookupStaticHost(strings.ReplaceAll(host,"\\",""))
		rMsg.Answer = make([]dns.RR, len(ips))
		for i, ip := range ips {
			rMsg.Answer[i] = genAnswerFromIP(qType, qName, ip)
		}
	}

	if rMsg.Answer == nil || len(rMsg.Answer) == 0 {
		return nil, errors.New("no answer form hostsfile")
	}
	log.Debugf("hosts resolved:\n %v", rMsg)
	return rMsg, nil
}

func genAnswerFromIP(t uint16, name string, ip string) dns.RR {
	if t == dns.TypeA {
		r := &dns.A{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.ParseIP(ip),
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
			AAAA: net.ParseIP(ip),
		}
		return r
	} else {
		return nil
	}
}
