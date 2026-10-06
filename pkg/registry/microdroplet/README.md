# MicroVM MCP tools

Lifecycle tools for DigitalOcean MicroVMs over `POST/GET/DELETE /v2/microvms/*`. REST JSON is passed through; argument names mirror the public API (**snake_case**).

Activate with `--services microdroplets` (or include `microdroplets` in `SERVICES`). Requires a valid `DIGITALOCEAN_API_TOKEN`.

Tool names keep the `microdroplet-*` prefix (same as the service flag). Paths follow the current public API (`/v2/microvms`, not `/instances`).

## Tools

| Tool | REST | Notes |
| --- | --- | --- |
| `microdroplet-create` | `POST /v2/microvms` | `source` is exactly one of `oci_ref` or `checkpoint_id` |
| `microdroplet-list` | `GET /v2/microvms` | Paginated |
| `microdroplet-get` | `GET /v2/microvms/{id}` | |
| `microdroplet-delete` | `DELETE /v2/microvms/{id}` | Destructive |
| `microdroplet-pause` | `POST /v2/microvms/{id}/pause` | Sync, idempotent |
| `microdroplet-resume` | `POST /v2/microvms/{id}/resume` | Sync, idempotent |
| `microdroplet-checkpoint-create` | `POST /v2/microvms/{id}/checkpoints` | Async; poll get/list |
| `microdroplet-checkpoint-list` | `GET /v2/microvms/checkpoints` | Optional `microvm_id` filter |
| `microdroplet-checkpoint-get` | `GET /v2/microvms/checkpoints/{id}` | |
| `microdroplet-checkpoint-delete` | `DELETE /v2/microvms/checkpoints/{id}` | Destructive |

Out of scope: exec/PTY, Images API.
