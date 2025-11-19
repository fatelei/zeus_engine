# Makefile for Zeus Workflow Engine

BINARY_NAME=zeus
GO_FILES=$(shell find . -name '*.go')

.PHONY: all build run test lint clean help

all: build

## build: Build the project
build:
	go build -o $(BINARY_NAME) main.go

## run: Run the demo workflow
run:
	go run main.go

## test: Run all tests
test:
	go test -v ./...

## lint: Run golangci-lint
lint:
	golangci-lint run

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'
