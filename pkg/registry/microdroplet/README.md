# MicroVM MCP tools

Lifecycle and one-shot exec tools for DigitalOcean MicroVMs. Arguments use
**snake_case** and mirror the public REST API under `/v2/microvms`.

Activate with `--services microvms` (or include `microvms` in `SERVICES`).
Requires a valid DigitalOcean token (or OAuth via a hosted remote MCP client).

Tool names use the `microvm-*` prefix (service flag is `microvms`).

**Hosted endpoint:** `https://microvms.mcp.digitalocean.com/mcp`

## Tools

| Tool                        | REST                                   | Notes                                                                 |
| --------------------------- | -------------------------------------- | --------------------------------------------------------------------- |
| `microvm-create`            | `POST /v2/microvms`                    | `source` is exactly one of `oci_ref` or `checkpoint_id`              |
| `microvm-list`              | `GET /v2/microvms`                     | Paginated                                                             |
| `microvm-get`               | `GET /v2/microvms/{id}`                |                                                                       |
| `microvm-delete`            | `DELETE /v2/microvms/{id}`             | Destructive                                                           |
| `microvm-pause`             | `POST /v2/microvms/{id}/pause`         | Sync, idempotent                                                      |
| `microvm-resume`            | `POST /v2/microvms/{id}/resume`        | Sync, idempotent                                                      |
| `microvm-exec`              | `POST /v2/microvms/{id}/exec`          | One-shot non-PTY; needs team `exec_pty` feature; see below            |
| `microvm-checkpoint-create` | `POST /v2/microvms/{id}/checkpoints`   | Async; poll get/list                                                  |
| `microvm-checkpoint-list`   | `GET /v2/microvms/checkpoints`         | Optional `microvm_id` filter                                          |
| `microvm-checkpoint-get`    | `GET /v2/microvms/checkpoints/{id}`    |                                                                       |
| `microvm-checkpoint-delete` | `DELETE /v2/microvms/checkpoints/{id}` | Destructive                                                           |

Out of scope as MCP tools: interactive console WebSocket (`GET …/console`),
Images API.

---

## Tool reference

### `microvm-create`

Create a MicroVM. `source` must have exactly one of `oci_ref` or
`checkpoint_id`. `region` and `size` are required with `oci_ref` (inherited
from the checkpoint when restoring).

| Argument         | Type    | Required | Notes                                      |
| ---------------- | ------- | -------- | ------------------------------------------ |
| `name`           | string  | yes      | Human-readable name                        |
| `source`         | object  | yes      | `{ "oci_ref": "…" }` or `{ "checkpoint_id": "…" }` |
| `region`         | string  | with oci | e.g. `mkc1`                                |
| `size`           | object  | with oci | `{ "cpu": 1, "memory": 2048 }` (MiB)       |
| `networking`     | string  | no       | `public` or `vpc`                          |
| `vpc_uuid`       | string  | if vpc   |                                            |
| `http_port`      | number  | no       | Ingress / auto-resume probe port           |
| `ports`          | int[]   | no       | Guest ports (max 5; include `http_port`)   |
| `http_protocol`  | string  | no       | `http` or `http2`                          |
| `auto_pause`     | object  | no       | `{ "idle_timeout" }` only — `enabled: false` is rejected (auto-pause cannot be turned off) |
| `auto_resume`    | boolean | no       | Default true                               |
| `environment`    | object  | no       | Key/value env at boot                      |
| `tags`           | string[]| no       |                                            |

Example:

```json
{
  "name": "demo-nginx",
  "region": "mkc1",
  "size": { "cpu": 1, "memory": 2048 },
  "source": { "oci_ref": "docker.io/library/nginx:stable" },
  "networking": "public",
  "http_port": 80,
  "ports": [80]
}
```

### `microvm-list`

| Argument   | Type   | Notes                         |
| ---------- | ------ | ----------------------------- |
| `name`     | string | Filter by name                |
| `region`   | string | Filter by region slug         |
| `tag_name` | string | Filter by tag                 |
| `page`     | number | Default 1                     |
| `per_page` | number | Default 25, max 200           |

