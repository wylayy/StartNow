.PHONY: build run fmt vet tidy

build:
	go build -o bin/startnow ./cmd/startnow

run:
	go run ./cmd/startnow

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy
