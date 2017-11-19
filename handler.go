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
		Type: r.Question[0].Qtype,
	}
	log.Infoln("requesting", q.Name, dns.TypeToString[q.Type])

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
			Qtype:  c.Type,
			Qclass: r.Question[idx].Qclass,
		})
	}

	// Parse google RRs to DNS RRs
	answers := transformRR(dnsResp.Answer, "answer")
	authorities := transformRR(dnsResp.Authority, "authority")
	extras := transformRR(dnsResp.Extra, "extra")

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

// for a given []DNSRR, transform to dns.RR, logging if any errors occur
func transformRR(rrs []DNSRR, logType string) []dns.RR {
	var t []dns.RR

	for _, r := range rrs {
		if rr, err := r.DNSRR(); err != nil {
			log.Errorln("unable to translate record rr", logType, r, err)
		} else {
			t = append(t, rr)
		}
	}

	return t
}
