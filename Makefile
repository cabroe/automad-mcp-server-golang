# Makefile for automad-mcp-server

BINARY  := automad-mcp-server
PKG     := github.com/cabroe/automad-mcp-server
VERSION ?= dev
LDFLAGS := -X main.version=$(VERSION)

.PHONY: all build run test race lint fmt check clean tidy

all: build

## build: Compile the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## run: Run the server directly (stdio mode)
run:
	go run .

## test: Run all tests
test:
	go test ./... -v -count=1

## race: Run all tests with the race detector
race:
	go test -race ./... -count=1

## lint: Run go vet
lint:
	go vet ./...

## fmt: Format all Go source files
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

## check: Run formatting, vet, tests, race detector, and build
check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))" || (echo "Go files need formatting"; gofmt -l $$(find . -name '*.go' -not -path './vendor/*'); exit 1)
	go vet ./...
	go test ./... -count=1
	go test -race ./... -count=1
	go build -ldflags "$(LDFLAGS)" ./...

## tidy: Tidy and verify go.mod
tidy:
	go mod tidy
	go mod verify

## clean: Remove built binary
clean:
	rm -f $(BINARY)

## help: Show this help
help:
	@echo "Available targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'
