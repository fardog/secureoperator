package dohProxy

import (
	"fmt"
	nestedFormatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/zput/zxcTool/ztLog/zt_formatter"
	"math/rand"
	"net"
	"path"
	"runtime"
	"strings"
	"time"
)

var(
	Log = NewLogger()
)

func NewLogger()*logrus.Logger{
	log := logrus.New()
	log.SetReportCaller(true)

	// use logrus default TextFormatter to get the IsColored() method.
	defaultTextFormatter := logrus.TextFormatter{}
	_, _ = defaultTextFormatter.Format(&logrus.Entry{Logger: logrus.New()})
	isColoredLog := defaultTextFormatter.IsColored()
	log.SetFormatter(&zt_formatter.ZtFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
		},
		Formatter: nestedFormatter.Formatter{
			FieldsOrder: []string{"component", "category"},
			NoColors: !isColoredLog,
			NoFieldsColors: !isColoredLog,
		},
	})
	return log
}
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

func ObtainEDN0Subnet(msg *dns.Msg) (edns0Subnet dns.EDNS0_SUBNET) {
	var edns0 = msg.IsEdns0()
	if edns0 != nil {
		for _, o := range edns0.Option {
			switch o.(type) {
			case *dns.EDNS0_SUBNET:
				subnet := o.(*dns.EDNS0_SUBNET)
				return *subnet
			}
		}
	}
	return dns.EDNS0_SUBNET{}
}

func ReplaceEDNS0Subnet(msg *dns.Msg, subnet *dns.EDNS0_SUBNET) {
	var edns0 = msg.IsEdns0()
	if edns0 != nil {
		if edns0.Option != nil && len(edns0.Option) > 0 {
			for i, o := range edns0.Option {
				switch o.(type) {
				case *dns.EDNS0_SUBNET:
					if subnet == nil{
						// nil will panic.
						edns0.Option = append([]dns.EDNS0{subnet}, edns0.Option...)
					}else{
						edns0.Option[i] = subnet
					}
					return
				}
			}
			edns0.Option = append([]dns.EDNS0{subnet}, edns0.Option...)
		} else {
			edns0.Option = append([]dns.EDNS0{subnet}, edns0.Option...)
		}
	} else if subnet != nil {
		opt := &dns.OPT{Hdr: dns.RR_Header{Name: ".", Rrtype: dns.TypeOPT},
			Option: []dns.EDNS0{subnet}}
		msg.Extra = append(msg.Extra, opt)
	}
}

func ReplaceEDNS0Padding(msg *dns.Msg, padding *dns.EDNS0_PADDING) {
	var edns0 = msg.IsEdns0()
	if edns0 != nil {
		if edns0.Option != nil && len(edns0.Option) > 0 {
			for i, o := range edns0.Option {
				switch o.(type) {
				case *dns.EDNS0_PADDING:
					if padding == nil{
						// nil will panic.
						edns0.Option = append([]dns.EDNS0{padding}, edns0.Option...)
					}else{
						edns0.Option[i] = padding
					}
					return
				}
			}
			edns0.Option = append([]dns.EDNS0{padding}, edns0.Option...)
		} else {
			edns0.Option = append([]dns.EDNS0{padding}, edns0.Option...)
		}
	} else if padding != nil {
		opt := &dns.OPT{Hdr: dns.RR_Header{Name: ".", Rrtype: dns.TypeOPT},
			Option: []dns.EDNS0{padding}}
		msg.Extra = append(msg.Extra, opt)
	}
}

func GetMinTTLFromDnsMsg(msg *dns.Msg) (minTTL uint32) {
	// cache will expire even ttl is greater than 3600 (1hr)
	minTTL = uint32(3600)
	if msg == nil {
		return 0
	}
	if len(msg.Answer) == 0 && len(msg.Ns) == 0 {
		minTTL = 60
	} else {
		for _, rs :=
		range [][]dns.RR{msg.Answer, msg.Ns} {
			for _, r := range rs {
				ttl := r.Header().Ttl
				if ttl < minTTL {
					minTTL = ttl
				}
			}
		}
	}
	if minTTL < 0 {
		return 0
	}
	return minTTL
}

func InsertIntoSlice(to []interface{}, from interface{}, inex int) []interface{} {
	return append(to[:inex], append([]interface{}{from}, to[inex:]...)...)
}

// resolve domain name to ips (ipv4 + ipv6) using traditional udp+tcp, fixed 60s ttl
func ResolveHostToIPClosure(name string, resolver string) (closure func()(ip4s []string, ip6s []string)) {
	var ip4s, ip16s []string

	const ttl = int64(60)
	expireTime := time.Now().Unix()

	resolve := func() {
		ipResolver := net.ParseIP(resolver)
		if ipResolver != nil {
			resolver = net.JoinHostPort(ipResolver.String(), "53")
		} else {
			_, _, err := net.SplitHostPort(resolver)
			if err != nil {
				Log.Error("Dns resolver can't be recognized: ", err)
				return
			}
		}
		mA4 := new(dns.Msg)
		mA4.SetQuestion(dns.CanonicalName(name), dns.TypeA)
		mA6 := new(dns.Msg)
		mA6.SetQuestion(name, dns.TypeAAAA)
		for _, dnsNet := range []string{"tcp", "udp"} {
			client := &dns.Client{Net: dnsNet}
			// ipv4
			r4, _, err := client.Exchange(mA4.Copy(), resolver)
			if err != nil {
				Log.Errorf("can't resolve endpoint host with provided dns resolver over %v: %v", dnsNet, err)
				continue
			} else {
				if r4.Answer != nil {
					for _, answer := range r4.Answer {
						switch answer.(type) {
						case *dns.A:
							ipv := answer.(*dns.A)
							if ipv != nil {
								ip4s = append(ip4s, ipv.A.String())
							}
						}
					}
				}

			}
			// ipv6
			r6, _, err := client.Exchange(mA6.Copy(), resolver)
			if err != nil {
				Log.Errorf("can't resolve endpoint host with provided dns resolver over %v: %v", dnsNet, err)
				continue
			} else {
				if r6.Answer != nil {
					for _, answer := range r6.Answer {
						switch answer.(type) {
						case *dns.AAAA:
							ipv := answer.(*dns.AAAA)
							if ipv != nil {
								ip16s = append(ip16s, ipv.AAAA.String())
							}
						}
					}
				}
			}
		}
		expireTime = time.Now().Unix() + ttl
	}
	return func()([]string,[]string){
		if len(ip4s) == 0 && len(ip16s) == 0{
			Log.Infof("no cache.")
			resolve()
		}else if time.Now().Unix() > expireTime {
			go resolve()
		}

		// if empty result, ttl reset to 1
		if len(ip4s) == 0 && len(ip16s) == 0{
			expireTime = time.Now().Unix() +1
		}
		Log.Infof("using cache.")
		return ip4s, ip16s
	}
}
