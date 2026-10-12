# MicroVM MCP tools

Lifecycle tools for DigitalOcean MicroVMs over `POST/GET/DELETE /v2/microvms/*`. REST JSON is passed through; argument names mirror the public API (**snake_case**).

Activate with `--services microvms` (or include `microvms` in `SERVICES`). Requires a valid `DIGITALOCEAN_API_TOKEN`.

Tool names use the `microvm-*` prefix (service flag is `microvms`). Paths follow the current public API (`/v2/microvms`, not `/instances`).

## Tools

| Tool                        | REST                                   | Notes                                                   |
| --------------------------- | -------------------------------------- | ------------------------------------------------------- |
| `microvm-create`            | `POST /v2/microvms`                    | `source` is exactly one of `oci_ref` or `checkpoint_id` |
| `microvm-list`              | `GET /v2/microvms`                     | Paginated                                               |
| `microvm-get`               | `GET /v2/microvms/{id}`                |                                                         |
| `microvm-delete`            | `DELETE /v2/microvms/{id}`             | Destructive                                             |
| `microvm-pause`             | `POST /v2/microvms/{id}/pause`         | Sync, idempotent                                        |
| `microvm-resume`            | `POST /v2/microvms/{id}/resume`        | Sync, idempotent                                        |
| `microvm-exec`              | `POST /v2/microvms/{id}/exec`          | One-shot non-PTY; needs `exec_pty` gate                 |
| `microvm-checkpoint-create` | `POST /v2/microvms/{id}/checkpoints`   | Async; poll get/list                                    |
| `microvm-checkpoint-list`   | `GET /v2/microvms/checkpoints`         | Optional `microvm_id` filter                            |
| `microvm-checkpoint-get`    | `GET /v2/microvms/checkpoints/{id}`    |                                                         |
| `microvm-checkpoint-delete` | `DELETE /v2/microvms/checkpoints/{id}` | Destructive                                             |

### `microvm-exec`

```json
{
  "id": "microvm-uuid",
  "argv": ["echo", "hi"],
  "cwd": "/app",
  "output_encoding": "utf-8"
}
```

Response fields: `stdout`, `stderr`, `exit_code`, optional `truncated` / `output_encoding`. A non-zero `exit_code` is still a successful tool call (HTTP 200).

Out of scope: interactive console WebSocket (`GET …/console`), Images API.
