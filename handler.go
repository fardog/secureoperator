package main

import (
	"github.com/miekg/dns"
)

// HandlerOptions specifies options to be used when instantiating a handler
type HandlerOptions struct{
	Cache bool
}

// Handler represents a DNS handler
type Handler struct {
	options  *HandlerOptions
	provider Provider
}

// NewHandler creates a new Handler
func NewHandler(provider Provider, options *HandlerOptions) *Handler {
	return &Handler{options, provider}
}

// Handle handles a DNS request
func (h *Handler) Handle(writer dns.ResponseWriter, msg *dns.Msg) {
	log.Infoln("requesting", msg.Question[0].Name, dns.TypeToString[msg.Question[0].Qtype])

	bytes, err := h.provider.Query(msg)
	if err != nil {
		log.Errorln("provider failed", err)
		dns.HandleFailed(writer, msg)
		return
	}

	// Write the response
	c, err := writer.Write(bytes)
	if err != nil {
		log.Errorln("Error writing DNS response:", err)
	}else{
		log.Debugln("Writen bytes:", c)
	}
}
