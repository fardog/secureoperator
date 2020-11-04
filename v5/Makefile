resolver_package = "cmd/resolver/main.go"
stub_package = "cmd/stub/main.go"
build_flags = -ldflags "-w -s"

.PHONY: release test
.DEFAULT_GOAL := test

release:
	mkdir -p release
	GOOS=linux GOARCH=amd64 go build  $(build_flags) -o release/doh-proxy_linux-amd64 $(resolver_package)
	GOOS=linux GOARCH=386 go build  $(build_flags) -o release/doh-proxy_linux-386 $(resolver_package)
	GOOS=linux GOARCH=arm go build  $(build_flags) -o release/doh-proxy_linux-arm $(resolver_package)
	GOOS=darwin GOARCH=amd64 go build $(build_flags) -o release/doh-proxy_macos-amd64 $(resolver_package)
	GOOS=windows GOARCH=amd64 go build $(build_flags) -o release/doh-proxy_windows-amd64.exe $(resolver_package)
	GOOS=windows GOARCH=386 go build $(build_flags) -o release/doh-proxy_windows-386.exe $(resolver_package)

	GOOS=linux GOARCH=amd64 go build  $(build_flags) -o release/doh-stub_linux-amd64 $(stub_package)
	GOOS=linux GOARCH=386 go build  $(build_flags) -o release/doh-stub_linux-386 $(stub_package)
	GOOS=linux GOARCH=arm go build  $(build_flags) -o release/doh-stub_linux-arm $(stub_package)
	GOOS=darwin GOARCH=amd64 go build $(build_flags) -o release/doh-stub_macos-amd64 $(stub_package)
	GOOS=windows GOARCH=amd64 go build $(build_flags) -o release/doh-stub_windows-amd64.exe $(stub_package)
	GOOS=windows GOARCH=386 go build $(build_flags) -o release/doh-stub_windows-386.exe $(stub_package)

test:
	go test -v ./
