# Sandbox

Tools for the DigitalOcean Sandboxes API (`/v2/sandboxes`). Covers sandbox lifecycle, snapshots, templates and template builds, one-shot command execution, long-running commands, and file transfer.

API source: [DigitalOcean-public.v2.yaml (feat/sandbox-apis)](https://raw.githubusercontent.com/digitalocean/openapi/refs/heads/feat/sandbox-apis/specification/DigitalOcean-public.v2.yaml)

Enable with `--services sandbox`.

## Scopes

| Area      | Scopes                                                                 |
|-----------|------------------------------------------------------------------------|
| Sandboxes | `sandbox:read`, `sandbox:create`, `sandbox:update`, `sandbox:delete`   |
| Templates | `sandbox_templates:read`, `sandbox_templates:create`, `sandbox_templates:update`, `sandbox_templates:delete` |

## Tools

### Sandboxes

| Tool | Description |
|------|-------------|
| `sandbox-list` | List sandboxes, optionally filtered by `State`. Paginated with `PageSize` and `PageToken`. |
| `sandbox-create` | Create a sandbox from a `Template` and `SizeSlug`. |
| `sandbox-get` | Get a sandbox by `SandboxID`. |
| `sandbox-delete` | Delete a sandbox. Destructive. |
| `sandbox-pause` | Pause a sandbox. |
| `sandbox-resume` | Resume a paused sandbox. |

### Snapshots

| Tool | Description |
|------|-------------|
| `sandbox-snapshot-create` | Snapshot a sandbox. |
| `sandbox-snapshot-list` | List snapshots, optionally for one `SandboxID`. |
| `sandbox-snapshot-get` | Get a snapshot by `SnapshotID`. |
| `sandbox-snapshot-delete` | Delete a snapshot by `SnapshotID`. |

### Templates

| Tool | Description |
|------|-------------|
| `sandbox-template-list` | List templates. |
| `sandbox-template-get` | Get a template by `TemplateID`. |
| `sandbox-template-create` | Create a template from `Name`, `BaseTemplate` and `SourceOCIRef`. |
| `sandbox-template-update` | Update a template. Needs at least one of `BaseTemplate` or `SourceOCIRef`. |
| `sandbox-template-delete` | Delete a template. Destructive. |
| `sandbox-template-build-list` | List builds for a template. |
| `sandbox-template-build-get` | Get one build by `TemplateID` and `BuildID`. |

### Commands

| Tool | Description |
|------|-------------|
| `sandbox-command-list` | List long-running commands in a sandbox. |
| `sandbox-command-start` | Start a long-running command (`Cmd`, optional `Args`, `Cwd`, `EnvVars`, `Stdin`, `TimeoutSeconds`). Returns its PID. |
| `sandbox-command-signal` | Send `SIGTERM` or `SIGKILL` to a command PID. |
| `sandbox-command-stdin` | Write base64 `Data` to a command's stdin, or close it with `EOF`. |
| `sandbox-exec-start` | Run a one-shot command (`Argv`) and wait for it to exit. Returns exit code, stdout and stderr. |

### Files

| Tool | Description |
|------|-------------|
| `sandbox-file-list` | List entries under `Path`, optionally to `Depth`. |
| `sandbox-file-info` | Get metadata for one `Path`. |
| `sandbox-file-exists` | Check whether `Path` exists. |
| `sandbox-file-download` | Read a file. Returns UTF-8 text, or base64 for binary content. Verifies the SHA-256 footer. |
| `sandbox-file-upload` | Write a file from `Content` (text) or `ContentBase64`. Sends a SHA-256 checksum. |
| `sandbox-file-make-directory` | Create a directory. |
| `sandbox-file-move` | Move or rename `Source` to `Destination`. |
| `sandbox-file-remove` | Remove a file or directory. Destructive. |

## Limits and behavior

- **File transfer is inline.** Upload and download payloads are capped at 1 MiB (`maxInlineFileBytes`). Larger files fail with a tool error before any bytes are sent. Binary content goes through as base64.
- **Download integrity.** The API appends a footer (`DOWSSHA1` + SHA-256 hex + newline). The tool strips it and checks the hash. A missing or mismatched footer is a tool error.
- **Upload integrity.** The tool computes a SHA-256 of the body and sends it in `X-Content-Sha256`. The API returns 503 when too many transfers are in flight; that comes back as a tool error.
- **Exec output is capped.** Stdout and stderr are each capped at 1 MiB. Truncation is flagged in `stdout_truncated` and `stderr_truncated`.
- **Exec stream parsing.** `sandbox-exec-start` accepts the event stream as SSE (`data:` lines) or NDJSON. A stream that ends without an exit event, or that carries an `error` envelope, is a tool error.
- **int64 fields.** The API sends some int64 values as JSON strings (for example `size`, `size_bytes`, `bytes_written`, `duration_ms`). The output schema declares them as integers, and the tests check that real responses match the schema.

## Not included

These operations are in the API but are not exposed as tools:

- **Streaming command attach.** It needs a long-lived stream, which the MCP request/response model does not provide.
- **Exec resize and stdin.** These need an open PTY session for the life of the process.

The `harness-runtime` service also has sandbox-adjacent tools, such as `harness-runtime-list-sandbox-sizes` and `harness-runtime-list-templates`. They use a different API. Use `sandbox-*` tools for the public Sandboxes API.

## Testing

```sh
go test ./pkg/registry/sandbox/
```

The tests run against an in-process HTTP server and need no token. `structured_output_test.go` runs each output-bearing tool through a real `MCPServer` with output-schema validation on. This package has no live end-to-end tests yet. Adding them needs a DigitalOcean token and creates billable sandboxes.
