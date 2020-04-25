FROM golang:alpine

EXPOSE 53

RUN apk --no-cache add ca-certificates && update-ca-certificates

RUN mkdir -p /go/src/github.com/tinkernels/doh-proxy
COPY . /go/src/github.com/tinkernels/doh-proxy

WORKDIR /go/src/github.com/tinkernels/doh-proxy/cmd/secure-operator
RUN go install -v

ENTRYPOINT ["secure-operator", "--listen", "0.0.0.0:53"]
