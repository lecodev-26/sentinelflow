.PHONY: build run test clean deps vet fmt lint

BINARY=bin/proxy
MAIN=cmd/proxy/main.go

build:
	go build -o $(BINARY) $(MAIN)

run:
	go run $(MAIN)

test:
	go test -v ./...

test-cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	go clean

deps:
	go mod tidy
	go mod download

vet:
	go vet ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run ./...

all: fmt vet test build

help:
	@echo "Comandos disponibles:"
	@echo "  make build      - Compilar el proxy"
	@echo "  make run        - Ejecutar el proxy"
	@echo "  make test       - Ejecutar tests"
	@echo "  make test-cover - Tests con cobertura"
	@echo "  make clean      - Limpiar binarios"
	@echo "  make deps       - Instalar dependencias"
	@echo "  make vet        - Análisis estático"
	@echo "  make fmt        - Formatear código"
	@echo "  make lint       - Linter (golangci-lint)"
	@echo "  make all        - fmt + vet + test + build"
