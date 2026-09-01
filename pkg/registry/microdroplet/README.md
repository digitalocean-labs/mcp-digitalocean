# MicroDroplet MCP tools

Lifecycle tools for [MicroDroplet](https://docs.digitalocean.com/products/microdroplets/) over `POST/GET/DELETE /v2/microdroplets/*`. REST JSON is passed through; argument names mirror the public API (**snake_case**).

Activate with `--services microdroplets` (or include `microdroplets` in `SERVICES`). Requires a valid `DIGITALOCEAN_API_TOKEN`.

## Tools

| Tool | REST | Notes |
| --- | --- | --- |
| `microdroplet-create` | `POST /instances` | Exactly one of `image`, `checkpoint`, `container` |
| `microdroplet-list` | `GET /instances` | Paginated |
| `microdroplet-get` | `GET /instances/{id}` | |
| `microdroplet-delete` | `DELETE /instances/{id}` | Destructive |
| `microdroplet-pause` | `POST /instances/{id}/pause` | Sync, idempotent |
| `microdroplet-resume` | `POST /instances/{id}/resume` | Sync, idempotent |
| `microdroplet-checkpoint-create` | `POST /instances/{id}/checkpoints` | Async; poll get/list |
| `microdroplet-checkpoint-list` | `GET /checkpoints` | Optional `micro_droplet_id` filter |
| `microdroplet-checkpoint-get` | `GET /checkpoints/{id}` | |
| `microdroplet-checkpoint-delete` | `DELETE /checkpoints/{id}` | Destructive |

Out of scope: exec/PTY, Images API.

Schema source (cthulhu): `docode/src/do/services/microdroplet/docs/specs/mcp-tools/tools.json` (MDROP-237).
