.PHONY: help run build test vet fmt fmt-check clean

.DEFAULT_GOAL := help

SERVER_BIN := bin/server

## help: show available commands
help:
	@awk 'BEGIN { printf "\nUsage:\n  make <target>\n\n" } /^## [a-zA-Z0-9_.-]+:/ { line=$$0; sub(/^## /, "", line); sep=index(line, ":"); printf "  make %-18s %s\n", substr(line, 1, sep-1), substr(line, sep+2) }' $(MAKEFILE_LIST)

## run: run the API server
run:
	go run ./cmd/server

## build: build the API server
build:
	@mkdir -p bin
	go build -o $(SERVER_BIN) ./cmd/server

## test: run tests
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## fmt: format Go code
fmt:
	go fmt ./...

## fmt-check: check Go formatting
fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

## clean: remove build output
clean:
	rm -rf bin
