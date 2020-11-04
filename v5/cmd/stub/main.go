package main

import (
	"flag"
	"fmt"
	proxy "github.com/tinkernels/doh-proxy/v5"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

const (
	gdnsEndpoint = "https://dns.google/dns-query"
)

// Create a new instance of the logger. You can have any number of instances.
var log = proxy.Log

var (
	listenAddressFlag = flag.String(
		"listen", ":53", "Listen address, as `[host]:port`",
	)
	// resolution of the Google DNS endpoint; the interaction of these values is
	// somewhat complex, and is further explained in the help message.
	upstreamAddrFlag = flag.String(
		"upstream-addr",
		gdnsEndpoint,
		"Upstream dns server",
	)
	upstreamProtocolFlag = flag.String(
		"upstream-protocol",
		"tcp",
		"Upstream dns server protocol, tcp or udp",
	)
	logLevelFlag = flag.String(
		"loglevel",
		"info",
		"Log level, one of: debug, info, warn, error, fatal, panic",
	)
	cacheFlag = flag.Bool("cache", true, "Cache the dns answers")
	versionFlag = flag.Bool(
		"version",
		false,
		"Print version info",
	)
)

func printVersion(){
	fmt.Println("v5.0.1")
}

func serve(net <- chan string) {
	listenNet := <- net
	log.Infof("starting %s service on %s", listenNet, *listenAddressFlag)

	server := &dns.Server{Addr: *listenAddressFlag, Net: listenNet, TsigSecret: nil}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to setup the %s server: %s\n", listenNet, err.Error())
	}

	log.Infof("shutting down %s on interrupt\n", listenNet)
	if err := server.Shutdown(); err != nil {
		log.Errorf("got unexpected error %s", err.Error())
	}
}

func main() {
	flag.Usage = func() {
		_, exe := filepath.Split(os.Args[0])
		_, _ = fmt.Fprint(os.Stderr, "A DoH stub server.\n\n")
		_, _ = fmt.Fprintf(os.Stderr, "Usage:\n\n  %s [options]\n\nOptions:\n\n", exe)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *versionFlag {
		printVersion()
		return
	}

	// seed the global random number generator
	rand.Seed(time.Now().UTC().UnixNano())

	// set the loglevel
	level, err := logrus.ParseLevel(*logLevelFlag)
	if err != nil {
		log.Fatalf("invalid log level: %s", err.Error())
	}

	log.SetLevel(level)
	fmt.Println("log level: ", log.GetLevel())

	stub := proxy.Stub{
		ListenAddr:       *listenAddressFlag,
		UpstreamAddr:     *upstreamAddrFlag,
		UpstreamProtocol: *upstreamProtocolFlag,
		UseCache:         *cacheFlag,
	}
	stub.Run()
}
