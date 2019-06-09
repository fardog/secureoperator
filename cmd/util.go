package cmd

import (
	"fmt"
	"net"
	"strings"

	secop ".."
)

// CSVtoEndpoints takes a comma-separated string of endpoints, and parses to a
// []secop.Endpoint
func CSVtoEndpoints(csv string) (eps []secop.Endpoint, err error) {
	reps := strings.Split(csv, ",")
	for _, r := range reps {
		if r == "" {
			continue
		}

		ep, err := secop.ParseEndpoint(r, 53)
		if err != nil {
			return eps, err
		}

		eps = append(eps, ep)
	}

	return eps, err
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
