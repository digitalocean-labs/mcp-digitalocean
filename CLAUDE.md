# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MCP DigitalOcean Integration — a Go-based Model Context Protocol (MCP) server that exposes DigitalOcean cloud infrastructure management as MCP tools. Built on [mcp-go](https://github.com/mark3labs/mcp-go) framework and the official [godo](https://github.com/digitalocean/godo) DigitalOcean client.

## Build & Test Commands

```bash
make all              # clean, lint, test-race, build-dist
make test             # go test -v ./...
make test-race        # go test -race -v ./... (CI uses this)
make lint             # revive -config revive.toml ./...
make format           # gofmt -w .
make format-check     # verify formatting without modifying
make gen              # go generate ./...
make build-dist       # goreleaser build + npm dist packaging
make test-e2e         # E2E tests (requires DIGITALOCEAN_API_TOKEN)
```

Run a single test:
```bash
go test -v -run TestReadOnlyMiddleware_BlocksMutatingTool ./internal/...
```

## Architecture

### Entry Point & Transport

`cmd/mcp-digitalocean/main.go` — Parses flags/env vars, sets up middleware chain, registers services, and runs the MCP server. Two transport modes:
- **stdio** (default): Single reused godo client, token from `--digitalocean-api-token` flag
- **http**: New godo client per request, token extracted from HTTP `Authorization` header via `internal/middleware.go`

### Service Registry Pattern

`pkg/registry/registry.go` dispatches to service-specific registration functions. Each service package (under `pkg/registry/`) exposes tool structs with a `Tools()` method returning `[]server.Tool`. Services are loaded selectively via `--services apps,droplets` or all if unspecified. Common tools (regions) always load.

### Tool Structure Convention

Each tool struct takes a client factory `func(ctx context.Context) (*godo.Client, error)` for dependency injection. Tool names use `service-action` kebab-case (e.g., `apps-list`, `droplet-create`). Tool arguments use UpperCamelCase (e.g., `AppID`, `PerPage`).

### Middleware Chain (`internal/`)

- `ToolLoggingMiddleware` — Logs tool call duration and outcome
- `ReadOnlyMiddleware` — Blocks mutating tools by checking action tokens in tool names (create, delete, update, etc.)
- `AuthFromRequest` — Extracts auth from HTTP headers into context

### Services (under `pkg/registry/`)

apps, droplet, networking, dbaas, doks, account, spaces, marketplace, insights, common

### Testing

- **Unit tests**: Alongside implementations, using `go.uber.org/mock` for godo client mocking
- **E2E tests**: In `testing/` directory, require real API token and use testcontainers
- Test dependency: `github.com/stretchr/testify` for assertions

### Key Config Flags

`--services`, `--digitalocean-api-token`, `--transport` (stdio|http), `--bind-addr`, `--read-only`, `--log-level`, `--ws-logging-url`, `--enable-tool-error-logging`

## Code Style

- Go 1.25.3, formatted with `gofmt`/`goimports`
- Linted with `revive` (config: `revive.toml`)
- Version constant in `cmd/mcp-digitalocean/main.go` (currently `1.0.31`)
