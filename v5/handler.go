package dohProxy

import (
	"fmt"
	"github.com/miekg/dns"
	"github.com/panjf2000/ants/v2"
	"time"
)

const (
	queryExpireDuration = time.Second * 10
	concurrentPoolSize  = 128
)

var (
	isSerialMode     bool
	serialTaskNotify chan bool
)

// HandlerOptions specifies options to be used when instantiating a handler
type HandlerOptions struct {
	Cache  bool
	NoAAAA bool
}

// Handler represents a DNS handler
type Handler struct {
	options           *HandlerOptions
	provider          Provider
	hostsFileProvider Provider
	cache             *Cache
	pool              *ants.PoolWithFunc
}

type ctxParamsPoolFunc struct {
	req  *dns.Msg
	resp chan *dns.Msg
	err  error
}

type writerCtx struct {
	msg           *dns.Msg
	isAnsweredCh  chan bool
	isCache       bool
	edns0SubnetIn dns.EDNS0_SUBNET
	receivedTime  time.Time
}

// NewHandler creates a new Handler
func NewHandler(provider Provider, options *HandlerOptions) *Handler {
	handler := &Handler{
		options:           options,
		provider:          provider,
		hostsFileProvider: NewHostsFileProvider(),
	}
	p, _ := ants.NewPoolWithFunc(concurrentPoolSize, func(payload interface{}) {
		ctx, ok := payload.(*ctxParamsPoolFunc)
		if !ok {
			ctx.err = fmt.Errorf("cast pool func context failed")
			return
		}
		resp, err := handler.provider.Query(ctx.req)
		ctx.err = err
		ctx.resp <- resp
	},
		ants.WithLogger(Log))
	handler.pool = p
	if options.Cache {
		handler.cache = NewCache()
	}
	handler.initSerialMode()
	return handler
}

// Handle handles a DNS request
func (h *Handler) Handle(writer dns.ResponseWriter, msg *dns.Msg) {

	Log.Infoln("requesting", msg.Question[0].Name, dns.TypeToString[msg.Question[0].Qtype])

	isAnsweredCh := make(chan bool)
	defer close(isAnsweredCh)

	edns0SubnetIn := ObtainEDN0Subnet(msg)
	ctx := &writerCtx{msg: msg, isCache: false, isAnsweredCh: isAnsweredCh,
		edns0SubnetIn: edns0SubnetIn, receivedTime: time.Now()}
	if h.options.Cache {
		rmsg := h.cache.Get(msg)
		if rmsg != nil {
			rmsg.Id = msg.Id
			ctx.msg = rmsg
			ctx.isCache = true
			go h.TryWriteAnswer(&writer, ctx)
			if <-isAnsweredCh {
				Log.Infof("resolved from cache: %v, cost time: %v",
					msg.Question[0].Name, time.Now().Sub(ctx.receivedTime))
				return
			}
		}
	}

	// lookup hosts file if retrieving ip address
	if msg.Question[0].Qtype == dns.TypeA || msg.Question[0].Qtype == dns.TypeAAAA {
		go h.AnswerByHostsFile(&writer, ctx)
		if <-isAnsweredCh {
			Log.Infof("resolved from hosts: %v, cost time: %v",
				msg.Question[0].Name, time.Now().Sub(ctx.receivedTime))
			return
		}
	}

	if isSerialMode && serialTaskNotify != nil {
		select {
		case <-serialTaskNotify:
			break
		case <-time.After(queryExpireDuration):
			Log.Errorf("timeout for waiting serial task channel.")
			dns.HandleFailed(writer, msg)
			return
		}
	}

	go h.AnswerByDoH(&writer, ctx)
	if <-isAnsweredCh {
		Log.Infof("resolved from DoH: %v, cost time: %v",
			msg.Question[0].Name, time.Now().Sub(ctx.receivedTime))
		return
	}

	if !isSerialMode {
		h.initSerialMode()
	}
	dns.HandleFailed(writer, msg)
}

func (h *Handler) initSerialMode() {
	isSerialMode = true
	if serialTaskNotify == nil {
		serialTaskNotify = make(chan bool)
	}
	go func() { serialTaskNotify <- true }()
	Log.Infof("enter serial mode.")
}

func (h *Handler) TryWriteAnswer(writer *dns.ResponseWriter, ctx *writerCtx) {
	if ctx.msg != nil {
		ReplaceEDNS0Subnet(ctx.msg, &ctx.edns0SubnetIn)
		if h.options.Cache && !ctx.isCache {
			msgch := make(chan *dns.Msg)
			defer close(msgch)
			go h.cache.Insert(msgch)
			msgch <- ctx.msg
		}
		// Write the response
		writerReal := *writer
		err := writerReal.WriteMsg(ctx.msg)
		if err != nil {
			Log.Errorf("Error writing DNS response: %v", err)
			ctx.isAnsweredCh <- false
		} else {
			ctx.isAnsweredCh <- true
			if isSerialMode && !ctx.isCache &&
				!(h.options.NoAAAA && ctx.msg.Question[0].Qtype == dns.TypeAAAA) {
				isSerialMode = false
				serialTaskNotify = nil
				Log.Infof("leave serial mode.")
			}
			Log.Debugf("Successfully write response message")
		}
	} else {
		ctx.isAnsweredCh <- false
	}
}

func (h *Handler) AnswerByHostsFile(writer *dns.ResponseWriter, ctx *writerCtx) {

	msgR, err := h.hostsFileProvider.Query(ctx.msg)
	if err != nil {
		Log.Debugf("hosts file provider failed: %v", err)
		ctx.isAnsweredCh <- false
		return
	}
	ctx.msg = msgR
	ctx.isCache = false
	go h.TryWriteAnswer(writer, ctx)
}

func (h *Handler) AnswerByDoH(writer *dns.ResponseWriter, ctx *writerCtx) {
	ctxP := &ctxParamsPoolFunc{req: ctx.msg, resp: make(chan *dns.Msg)}
	if err := h.pool.Invoke(ctxP); err != nil {
		Log.Errorf("dns-message provider failed: %v", err)
		ctx.isAnsweredCh <- false
		return
	}

	if ctxP.err != nil {
		Log.Errorf("query failed: %v", ctxP.err)
		ctx.isAnsweredCh <- false
		return
	}
	ctx.msg = <- ctxP.resp
	ctx.isCache = false
	go h.TryWriteAnswer(writer, ctx)
	if isSerialMode && serialTaskNotify != nil {
		go func(){ serialTaskNotify <- true}()
	}
}
