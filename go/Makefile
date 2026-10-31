.PHONY: build test test-race run clean

build:
	go build -o bin/aia ./cmd/aia

test:
	go test ./...

test-race:
	go test -race ./...

run:
	go run ./cmd/aia

clean:
	rm -rf bin
