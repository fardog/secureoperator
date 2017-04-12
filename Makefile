package = github.com/fardog/secureoperator
cmd_package = $(package)/cmd/secure-operator

.PHONY: release test
.DEFAULT_GOAL := test

release:
	mkdir -p release
	GOOS=linux GOARCH=amd64 go build -o release/secure-operator_linux-amd64 $(cmd_package)
	GOOS=linux GOARCH=386 go build -o release/secure-operator_linux-386 $(cmd_package)
	GOOS=linux GOARCH=arm go build -o release/secure-operator_linux-arm $(cmd_package)
	GOOS=darwin GOARCH=amd64 go build -o release/secure-operator_macos-amd64 $(cmd_package)
	GOOS=darwin GOARCH=386 go build -o release/secure-operator_macos-386 $(cmd_package)
	GOOS=windows GOARCH=amd64 go build -o release/secure-operator_windows-amd64.exe $(cmd_package)
	GOOS=windows GOARCH=386 go build -o release/secure-operator_windows-386.exe $(cmd_package)

test:
	go test -v ./ ./cmd
