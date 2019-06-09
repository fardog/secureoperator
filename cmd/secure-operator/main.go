package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/miekg/dns"

	secop "../.."
	cmd ".."
	//"github.com/fardog/secureoperator/cmd"
)

const (
	gdnsEndpoint       = "https://dns.google.com/resolve"
	cloudflareEndpoint = "https://cloudflare-dns.com/dns-query"
	quad9Endpoint      = "https://dns.quad9.net/dns-query"
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

	http2 = flag.Bool(
		"http2",
		false,
		"Using http2 for query connection",
	)

	// one-stop configuration flags; when used, these configure sane defaults
	google = flag.Bool(
		"google",
		false,
		fmt.Sprintf(`Use Google defaults. When set, the following options will be used unless
explicitly overridden:
	dns-servers: 8.8.8.8,8.8.4.4
	endpoint: %v`, gdnsEndpoint),
	)
	cloudflare = flag.Bool(
		"cloudflare",
		false,
		fmt.Sprintf(`Use Cloudflare defaults. When set, the following options will be used
unless explicitly overridden:
	dns-servers: 1.0.0.1,1.1.1.1
	params: ct=application/dns-json
	endpoint: %v`, cloudflareEndpoint),
	)
	quad9 = flag.Bool(
		"quad9",
		false,
		fmt.Sprintf(`Use Quad9 defaults. When set, the following options will be used
unless explicitly overriden:
	dns-servers: 9.9.9.9, 149.112.112.112
	params: ct=application/dns-json
	endpoint : %v`, quad9Endpoint),
	)
	// resolution of the Google DNS endpoint; the interaction of these values is
	// somewhat complex, and is further explained in the help message.
	endpoint = flag.String(
		"endpoint",
		gdnsEndpoint,
		"DNS-over-HTTPS endpoint url",
	)
	endpointIPs = flag.String(
		"endpoint-ips",
		"",
		`IPs of the DNS-over-HTTPS endpoint; if provided, endpoint lookup is
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

	// variables set in main body
	headers         = make(cmd.KeyValue)
	queryParameters = make(cmd.KeyValue)
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
	// non-standard flag vars
	flag.Var(
		headers,
		"header",
		`Additional headers to be sent with http requests, as Key=Value; specify
multiple as:
    -header Key-1=Value-1-1 -header Key-1=Value1-2 -header Key-2=Value-2`,
	)
	flag.Var(
		queryParameters,
		"param",
		`Additional query parameters to be sent with http requests, as key=value;
specify multiple as:
    -param key1=value1-1 -param key1=value1-2 -param key2=value2`,
	)
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

	if *google && *cloudflare || *google && *quad9 ||
		*cloudflare && *quad9 || *google && *cloudflare && *quad9 {
		log.Fatalf("you may not specify `-google` and `-cloudflare` and `-quad9` arguments together")
	}

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

	ep := *endpoint
	opts := &secop.GDNSOptions{
		Pad:                 !*noPad,
		EndpointIPs:         eips,
		DNSServers:          dips,
		UseEDNSsubnetOption: true,
		EDNSSubnet:          edns,
		QueryParameters:     map[string][]string(queryParameters),
		Headers:             http.Header(headers),
		HTTP2:               *http2,
	}

	// handle "sane defaults" if requested; only where settings are not explicitly
	// provided by the user
	if *google {
		if len(opts.DNSServers) == 0 {
			opts.DNSServers = []secop.Endpoint{
				secop.Endpoint{IP: net.ParseIP("8.8.8.8"), Port: 53},
				secop.Endpoint{IP: net.ParseIP("8.8.4.4"), Port: 53},
			}
		}
	} else if *cloudflare {
		// override only if it's currently the default
		if ep == gdnsEndpoint {
			ep = cloudflareEndpoint
		}
		if len(opts.DNSServers) == 0 {
			opts.DNSServers = []secop.Endpoint{
				secop.Endpoint{IP: net.ParseIP("1.0.0.1"), Port: 53},
				secop.Endpoint{IP: net.ParseIP("1.1.1.1"), Port: 53},
			}
		}
		if _, ok := opts.QueryParameters["ct"]; !ok {
			opts.QueryParameters["ct"] = []string{"application/dns-json"}
		}
	} else if *quad9 {
		// override only if it's currently the default
		if ep == gdnsEndpoint {
			ep = quad9Endpoint
		}
		if len(opts.DNSServers) == 0 {
			opts.DNSServers = []secop.Endpoint{
				secop.Endpoint{IP: net.ParseIP("9.9.9.9"), Port: 53},
				secop.Endpoint{IP: net.ParseIP("149.112.112.112"), Port: 53},
			}
		}
		if _, ok := opts.QueryParameters["ct"]; !ok {
			opts.QueryParameters["ct"] = []string{"application/dns-json"}
		}
	}

	provider, err := secop.NewGDNSProvider(ep, opts)
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
