BIN := markpost
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build test test-race cover lint fmt acceptance clean

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BIN) ./cmd/markpost

test:
	go test ./...

test-race:
	go test -race ./...

cover:
	go test -cover ./...

lint:
	golangci-lint run ./...

fmt:
	golangci-lint fmt

# End-to-end against a live server (see acceptance/acceptance_test.go for the
# required MARKPOST_E2E_* variables). Skipped, not failed, without them.
acceptance:
	go test -tags acceptance ./acceptance

clean:
	rm -f $(BIN)
