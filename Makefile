all: clean lint test-race build-dist
build-dist: build-bin dist
build-dist-snapshot: build-bin-snapshot dist

build-bin-snapshot:
	goreleaser build --snapshot --clean --skip validate

build-bin:
	goreleaser build --auto-snapshot --clean --skip validate

.PHONY: dist
dist:
	mkdir -p ./scripts/npm/dist
	mkdir -p ./scripts/npm/apps/docs
	cp ./README.md ./scripts/npm/README.md
	cp ./dist/*/mcp-digitalocean* ./scripts/npm/dist/
	cp ./pkg/registry/apps/spec/*.json ./scripts/npm/dist/
	cp ./pkg/registry/doks/spec/*.json ./scripts/npm/dist/
	cp -r ./pkg/registry/apps/docs/* ./scripts/npm/apps/docs/
	npm install --prefix ./scripts/npm/

clean:
	rm -rf ./dist ./scripts/npm/dist ./scripts/npm/apps

lint:
	revive -config revive.toml ./...

test:
	go test -v ./...

test-race:
	go test -race -v ./...

test-e2e:
	go test -v -tags=integration,logging -timeout 10m ./...

format:
	gofmt -w .
	@echo "Code formatted successfully."

format-check:
	bash -c 'diff -u <(echo -n) <(gofmt -d ./)'

gen:
	go generate ./...
