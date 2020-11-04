resolver_package = "cmd/resolver/main.go"
stub_package = "cmd/stub/main.go"
build_flags = -ldflags "-extldflags=-static -w -s"

CGO_ENABLED ?= 0
ifeq ($(RACE),1)
	build_flags += -race -a
    CGO_ENABLED=1
endif

.PHONY: release test linux-amd64 linux-386 linux-arm macos-amd64 windows-amd64 windows-386
.DEFAULT_GOAL := test

release: linux-amd64 linux-386 linux-arm macos-amd64 windows-amd64 windows-386

linux-amd64:
	mkdir -p release
	GOOS=linux GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-proxy_linux-amd64 $(resolver_package)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-stub_linux-amd64 $(stub_package)
linux-386:
	mkdir -p release
	GOOS=linux GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-proxy_linux-386 $(resolver_package)
  	GOOS=linux GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-stub_linux-386 $(stub_package)
linux-arm:
	mkdir -p release
	GOOS=linux GOARCH=arm CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-proxy_linux-arm $(resolver_package)
  	GOOS=linux GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-stub_linux-arm $(stub_package)
macos-amd64:
	mkdir -p release
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build $(build_flags) -o release/doh-proxy_macos-amd64 $(resolver_package)
  	GOOS=linux GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-stub_macos-amd64 $(stub_package)
windows-amd64:
	mkdir -p release
	GOOS=windows GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build $(build_flags) -o release/doh-proxy_windows-amd64.exe $(resolver_package)
  	GOOS=linux GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-stub_windows-amd64.exe $(stub_package)
windows-386:
	mkdir -p release
	GOOS=windows GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build $(build_flags) -o release/doh-proxy_windows-386.exe $(resolver_package)
  	GOOS=linux GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-stub_windows-386.exe $(stub_package)

test:
	go test -v ./