### `microvm-get` / `microvm-delete` / `microvm-pause` / `microvm-resume`

| Argument | Type   | Required |
| -------- | ------ | -------- |
| `id`     | string | yes (MicroVM UUID) |

### `microvm-exec`

Run a **one-shot, non-interactive** command in a running MicroVM. Requires the
team feature gate **`exec_pty`**. There is no stdin; commands that run longer
than about **50 seconds** may time out. A non-zero `exit_code` is still a
successful tool call (HTTP 200).

| Argument           | Type     | Required | Notes                                      |
| ------------------ | -------- | -------- | ------------------------------------------ |
| `id`               | string   | yes      | MicroVM UUID                               |
| `argv`             | string[] | yes      | Non-empty; `argv[0]` is the command        |
| `cwd`              | string   | no       | Working directory                          |
| `output_encoding`  | string   | no       | `utf-8` (default) or `base64` for binary   |

Example:

```json
{
  "id": "microvm-uuid",
  "argv": ["echo", "hi"],
  "cwd": "/app",
  "output_encoding": "utf-8"
}
```

Response fields: `stdout`, `stderr`, `exit_code`, optional `truncated` /
`output_encoding`.

### Checkpoint tools

| Tool                          | Arguments                                      |
| ----------------------------- | ---------------------------------------------- |
| `microvm-checkpoint-create`   | `id` (MicroVM UUID), optional `name`           |
| `microvm-checkpoint-list`     | optional `microvm_id`, `page`, `per_page`      |
| `microvm-checkpoint-get`      | `id` (checkpoint UUID)                         |
| `microvm-checkpoint-delete`   | `id` (checkpoint UUID)                         |

Checkpoint create is async — poll `microvm-checkpoint-get` until status is
`CHECKPOINT_AVAILABLE` or `CHECKPOINT_FAILED`. Deleting a checkpoint while
capture is still in progress returns **400** (that restriction is on the
checkpoint delete, not on deleting the MicroVM).

---

## Agent-client walkthrough (Claude Code)

Verified path: add the hosted MicroVM MCP server, create a MicroVM, inspect it,
then delete it. Uses OAuth (recommended) so you do not put a PAT in config.

### 1. Add the remote server

```bash
claude mcp add --transport http digitalocean-microvms https://microvms.mcp.digitalocean.com/mcp
```

On first tool use, Claude opens a browser for DigitalOcean sign-in.

**API token alternative** (shell expands the token once into Claude’s secure store):

```bash
claude mcp add --transport http digitalocean-microvms https://microvms.mcp.digitalocean.com/mcp \
  --header "Authorization: Bearer ${DIGITALOCEAN_API_TOKEN}"
```

### 2. Create a MicroVM

In Claude Code, ask something like:

> Using the DigitalOcean MicroVM tools, create a MicroVM named
> `tmp-mcp-walkthrough` in region `mkc1` with size cpu=1 memory=2048,
> image `docker.io/library/nginx:stable`, public networking, http_port 80.

Expected tool: `microvm-create`. Poll with `microvm-get` until `state` is
`running` (or `failed`).

### 3. Optional: run a command

If your team has `exec_pty` enabled:

> Run `["uname", "-a"]` in that MicroVM with `microvm-exec`.

### 4. Clean up

> Delete the MicroVM with `microvm-delete`.

Confirm with `microvm-list` that it is gone.

### Cursor / VS Code

Add the same URL under MCP settings (OAuth or Bearer token). Tool names and
arguments are identical to the table above. See the root
[README](../../../README.md) for `mcp.json` patterns used by other DigitalOcean
hosted services.

---

## Action Gateway

Hosted Agents discover these tools via the DigitalOcean Action Gateway catalog
(`digitalocean_microvm-*` names). Registration SoT is
`digitalocean/tool-registry-manifest` (`providers/digitalocean/`), not local
cthulhu seed SQL.
