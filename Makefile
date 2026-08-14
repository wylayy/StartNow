.PHONY: build run fmt vet test tidy

build:
	go build -o bin/startnow ./cmd/startnow

run:
	go run ./cmd/startnow

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

tidy:
	go mod tidy
