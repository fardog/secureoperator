package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/miekg/dns"

	secop "github.com/fardog/secureoperator"
	"github.com/fardog/secureoperator/cmd"
)

var (
	listenAddress = flag.String(
		"listen", ":53", "listen address, as `[host]:port`",
	)

	noPad = flag.Bool(
		"no-pad",
		false,
		"Disable padding of Google DNS-over-HTTPS requests to identical length",
	)

	logLevel = flag.String(
		"level",
		"info",
		"Log level, one of: debug, info, warn, error, fatal, panic",
	)

	// resolution of the Google DNS endpoint; the interaction of these values is
	// somewhat complex, and is further explained in the help message.
	endpoint = flag.String(
		"endpoint",
		"https://dns.google.com/resolve",
		"Google DNS-over-HTTPS endpoint url",
	)
	endpointIPs = flag.String(
		"endpoint-ips",
		"",
		`IPs of the Google DNS-over-HTTPS endpoint; if provided, endpoint lookup is
        skipped, and the host value in "endpoint" is sent as the Host header. Comma
        separated with no spaces; e.g. "74.125.28.139,74.125.28.102". One server is
        randomly chosen for each request, failed requests are not retried.`,
	)
	dnsServers = flag.String(
		"dns-servers",
		"",
		`DNS Servers used to look up the endpoint; system default is used if absent.
        Ignored if "endpoint-ips" is set. Comma separated, e.g. "8.8.8.8,8.8.4.4:53".
        The port section is optional, and 53 will be used by default.`,
	)
	autoEDNS = flag.Bool(
		"auto-edns-subnet",
		false,
		`By default, we use an EDNS subnet of 0.0.0.0/0 which does not reveal your
        IP address or subnet to authoratative DNS servers. If privacy of your IP
        address is not a concern and you want to take advantage of an authoratative
        server determining the best DNS results for you, set this flag. This flag
        specifies that Google should choose what subnet to send; if you'd like to
        specify your own subnet, use the -edns-subnet option.`,
	)
	ednsSubnet = flag.String(
		"edns-subnet",
		secop.GoogleEDNSSentinelValue,
		`Specify a subnet to be sent in the edns0-client-subnet option; by default
        we specify that this option should not be used, for privacy. If
        -auto-edns-subnet is used, the value specified here is ignored.
       `,
	)

	enableTCP = flag.Bool("tcp", true, "Listen on TCP")
	enableUDP = flag.Bool("udp", true, "Listen on UDP")
)

func serve(net string) {
	log.Infof("starting %s service on %s", net, *listenAddress)

	server := &dns.Server{Addr: *listenAddress, Net: net, TsigSecret: nil}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to setup the %s server: %s\n", net, err.Error())
		}
	}()

	// serve until exit
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Infof("shutting down %s on interrupt\n", net)
	if err := server.Shutdown(); err != nil {
		log.Errorf("got unexpected error %s", err.Error())
	}
}

func main() {
	flag.Usage = func() {
		_, exe := filepath.Split(os.Args[0])
		fmt.Fprint(os.Stderr, "A DNS-protocol proxy for Google's DNS-over-HTTPS service.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\n  %s [options]\n\nOptions:\n\n", exe)
		flag.PrintDefaults()
	}
	flag.Parse()

	// seed the global random number generator, used in some utilities and the
	// google provider
	rand.Seed(time.Now().UTC().UnixNano())

	// set the loglevel
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatalf("invalid log level: %s", err.Error())
	}
	log.SetLevel(level)

	eips, err := cmd.CSVtoIPs(*endpointIPs)
	if err != nil {
		log.Fatalf("error parsing endpoint-ips: %v", err)
	}
	dips, err := cmd.CSVtoEndpoints(*dnsServers)
	if err != nil {
		log.Fatalf("error parsing dns-servers: %v", err)
	}

	edns := *ednsSubnet
	if *autoEDNS {
		edns = ""
	}
	if _, _, err := net.ParseCIDR(edns); edns != "" && err != nil {
		log.Fatal(err)
	}
	if edns != secop.GoogleEDNSSentinelValue {
		log.Warn("EDNS will be used; authoritative name servers may be able to determine your location")
	}

	provider, err := secop.NewGDNSProvider(*endpoint, &secop.GDNSOptions{
		Pad:                 !*noPad,
		EndpointIPs:         eips,
		DNSServers:          dips,
		UseEDNSsubnetOption: true,
		EDNSSubnet:          edns,
	})
	if err != nil {
		log.Fatal(err)
	}
	options := &secop.HandlerOptions{}
	handler := secop.NewHandler(provider, options)

	dns.HandleFunc(".", handler.Handle)

	// push the list of enabled protocols into an array
	var protocols []string
	if *enableTCP {
		protocols = append(protocols, "tcp")
	}
	if *enableUDP {
		protocols = append(protocols, "udp")
	}

	// start the servers
	servers := make(chan bool)
	for _, protocol := range protocols {
		go func(protocol string) {
			serve(protocol)
			servers <- true
		}(protocol)
	}

	// wait for servers to exit
	for i := 0; i < len(protocols); i++ {
		<-servers
	}

	log.Infoln("servers exited, stopping")
}
