package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"strings"
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

func IsLocalListen(addr string)bool{
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
	for _, ch := range localNets{
		if ch == h {
			return true
		}
	}
	return false
}

func obtainCurrentExternalIP() {

}
