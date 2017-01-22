package secureoperator

import (
	log "github.com/Sirupsen/logrus"
	"github.com/miekg/dns"
)

// HandlerOptions specifies options to be used when instantiating a handler
type HandlerOptions struct{}

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
func (h *Handler) Handle(w dns.ResponseWriter, r *dns.Msg) {
	q := DNSQuestion{
		Name: r.Question[0].Name,
		Type: int32(r.Question[0].Qtype),
	}
	log.Infoln("requesting", q.Name, q.Type)

	dnsResp, err := h.provider.Query(q)
	if err != nil {
		log.Errorln("provider failed", err)
		dns.HandleFailed(w, r)
		return
	}

	questions := []dns.Question{}
	for idx, c := range dnsResp.Question {
		questions = append(questions, dns.Question{
			Name:   c.Name,
			Qtype:  uint16(c.Type),
			Qclass: r.Question[idx].Qclass,
		})
	}

	// Parse google RRs to DNS RRs
	answers := []dns.RR{}
	for _, a := range dnsResp.Answer {
		answers = append(answers, a.RR())
	}

	// Parse google RRs to DNS RRs
	authorities := []dns.RR{}
	for _, ns := range dnsResp.Authority {
		authorities = append(authorities, ns.RR())
	}

	// Parse google RRs to DNS RRs
	extras := []dns.RR{}
	for _, extra := range dnsResp.Extra {
		authorities = append(authorities, extra.RR())
	}

	resp := dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:                 r.Id,
			Response:           (dnsResp.ResponseCode == 0),
			Opcode:             dns.OpcodeQuery,
			Authoritative:      false,
			Truncated:          dnsResp.Truncated,
			RecursionDesired:   dnsResp.RecursionDesired,
			RecursionAvailable: dnsResp.RecursionAvailable,
			AuthenticatedData:  dnsResp.AuthenticatedData,
			CheckingDisabled:   dnsResp.CheckingDisabled,
			Rcode:              dnsResp.ResponseCode,
		},
		Compress: r.Compress,
		Question: questions,
		Answer:   answers,
		Ns:       authorities,
		Extra:    extras,
	}

	// Write the response
	err = w.WriteMsg(&resp)
	if err != nil {
		log.Errorln("Error writing DNS response:", err)
	}
}
