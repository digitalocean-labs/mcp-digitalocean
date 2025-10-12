# MCP DigitalOcean Makefile

.PHONY: all build-dist build-dist-snapshot build-bin build-bin-snapshot dist lint test format format-check gen clean help install-tools benchmark coverage

# Default target
all: lint test build-dist

# Build targets
build-dist: build-bin dist
build-dist-snapshot: build-bin-snapshot dist

build-bin-snapshot:
	goreleaser build --snapshot --clean --skip validate

build-bin:
	goreleaser build --auto-snapshot --clean --skip validate

# Distribution
dist:
	mkdir -p ./scripts/npm/dist
	cp ./README.md ./scripts/npm/README.md
	cp ./CHANGELOG.md ./scripts/npm/CHANGELOG.md
	cp ./dist/*/mcp-digitalocean* ./scripts/npm/dist/
	cp ./internal/apps/spec/*.json ./scripts/npm/dist/
	cp ./internal/doks/spec/*.json ./scripts/npm/dist/
	npm install --prefix ./scripts/npm/

# Code quality
lint:
	revive -config revive.toml ./...
	@echo "✅ Linting completed successfully"

test:
	go test -v ./...
	@echo "✅ Tests completed successfully"

# Enhanced test targets
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

test-race:
	go test -v -race ./...
	@echo "✅ Race condition tests completed"

benchmark:
	go test -v -bench=. -benchmem ./...
	@echo "✅ Benchmarks completed"

# Code formatting
format:
	gofmt -w .
	@echo "✅ Code formatted successfully"

format-check:
	@bash -c 'diff -u <(echo -n) <(gofmt -d ./)'
	@echo "✅ Code formatting check passed"

# Code generation
gen:
	go generate ./...
	@echo "✅ Code generation completed"

# Development tools
install-tools:
	go install github.com/mgechev/revive@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/goreleaser/goreleaser@latest
	@echo "✅ Development tools installed"

# Security scanning
security-scan:
	@command -v gosec >/dev/null 2>&1 || { echo "Installing gosec..."; go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; }
	gosec ./...
	@echo "✅ Security scan completed"

# Dependency management
deps-update:
	go get -u ./...
	go mod tidy
	@echo "✅ Dependencies updated"

deps-verify:
	go mod verify
	go mod tidy
	@echo "✅ Dependencies verified"

# Clean up
clean:
	rm -rf dist/
	rm -rf ./scripts/npm/dist/
	rm -f coverage.out coverage.html
	go clean -cache -testcache -modcache
	@echo "✅ Cleanup completed"

# Docker targets
docker-build:
	docker build -t mcp-digitalocean:latest .
	@echo "✅ Docker image built"

docker-test:
	docker run --rm -e DIGITALOCEAN_API_TOKEN=test-token mcp-digitalocean:latest --services apps --log-level debug
	@echo "✅ Docker test completed"

# CI/CD helpers
ci-setup: install-tools deps-verify
	@echo "✅ CI environment setup completed"

ci-test: format-check lint test-race test-coverage security-scan
	@echo "✅ CI tests completed"

# Development workflow
dev-setup: install-tools deps-verify gen
	@echo "✅ Development environment setup completed"

dev-test: format lint test
	@echo "✅ Development tests completed"

# Release preparation
pre-release: clean gen format lint test-coverage security-scan build-dist
	@echo "✅ Pre-release checks completed"

# Help target
help:
	@echo "Available targets:"
	@echo "  all              - Run lint, test, and build-dist"
	@echo "  build-dist       - Build distribution packages"
	@echo "  build-bin        - Build binary only"
	@echo "  test             - Run all tests"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  test-race        - Run tests with race detection"
	@echo "  benchmark        - Run benchmarks"
	@echo "  lint             - Run linting"
	@echo "  format           - Format code"
	@echo "  format-check     - Check code formatting"
	@echo "  gen              - Generate code"
	@echo "  security-scan    - Run security analysis"
	@echo "  deps-update      - Update dependencies"
	@echo "  deps-verify      - Verify dependencies"
	@echo "  clean            - Clean build artifacts"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-test      - Test Docker image"
	@echo "  install-tools    - Install development tools"
	@echo "  dev-setup        - Setup development environment"
	@echo "  dev-test         - Run development tests"
	@echo "  ci-setup         - Setup CI environment"
	@echo "  ci-test          - Run CI tests"
	@echo "  pre-release      - Run pre-release checks"
	@echo "  help             - Show this help message"
