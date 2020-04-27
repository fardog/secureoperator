package main

import (
	"github.com/miekg/dns"
)

// HandlerOptions specifies options to be used when instantiating a handler
type HandlerOptions struct {
	Cache bool
}

// Handler represents a DNS handler
type Handler struct {
	options           *HandlerOptions
	provider          Provider
	hostsFileProvider Provider
}

// NewHandler creates a new Handler
func NewHandler(provider Provider, options *HandlerOptions) *Handler {
	handler := new(Handler)
	handler.options = options
	handler.provider = provider
	handler.hostsFileProvider = NewHostsFileProvider()
	return handler
}

// Handle handles a DNS request
func (h *Handler) Handle(writer dns.ResponseWriter, msg *dns.Msg) {

	log.Infoln("requesting", msg.Question[0].Name, dns.TypeToString[msg.Question[0].Qtype])

	isAnsweredCh := make(chan bool)
	defer close(isAnsweredCh)

	// lookup hosts file if retrieving ip address
	if msg.Question[0].Qtype == dns.TypeA || msg.Question[0].Qtype == dns.TypeAAAA {
		go h.AnswerByHostsFile(&writer, msg, isAnsweredCh)
		if <-isAnsweredCh {
			log.Debugf("resolved from hosts: %v",  msg.Question[0].Name)
			return
		}
	}

	go h.AnswerByDoH(&writer, msg, isAnsweredCh)
	if <-isAnsweredCh {
		log.Debugf("resolved from DoH: %v",  msg.Question[0].Name)
		return
	}

	dns.HandleFailed(writer, msg)
}

func (h *Handler) TryWriteAnswer(writer *dns.ResponseWriter, rMsg *dns.Msg, isAnsweredCh chan bool) {
	if rMsg != nil {
		// Write the response
		writerReal := *writer
		err := writerReal.WriteMsg(rMsg)
		if err != nil {
			log.Errorf("Error writing DNS response: %v", err)
			isAnsweredCh <- false
		} else {
			isAnsweredCh <- true
			log.Debugf("Successfully write response message")
		}
	} else {
		isAnsweredCh <- false
	}
}

func (h *Handler) AnswerByHostsFile(writer *dns.ResponseWriter,msg *dns.Msg, isOKCh chan bool) {

	msgR, err := h.hostsFileProvider.Query(msg)
	if err != nil {
		log.Debugf("hosts file provider failed: %v", err)
		isOKCh <- false
		return
	}
	go h.TryWriteAnswer(writer, msgR, isOKCh)
}

func (h *Handler) AnswerByDoH(writer *dns.ResponseWriter, msg *dns.Msg, isOKCh chan bool) {

	msgR, err := h.provider.Query(msg)
	if err != nil {
		log.Errorf("dns-message provider failed: %v", err)
		isOKCh <- false
		return
	}
	go h.TryWriteAnswer(writer, msgR, isOKCh)
}
