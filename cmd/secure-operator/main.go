package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/miekg/dns"

	secop "github.com/fardog/secureoperator"
)

var (
	listenAddress = flag.String(
		"listen", ":53", "listen address, as `[host]:port`",
	)

	endpoint = flag.String(
		"endpoint",
		"https://dns.google.com/resolve",
		"Google DNS-over-HTTPS endpoint url",
	)
	pad = flag.Bool(
		"pad",
		true,
		"Pad requests to identical length",
	)

	logLevel = flag.String(
		"level",
		"info",
		"Log level, one of: debug, info, warn, error, fatal, panic",
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
		flag.PrintDefaults()
	}
	flag.Parse()

	// set the loglevel
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatalf("invalid log level: %s", err.Error())
		return
	}
	log.SetLevel(level)

	provider := secop.GDNSProvider{
		Endpoint: *endpoint,
		Pad:      *pad,
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
