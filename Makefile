package = github.com/tinkernels/doh-proxy

build_flags = -ldflags "-w -s"

.PHONY: release test
.DEFAULT_GOAL := test

release:
	mkdir -p release
	GOOS=linux GOARCH=amd64 go build $(build_flags) -o release/doh-proxy_linux-amd64 $(package)
	GOOS=linux GOARCH=386 go build $(build_flags) -o release/doh-proxy_linux-386 $(package)
	GOOS=linux GOARCH=arm go build $(build_flags) -o release/doh-proxy_linux-arm $(package)
	GOOS=darwin GOARCH=amd64 go build $(build_flags) -o release/doh-proxy_macos-amd64 $(package)
	GOOS=windows GOARCH=amd64 go build $(build_flags) -o release/doh-proxy_windows-amd64.exe $(package)
	GOOS=windows GOARCH=386 go build $(build_flags) -o release/doh-proxy_windows-386.exe $(package)

test:
	go test -v ./
