# OpenAPI service (`openapi`)

The `openapi` service exposes the DigitalOcean public REST API through an embedded [OpenAPI 3](https://swagger.io/specification/) document (`spec/DigitalOcean-public.v2.yaml`). It complements the hand-written godo-based tools: agents discover operations from the spec, inspect schemas, then call the API with **kin-openapi** request validation before traffic is sent.

## Enabling the service

Use the service name `openapi` with `--services` or the `SERVICES` environment variable (comma-separated), same as other services:

```bash
npx @digitalocean/mcp --services openapi --digitalocean-api-token "$DIGITALOCEAN_API_TOKEN"
```

`openapi-search` and `openapi-get-operation` do not perform account mutations and do not require the token for their own logic, but the MCP server still expects a token for stdio mode when other services or `openapi-execute` / `openapi-execute-delete` are used.

## Tools

| Tool | Purpose |
|------|---------|
| `openapi-search` | Keyword search over `operationId`, summary, path, HTTP method, and tags. Optional `Tag` filters to operations that list that exact tag. **`Limit` is clamped between 1 and 50.** Annotated `readOnlyHint: true` (embedded spec only). |
| `openapi-get-operation` | Full text description for one `operationId`: parameters, request body outline, response codes. Annotated `readOnlyHint: true`. |
| `openapi-execute` | Non-**DELETE** operations only. Build a request from `Parameters` (path/query/header by name), optional JSON `Body`, run [openapi3filter.ValidateRequest](https://pkg.go.dev/github.com/getkin/kin-openapi/openapi3filter), then `godo.Client.Do`. Responses include HTTP status, selected headers (rate limits, pagination `Link`, etc.), then the body (JSON/text bodies are trimmed for display). Large bodies are truncated at **1 MiB** with a clear marker. Annotated `openWorldHint: true`; rejects DELETE (use `openapi-execute-delete`). |
| `openapi-execute-delete` | **DELETE** operations only. Same validation and execution path as `openapi-execute`. Annotated `destructiveHint: true` so MCP hosts can require explicit user approval per call. Not registered when deletes are disabled server-side (see below). |

### `openapi-execute` and `openapi-execute-delete` arguments

Both tools accept:

- **`OperationID`** (required): From the spec / search results.
- **`Parameters`** (optional): Map of parameter **name** → value. Values may be string, number, boolean, or an array for repeated query parameters. Path parameters must be present for templated paths such as `/v2/droplets/{droplet_id}`.
- **`Body`** (optional): JSON object serialized as **`application/json`** when the operation accepts that content type. Operations that only allow non-JSON bodies (e.g. multipart) are rejected with the MIME types listed in the OpenAPI spec.
- **`Select`** (optional): After a successful JSON response, dotted path to extract part of the decoded body (e.g. `droplets[*].id`). Ignored when the response is not `application/json` (a `select-not-applied` line is included in that case).

Destructive intent is signaled by **`destructiveHint: true`** on `openapi-execute-delete` so the host client can prompt the user before running the tool. That is advisory only if the host auto-approves tools.

### DELETE safety

1. **Host-side approval:** MCP clients are expected to treat `destructiveHint: true` as requiring explicit user confirmation before the tool runs.
2. **Server-side capability removal:** If the server was started with **`--openapi-disable-deletes`** or **`OPENAPI_DISABLE_DELETES=true`**, **`openapi-execute-delete` is not registered** — the delete capability is absent from `tools/list`, not merely rejected at call time.
3. **Method gating:** `openapi-execute` refuses **DELETE** operations (direct the agent to `openapi-execute-delete`). `openapi-execute-delete` refuses non-DELETE operations (use `openapi-execute`).

The only hard guarantee that deletes cannot run through this MCP surface is **`--openapi-disable-deletes`** (plus API token scopes outside this server).

## Server flags (main binary)

| Flag | Env | Effect |
|------|-----|--------|
| `--openapi-disable-deletes` | `OPENAPI_DISABLE_DELETES=true` | Skips registering `openapi-execute-delete` so the model never sees a delete capability. |

## Refreshing the embedded spec

The YAML is compiled into the binary with `//go:embed`. To download the current public spec from DigitalOcean and overwrite the committed file:

```bash
make update-openapi-spec
```

Then commit `pkg/registry/openapi/spec/DigitalOcean-public.v2.yaml` when you intentionally bump the API surface.

## Implementation notes

- Operations that declare **cookie parameters** are skipped when the embedded spec is loaded (they cannot be executed by these tools); a warning is logged once per skipped operation.
- Tools declare **`outputSchema`** via `mcp.WithOutputSchema` and return **`structuredContent`** plus human-readable fallback text (`NewToolResultStructured`) so clients that support structured results get typed payloads while others still receive plain text.
- Parsing and `$ref` resolution: [`github.com/getkin/kin-openapi/openapi3`](https://pkg.go.dev/github.com/getkin/kin-openapi/openapi3).
- Request validation: [`openapi3filter`](https://pkg.go.dev/github.com/getkin/kin-openapi/openapi3filter) with `AuthenticationFunc` set to noop (Bearer auth is enforced by godo, not duplicated in the validation request).
- Validation uses the spec's **first** `servers[].URL`; execution uses **`godo.Client`'s** configured base URL so custom `DIGITALOCEAN_API_ENDPOINT` continues to work.
