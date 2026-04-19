BINARY_NAME = mihomo-cli-tui
VERSION ?= 1.0.0

.PHONY: build build-all clean

build:
	go build -ldflags "-s -w" -o $(BINARY_NAME) main.go

build-all: build-linux build-darwin build-windows

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dist/$(BINARY_NAME)-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o dist/$(BINARY_NAME)-linux-arm64 main.go

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o dist/$(BINARY_NAME)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o dist/$(BINARY_NAME)-darwin-arm64 main.go

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o dist/$(BINARY_NAME)-windows-amd64.exe main.go

clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/
