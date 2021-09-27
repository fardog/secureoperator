FROM golang:1.14.4-alpine3.12 AS builder

RUN apk --no-cache add make
RUN mkdir -p /go/src/github.com/tinkernels/doh-proxy
COPY . /go/src/github.com/tinkernels/doh-proxy

WORKDIR /go/src/github.com/tinkernels/doh-proxy/v5/
RUN make linux-amd64

FROM alpine:3.12.0 as final
EXPOSE 53 53/udp
RUN apk --no-cache add ca-certificates libcap && update-ca-certificates

COPY --from=builder /go/src/github.com/tinkernels/doh-proxy/v5/release/doh-proxy_linux-amd64 /doh

#https://jessicadeen.com/how-to-solve-the-listen-tcp-80-bind-permission-denied-error-in-docker/
RUN setcap 'cap_net_bind_service=+ep' /doh

RUN addgroup -g 1000 doh
RUN adduser -D -u 1000 -G doh -g 'doh' doh

USER 1000:1000

ENTRYPOINT ["/doh"]
CMD ["-google", "-http2", "-endpoint", "https://dns.google/resolve", "-endpoint-ips", "8.8.8.8,8.8.4.4", "-edns-subnet", "auto", "-listen", "127.0.0.1:53", "-no-ipv6", "-cache=true", "-loglevel", "info"]

#HOW TO RUN:
#docker run -p 53:53 --name doh -d doh:latest
#https://github.com/pi-hole/docker-pi-hole#note-on-capabilities
#docker run -p 53:53 --name doh --cap-add=NET_ADMIN --cap-add=CAP_NET_BIND_SERVICE --cap-add=NET_RAW -d doh:latest