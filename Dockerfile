FROM alpine:latest

EXPOSE 53

RUN apk --no-cache add ca-certificates && update-ca-certificates

# link shared libs in place for go binary
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

COPY release/secure-operator_linux-amd64 /usr/local/bin/secure-operator

CMD secure-operator --listen 0.0.0.0:53