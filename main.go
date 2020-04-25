package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	nested_formatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/zput/zxcTool/ztLog/zt_formatter"
)

const (
	gdnsEndpoint       = "https://dns.google/dns-query"
)

// Create a new instance of the logger. You can have any number of instances.
var log = logrus.New()

var (
	listenAddressFlag = flag.String(
		"listen", ":53", "listen address, as `[host]:port`",
	)

	logLevelFlag = flag.String(
		"loglevel",
		"info",
		"Log level, one of: debug, info, warn, error, fatal, panic",
	)
	googleFlag = flag.Bool(
		"google",
		false,
		fmt.Sprintf(`Alternative google url scheme for dns.google/resolve.`),
	)
	// resolution of the Google DNS endpoint; the interaction of these values is
	// somewhat complex, and is further explained in the help message.
	endpointFlag = flag.String(
		"endpoint",
		gdnsEndpoint,
		"DNS-over-HTTPS endpoint url",
	)
	endpointIPsFlag = flag.String(
		"endpoint-ips",
		"",
		`IPs of the DNS-over-HTTPS endpoint; if provided, endpoint lookup is
skipped, the TLS establishment will direct hit the "endpoint-ips". Comma
separated with no spaces; e.g. "74.125.28.139,74.125.28.102". One server is
randomly chosen for each request, failed requests are not retried.`,
	)
	ednsSubnetFlag = flag.String(
		"edns-subnet",
		"no",
		`Specify a subnet to be sent in the edns0-client-subnet option;
take your own risk of privacy to use this option;
no: will not use edns_subnet;
auto: will use your current external IP address;
net/mask: will use specified subnet, e.g. 66.66.66.66/24.
       `,
	)

	cacheFlag = flag.Bool("cache", true, "Cache the dns answers")

	enableTCPFlag = flag.Bool("tcp", true, "Listen on TCP")
	enableUDPFlag = flag.Bool("udp", true, "Listen on UDP")

	// variables set in main body
	headersFlag     = make(KeyValue)
	queryParameters = make(KeyValue)

	http2Flag = flag.Bool(
		"http2",
		false,
		"Using http2 for query connection",
	)

	cacertFlag = flag.String(
		"cacert",
		"",
		"CA certificate for TLS establishment",
	)

	noAAAAFlag = flag.Bool(
		"no-ipv6",
		false,
		`Reply all AAAA questions with a fake answer`,
	)
)

func serve(net string) {
	log.Infof("starting %s service on %s", net, *listenAddressFlag)

	server := &dns.Server{Addr: *listenAddressFlag, Net: net, TsigSecret: nil}
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
		headersFlag,
		"headers",
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
		_, _ = fmt.Fprint(os.Stderr, "A DNS-protocol proxy for Google's DNS-over-HTTPS service.\n\n")
		_, _ = fmt.Fprintf(os.Stderr, "Usage:\n\n  %s [options]\n\nOptions:\n\n", exe)
		flag.PrintDefaults()
	}
	flag.Parse()

	// seed the global random number generator, used in some utilities and the
	// google provider
	rand.Seed(time.Now().UTC().UnixNano())

	// set the loglevel
	level, err := logrus.ParseLevel(*logLevelFlag)
	if err != nil {
		log.Fatalf("invalid log level: %s", err.Error())
	}
	log.SetLevel(level)
	fmt.Println("log level: ", log.GetLevel())
	log.SetReportCaller(true)
	log.SetFormatter(&zt_formatter.ZtFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
		},
		Formatter: nested_formatter.Formatter{
			// HideKeys: true,
			FieldsOrder: []string{"component", "category"},
		},
	})

	endpointIps, err := CSVtoIPs(*endpointIPsFlag)
	if err != nil {
		log.Fatalf("error parsing endpoint-ips: %v", err)
	}
	if err != nil {
		log.Fatalf("error parsing dns-servers: %v", err)
	}

	ep := *endpointFlag
	opts := &GDNSOptions{
		EndpointIPs:     endpointIps,
		EDNSSubnet:      *ednsSubnetFlag,
		QueryParameters: map[string][]string(queryParameters),
		Headers:         http.Header(headersFlag),
		HTTP2:           *http2Flag,
		CACertFilePath:  *cacertFlag,
		NoAAAA:          *noAAAAFlag,
		Alternative:	 *googleFlag,
	}

	provider, err := NewGDNSProvider(ep, opts)
	if err != nil {
		log.Fatal(err)
	}
	options := &HandlerOptions{Cache: *cacheFlag}
	handler := NewHandler(provider, options)

	dns.HandleFunc(".", handler.Handle)

	// push the list of enabled protocols into an array
	var protocols []string
	if *enableTCPFlag {
		protocols = append(protocols, "tcp")
	}
	if *enableUDPFlag {
		protocols = append(protocols, "udp")
	}

	// start the servers
	servers := make(chan bool)
	for _, protocol := range protocols {
		go func(protocol string, c <- chan bool) {
			serve(protocol)
		}(protocol, servers)
		servers <- true
	}

	for 

	log.Infoln("servers exited, stopping")
}
