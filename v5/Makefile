main_package = "cmd/main.go"
build_flags = -ldflags "-extldflags=-static -w -s"

CGO_ENABLED ?= 0
ifeq ($(RACE),1)
	build_flags += -race -a
    CGO_ENABLED=1
endif

.PHONY: release test
.DEFAULT_GOAL := test

release:
	mkdir -p release
	GOOS=linux GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-proxy_linux-amd64 $(main_package)
	GOOS=linux GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-proxy_linux-386 $(main_package)
	GOOS=linux GOARCH=arm CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-proxy_linux-arm $(main_package)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build $(build_flags) -o release/doh-proxy_macos-amd64 $(main_package)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build $(build_flags) -o release/doh-proxy_windows-amd64.exe $(main_package)
	GOOS=windows GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build $(build_flags) -o release/doh-proxy_windows-386.exe $(main_package)

linux-amd64:
	mkdir -p release
	GOOS=linux GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-proxy_linux-amd64 $(main_package)
linux-386:
	mkdir -p release
	GOOS=linux GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-proxy_linux-386 $(main_package)
linux-arm:
	mkdir -p release
	GOOS=linux GOARCH=arm CGO_ENABLED=${CGO_ENABLED} go build  $(build_flags) -o release/doh-proxy_linux-arm $(main_package)
macos-amd64:
	mkdir -p release
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build $(build_flags) -o release/doh-proxy_macos-amd64 $(main_package)
windows-amd64:
	mkdir -p release
	GOOS=windows GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build $(build_flags) -o release/doh-proxy_windows-amd64.exe $(main_package)
windows-386:
	mkdir -p release
	GOOS=windows GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build $(build_flags) -o release/doh-proxy_windows-386.exe $(main_package)

test:
	go test -v ./
