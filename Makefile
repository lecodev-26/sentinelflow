.PHONY: build run test clean

build:
	go build -o bin/proxy cmd/proxy/main.go

run:
	go run cmd/proxy/main.go

test:
	go test -v ./...

clean:
	rm -rf bin/
	go clean

deps:
	go mod tidy
	go mod download
