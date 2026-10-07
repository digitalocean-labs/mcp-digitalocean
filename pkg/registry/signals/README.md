# Signals (`signals`)

## What this service does

This package registers **Signals** tools for the DigitalOcean MCP server: investigation GETs on `api.digitalocean.com/v1/signals`, consent list/get/set on `/v1/consent`, and export create/list/get/download on `/v1/signals/exports`. Generate-report is omitted until it is an explicit product ask.

**Enable these tools** with service name **`signals`**. Hosted URL: `https://signals.mcp.digitalocean.com/mcp`.

**Code layout**

| Path | Purpose |
| --- | --- |
| [`signals_tools.go`](signals_tools.go) | Tool handlers and MCP tool definitions |
| [`signals_tools_test.go`](signals_tools_test.go) | Unit tests (mocked API client) |
| [`outputs.go`](outputs.go) | Structured output schemas |
| [`generate.go`](generate.go) | mockgen directive |
| [`mocks.go`](mocks.go) | Generated mock for `godo.SignalsService` |

---

## Tools exposed

| Tool | What it does |
| --- | --- |
| `signals-list-consents` | List collection consent records for the team. |
| `signals-get-agent-consent` | GET collection consent for one agent. Missing row is default deny. |
| `signals-set-agent-consent` | PUT enable or disable collection for one agent. |
| `signals-list-agent-sessions` | List sessions for an agent. |
| `signals-list-session-segments` | List segments in a session. |
| `signals-list-session-dialogues` | List dialogue turns for a session. |
| `signals-get-segment` | Get one segment plus nested signals. |
| `signals-get-signal-report` | Get a persisted post-session analysis report if present. |
| `signals-create-export` | Create an export job for an agent. |
| `signals-list-exports` | List prior export jobs. |
| `signals-get-export` | Poll one export job. |
| `signals-get-export-download` | 15-minute presigned Spaces URL for a completed export. |
| `signals-get-export-options` | Catalog of exportable entity types. |

---

## Auth

Requires a DigitalOcean API token with `signals:query` (GETs) and `signals:update` (writes such as consent PUT and create export). The public API is gated by Flipper `fi_signals_apis`. Ingest may take up to 15 minutes to honor a consent change.
