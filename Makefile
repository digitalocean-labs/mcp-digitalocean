all: lint build test

build:
	go build ./...


build-bin:
	go build -o bin/mcp-digitalocean ./cmd/mcp.go 

lint:
	revive ./...
	staticcheck ./...

test:
	go test -v ./...

format:
	gofmt -w .
	@echo "Code formatted successfully."

format-check:
	bash -c 'diff -u <(echo -n) <(gofmt -d ./)'

