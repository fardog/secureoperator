package secureoperator

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	// DNSNameMaxBytes is the maximum number of bytes a DNS name may contain
	DNSNameMaxBytes = 253
	// max number of characters in a 16-bit uint integer, converted to string
	extraPad         = 5
	paddingParameter = "random_padding"
)

// GDNSQuestion represents a question response item from Google's DNS service
// This is currently the same as DNSQuestion, our internal implementation, but
// since Google's API is in flux, we keep them separate
type GDNSQuestion DNSQuestion

// DNSQuestion transforms a GDNSQuestion to a DNSQuestion and returns it.
func (r GDNSQuestion) DNSQuestion() DNSQuestion {
	return DNSQuestion{
		Name: r.Name,
		Type: r.Type,
	}
}

// GDNSQuestions is a array of GDNSQuestion objects
type GDNSQuestions []GDNSQuestion

// DNSQuestions transforms an array of GDNSQuestion objects to an array of
// DNSQuestion objects
func (rs GDNSQuestions) DNSQuestions() (rqs []DNSQuestion) {
	for _, r := range rs {
		rqs = append(rqs, r.DNSQuestion())
	}

	return
}

// GDNSRR represents a dns response record item from Google's DNS service.
// This is currently the same as DNSRR, our internal implementation, but since
// Google's API is in flux, we keep them separate
type GDNSRR DNSRR

// DNSRR transforms a GDNSRR to a DNSRR
func (r GDNSRR) DNSRR() DNSRR {
	return DNSRR{
		Name: r.Name,
		Type: r.Type,
		TTL:  r.TTL,
		Data: r.Data,
	}
}

// GDNSRRs represents an array of GDNSRR objects
type GDNSRRs []GDNSRR

// DNSRRs transforms an array of GDNSRR objects to an array of DNSRR objects
func (rs GDNSRRs) DNSRRs() (rrs []DNSRR) {
	for _, r := range rs {
		rrs = append(rrs, r.DNSRR())
	}

	return
}

// GDNSResponse represents a response from the Google DNS-over-HTTPS servers
type GDNSResponse struct {
	Status           int32         `json:"Status,omitempty"`
	TC               bool          `json:"TC,omitempty"`
	RD               bool          `json:"RD,omitempty"`
	RA               bool          `json:"RA,omitempty"`
	AD               bool          `json:"AD,omitempty"`
	CD               bool          `json:"CD,omitempty"`
	Question         GDNSQuestions `json:"Question,omitempty"`
	Answer           GDNSRRs       `json:"Answer,omitempty"`
	Authority        GDNSRRs       `json:"Authority,omitempty"`
	Additional       GDNSRRs       `json:"Additional,omitempty"`
	EDNSClientSubnet string        `json:"edns_client_subnet,omitempty"`
	Comment          string        `json:"Comment,omitempty"`
}

// GDNSProvider is the Google DNS-over-HTTPS provider; it implements the
// Provider interface.
type GDNSProvider struct {
	Endpoint string
	Pad      bool
}

// Query sends a DNS question to Google, and returns the response
func (g GDNSProvider) Query(q DNSQuestion) (*DNSResponse, error) {
	httpreq, err := http.NewRequest(http.MethodGet, g.Endpoint, nil)
	if err != nil {
		return nil, err
	}

	qry := httpreq.URL.Query()
	dnsType := fmt.Sprintf("%v", q.Type)

	l := len([]byte(q.Name))
	if l > DNSNameMaxBytes {
		return nil, fmt.Errorf("name length of %v exceeds DNS name max length", l)
	}

	qry.Add("name", q.Name)
	qry.Add("type", dnsType)
	qry.Add("edns_client_subnet", "0.0.0.0/0")

	httpreq.URL.RawQuery = qry.Encode()

	if g.Pad {
		// pad to the maximum size a valid request could be. we add `1` because
		// Google's DNS service ignores a trailing period, increasing the
		// possible size of a name by 1
		pad := randSeq(DNSNameMaxBytes + extraPad - l - len(dnsType) + 1)
		qry.Add(paddingParameter, pad)

		httpreq.URL.RawQuery = qry.Encode()
	}

	httpresp, err := http.DefaultClient.Do(httpreq)
	if err != nil {
		return nil, err
	}
	defer httpresp.Body.Close()

	dnsResp := new(GDNSResponse)
	decoder := json.NewDecoder(httpresp.Body)
	err = decoder.Decode(&dnsResp)
	if err != nil {
		return nil, err
	}

	return &DNSResponse{
		Question:           dnsResp.Question.DNSQuestions(),
		Answer:             dnsResp.Answer.DNSRRs(),
		Authority:          dnsResp.Authority.DNSRRs(),
		Extra:              dnsResp.Additional.DNSRRs(),
		Truncated:          dnsResp.TC,
		RecursionDesired:   dnsResp.RD,
		RecursionAvailable: dnsResp.RA,
		AuthenticatedData:  dnsResp.AD,
		CheckingDisabled:   dnsResp.CD,
		ResponseCode:       int(dnsResp.Status),
	}, nil
}
