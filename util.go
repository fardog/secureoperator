package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

func GenerateUrlSafeString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// CSVtoIPs takes a comma-separated string of IPs, and parses to a []net.IP
func CSVtoIPs(csv string) (ips []net.IP, err error) {
	rs := strings.Split(csv, ",")

	for _, r := range rs {
		if r == "" {
			continue
		}

		ip := net.ParseIP(r)
		if ip == nil {
			return ips, fmt.Errorf("unable to parse IP from string %s", r)
		}
		ips = append(ips, ip)
	}

	return
}

type KeyValue map[string][]string

func (k KeyValue) Set(kv string) error {
	kvs := strings.SplitN(kv, "=", 2)
	if len(kvs) != 2 {
		return fmt.Errorf("invalid format for %v; expected KEY=VALUE", kv)
	}
	key, value := kvs[0], kvs[1]

	if vs, ok := k[key]; ok {
		k[key] = append(vs, value)
	} else {
		k[key] = []string{value}
	}

	return nil
}

func (k KeyValue) String() string {
	var s []string
	for k, vs := range k {
		for _, v := range vs {
			s = append(s, fmt.Sprintf("%v=%v", k, v))
		}
	}

	return strings.Join(s, " ")
}

func logDebug(args ...interface{}) {
	lvl, err := logrus.ParseLevel(*logLevelFlag)
	if err != nil {
		return
	}
	if log.IsLevelEnabled(lvl) {
		log.Debug(args...)
	}
}

func IsLocalListen(addr string) bool {
	localNets := []string{
		"127.0.0.1",
		"0.0.0.0",
		"::1",
		"::",
		"localhost",
	}
	h, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	for _, ch := range localNets {
		if ch == h {
			return true
		}
	}
	return false
}

func ResolveHostToIP(name string, resolver string) []string {
	ips := make(chan []string)
	ipResolver := net.ParseIP(resolver)
	if ipResolver != nil{
		resolver = net.JoinHostPort(ipResolver.String(), "53")
	}else{
		_, _, err := net.SplitHostPort(resolver)
		if err != nil {
			log.Error("Dns resolver can't be recognized: ", err)
			return nil
		}
	}

	mA := new(dns.Msg)
	mA.SetQuestion(name, dns.TypeA)

	go func() {
		var ipsResolved []string
		client := &dns.Client{}
		r, _, err := client.Exchange(mA, resolver)
		if err != nil {
			log.Error("can't resolve endpoint host with provided dns resolver:", err)
		}

		for _, ip := range r.Answer {
			ipv4 := ip.(*dns.A)
			if ipv4 != nil && ipv4.A != nil {
				ipsResolved = append(ipsResolved, ipv4.A.String())
			}
		}
		log.Infof("ips resolved by dns: %v -> %v",name, ipsResolved)
		ips <- ipsResolved
	}()
	return <- ips
}

func ObtainCurrentExternalIP(dnsResolver string) (string,error){
	ip := ""
	type IPRespModel struct {
		Address string `json:"address"`
		Ip      string `json:"ip"`
	}

	apiToTry := []string{
		"https://wq.apnic.net/ip",
		"https://accountws.arin.net/public/seam/resource/rest/myip",
		"https://rdap.lacnic.net/rdap/info/myip",
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// custom transport for supporting servernames which may not match the url,
	// in cases where we request directly against an IP
	tr := &http.Transport{
		DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			h, p, _ := net.SplitHostPort(addr)
			var ipResolved []string
			ipResolved = ResolveHostToIP(h + ".", dnsResolver)

			if ipResolved == nil{
				log.Errorf("Can't resolve endpoint %v from provided dns server %v", h, dnsResolver)
				return dialer.DialContext(ctx, network, addr)
			}else if len(ipResolved) == 0 {
				log.Debugf("Resolve answder of endpoint %v is empty from provided dns server %v",
					h, dnsResolver)
				return dialer.DialContext(ctx, network, addr)
			}
			ip := ipResolved[rand.Intn(len(ipResolved))]
			addr = net.JoinHostPort(ip, p)
			log.Info("endpoint ip address from dns-resolver: ", addr)

			return dialer.DialContext(ctx, network, addr)
		},
	}

	client := &http.Client{Transport: tr}

	for _, uri := range apiToTry{
		log.Debugf("start obtain external ip from: %v", uri)
		httpReq, err := http.NewRequest(http.MethodGet, uri, nil)
		if err != nil {
			log.Errorf("retrieve external ip error: %v", err)
			continue
		}
		httpResp, err := client.Do(httpReq)
		if err != nil{
			log.Errorf("http api call failed: %v", err)
			continue
		}
		ipResp := new(IPRespModel)
		httpRespBytes, err:= ioutil.ReadAll(httpResp.Body)
		if err != nil {
			log.Errorf("http api call result read error: %v, %v",httpRespBytes, err)
		}
		err = json.Unmarshal(httpRespBytes, &ipResp)
		if err != nil{
			log.Errorf("retrieve external ip error: %v", err)
			continue
		}
		if ipResp.Ip != ""{
			ip = ipResp.Ip
			log.Errorf("API result of obtain external ip: %v", ipResp)
		}
		if ipResp.Address != ""{
			ip = ipResp.Address
			log.Infof("API result of obtain external ip: %v", ipResp)
		}
		if ip != "" {
			break
		}
	}

	if ip == ""{
		return "", errors.New("can't obtain your external ip address")
	}
	return ip, nil
}
