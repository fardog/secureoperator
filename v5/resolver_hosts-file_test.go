package dohProxy

import (
	"net"
	"testing"
)

func TestResolveFromHostsFile(t *testing.T){

	localhostStr := "localhost"
	ipsReal, _ := net.LookupHost(localhostStr)
	t.Log("use net.LookupHost resolved localhost to: ", ipsReal)

	hostsResolver := new(HostsFileResolver)
	ips := hostsResolver.LookupStaticHost("rmbp@tinker.")
	t.Log("use hosts file resolved localhost to: ", ips)

	for _, ip := range ipsReal{
		ret := false
		for _, ip2test := range ips{
			if ip == ip2test{
				ret = true
			}
		}
		if !ret{
			Log.Println("localhost ip must be resolved to isn't resolved:", ip)
		}
	}
}