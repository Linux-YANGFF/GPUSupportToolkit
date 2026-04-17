.PHONY: dev build test lint clean

dev:
	go run ./cmd/gst

build:
	go build -o bin/gst ./cmd/gst

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
