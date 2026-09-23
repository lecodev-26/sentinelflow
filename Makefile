.PHONY: help build build-gateway build-controlplane build-worker build-cli
.PHONY: run run-gateway run-controlplane run-worker
.PHONY: test test-cover test-race vet fmt fmt-check lint clean deps
.PHONY: docker-build docker-build-gateway docker-build-controlplane docker-build-worker
.PHONY: backup dr-test all ci

BIN_DIR := bin
GATEWAY := $(BIN_DIR)/gateway
CONTROLPLANE := $(BIN_DIR)/controlplane
WORKER := $(BIN_DIR)/worker
CLI := $(BIN_DIR)/sfctl

help:
	@echo "SentinelFlow V3 - Available targets:"
	@echo ""
	@echo "  Build:  make build | build-gateway | build-controlplane | build-worker | build-cli"
	@echo "  Run:    make run-gateway | run-controlplane | run-worker"
	@echo "  Test:   make test | test-cover | test-race"
	@echo "  QA:     make vet | fmt | fmt-check | lint"
	@echo "  Ops:    make backup | dr-test"
	@echo "  CI:     make ci"

build: build-gateway build-controlplane build-worker build-cli
	@echo "All V3 binaries built in $(BIN_DIR)/"

build-gateway:
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(GATEWAY) ./cmd/gateway

build-controlplane:
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(CONTROLPLANE) ./cmd/controlplane

build-worker:
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(WORKER) ./cmd/worker

build-cli:
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(CLI) ./cmd/cli

run-gateway:
	go run ./cmd/gateway

run-controlplane:
	go run ./cmd/controlplane

run-worker:
	go run ./cmd/worker

test:
	go test -v ./...

test-cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-race:
	go test -race -v ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@if [ -n "$$(gofmt -l .)" ]; then echo "Not formatted:"; gofmt -l .; exit 1; fi

lint:
	golangci-lint run ./...

deps:
	go mod tidy
	go mod download

clean:
	rm -rf $(BIN_DIR)
	go clean

docker-build-gateway:
	docker build --build-arg SERVICE=gateway -t sentinelflow-gateway:3.0.0 .

docker-build-controlplane:
	docker build --build-arg SERVICE=controlplane -t sentinelflow-controlplane:3.0.0 .

docker-build-worker:
	docker build --build-arg SERVICE=worker -t sentinelflow-worker:3.0.0 .

backup:
	./scripts/backup-supabase.sh

dr-test:
	./scripts/dr-test.sh

ci: fmt-check vet test build
	@echo "CI simulation passed"

all: ci

.DEFAULT_GOAL := help
