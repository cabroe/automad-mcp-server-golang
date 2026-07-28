# Makefile for automad-mcp-server

BINARY := automad-mcp-server
PKG    := github.com/cabroe/automad-mcp-server

.PHONY: all build run test lint clean tidy

all: build

## build: Compile the binary
build:
	go build -o $(BINARY) .

## run: Run the server directly (stdio mode)
run:
	go run .

## test: Run all tests
test:
	go test ./... -v -count=1

## lint: Run go vet
lint:
	go vet ./...

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
