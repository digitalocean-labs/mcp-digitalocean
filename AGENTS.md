# AGENTS.md

Guidance for coding agents working in this repository.

## What this is

The DigitalOcean MCP server. It exposes DigitalOcean API operations as MCP tools, built on [godo](https://github.com/digitalocean/godo) and [mcp-go](https://github.com/mark3labs/mcp-go). It runs locally over stdio (published to npm as `@digitalocean/mcp`) or as a remote HTTP server.

- User docs: https://docs.digitalocean.com/reference/mcp/
- Contributor guide: [CONTRIBUTING.md](./CONTRIBUTING.md); onboarding a service: [pkg/README.md](./pkg/README.md)

## Layout

- `cmd/mcp-digitalocean/`: server entry point and flags (`--services`, `--transport`, `--log-level`, ...).
- `pkg/registry/registry.go`: `supportedServices` and `Register`, which wires each service's tools into the server.
- `pkg/registry/<service>/`: one package per service (`apps`, `droplet`, `networking`, `doks`, ...). Each has a `README.md` listing its tools.
- `pkg/registry/common/`: shared helpers, including tool annotations (`annotations.go`) and structured output (`output.go`).
- `internal/`: middleware, OAuth metadata, logging, and test helpers.
- `testing/`: end-to-end tests behind the `integration` build tag.
- `scripts/npm/`: npm wrapper package.

## Commands

- `make test`: unit tests (`go test -v ./...`). No credentials needed.
- `make test-race`: unit tests with the race detector.
- `make lint`: `revive -config revive.toml ./...` (install with `go install github.com/mgechev/revive@latest`).
- `make format` / `make format-check`: gofmt.
- `make gen`: `go generate ./...`, which regenerates the gomock `mocks.go` files.
- `make build-dist-snapshot`: local build with goreleaser.

CI (`.github/workflows/ci.yml`) runs `make lint`, `make format-check`, `make test-race`, and `make build-dist-snapshot`.

## End-to-end tests

`make test-e2e` and `go test -tags=integration ./testing/...` call the real DigitalOcean API with `DIGITALOCEAN_API_TOKEN` and can create billable resources. Do not run them unless the user explicitly asks. Never print, log, or commit a token. Unit tests mock godo and need no token.

## Adding or changing tools

- Tools live in `<resource>_tools.go` in the service package. A constructor takes the client factory (`func(ctx context.Context) (*godo.Client, error)`) and a `Tools()` method returns `[]server.ServerTool`.
- Tool names are lowercase and hyphenated, `<service>-<action>` (for example `apps-list`, `spaces-key-create`, `domain-record-get`). Names must be unique across all services.
- Tool arguments use UpperCamelCase (`AppID`, `Page`, `PerPage`).
- Every tool needs `common.WithHints(...)` with one of the profiles (`HintsRead`, `HintsCreate`, `HintsAction`, `HintsToggle`, `HintsDelete`, `HintsReplace`) and `common.WithRisk(...)`. `pkg/registry/annotations_guard_test.go` fails otherwise.
- Tests go in `<resource>_tools_test.go` using the generated godo mocks. Add new godo services to the `mockgen` line in the package's `generate.go` and run `make gen`.
- A new service needs an entry in `supportedServices`, a `register<Service>Tools` function and `case` in `Register`, a package `README.md`, and a link in the root `README.md` Documentation section.
- Update the service `README.md` when tools are added, renamed, or removed.
