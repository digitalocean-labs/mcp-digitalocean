all: lint build test
build-dist: build-bin dist

build:
	go build ./...

build-bin:
	GOOS=darwin GOARCH=arm64 go build -o bin/mcp-digitalocean-darwin-arm64 ./cmd/mcp-digitalocean/main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/mcp-digitalocean-darwin-amd64 ./cmd/mcp-digitalocean/main.go
	GOOS=linux GOARCH=arm64 go build -o bin/mcp-digitalocean-linux-arm64 ./cmd/mcp-digitalocean/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/mcp-digitalocean-linux-amd64 ./cmd/mcp-digitalocean/main.go
	GOOS=windows GOARCH=amd64 go build -o bin/mcp-digitalocean-windows-amd64.exe ./cmd/mcp-digitalocean/main.go
	GOOS=windows GOARCH=arm64 go build -o bin/mcp-digitalocean-windows-arm64.exe ./cmd/mcp-digitalocean/main.go

dist:
	cp ./bin/* ./scripts/npm/dist/
	cp ./internal/apps/spec/*.json ./scripts/npm/dist/
	npm install --prefix ./scripts/npm/

lint:
	revive -config revive.toml ./...

test:
	go test -v ./...

format:
	gofmt -w .
	@echo "Code formatted successfully."

format-check:
	bash -c 'diff -u <(echo -n) <(gofmt -d ./)'

gen:
	go generate ./...
