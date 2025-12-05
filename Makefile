.PHONY: build run test lint clean

build:
	go build -o bin/lazylab ./cmd/lazylab

run: build
	./bin/lazylab

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/

.DEFAULT_GOAL := build
