# MicroVM MCP tools

Lifecycle tools for DigitalOcean MicroVMs over `POST/GET/DELETE /v2/microvms/*`. REST JSON is passed through; argument names mirror the public API (**snake_case**).

Activate with `--services microvms` (or include `microvms` in `SERVICES`). Requires a valid `DIGITALOCEAN_API_TOKEN`.

Tool names use the `microvm-*` prefix (service flag is `microvms`). Paths follow the current public API (`/v2/microvms`, not `/instances`).

## Tools

| Tool | REST | Notes |
| --- | --- | --- |
| `microvm-create` | `POST /v2/microvms` | `source` is exactly one of `oci_ref` or `checkpoint_id` |
| `microvm-list` | `GET /v2/microvms` | Paginated |
| `microvm-get` | `GET /v2/microvms/{id}` | |
| `microvm-delete` | `DELETE /v2/microvms/{id}` | Destructive |
| `microvm-pause` | `POST /v2/microvms/{id}/pause` | Sync, idempotent |
| `microvm-resume` | `POST /v2/microvms/{id}/resume` | Sync, idempotent |
| `microvm-checkpoint-create` | `POST /v2/microvms/{id}/checkpoints` | Async; poll get/list |
| `microvm-checkpoint-list` | `GET /v2/microvms/checkpoints` | Optional `microvm_id` filter |
| `microvm-checkpoint-get` | `GET /v2/microvms/checkpoints/{id}` | |
| `microvm-checkpoint-delete` | `DELETE /v2/microvms/checkpoints/{id}` | Destructive |

Out of scope: exec/PTY, Images API.
