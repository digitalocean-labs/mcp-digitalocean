## MicroVM MCP Tools

Lifecycle tools for DigitalOcean MicroVMs over public REST `/v2/microvms/*`.
REST JSON is passed through; argument names mirror the public API (**snake_case**).
All operations are argument-based tools (no resource URIs). Destructive tools
return a short success message; other tools declare an `outputSchema` and emit
`structuredContent`.

Enable with service name **`microvms`**. Hosted URL:
`https://microvms.mcp.digitalocean.com/mcp`.

Out of scope: exec/PTY, Images API.

---

## Supported Tools

### MicroVM lifecycle

- **microvm-create**  
  Create a MicroVM. `source` must have exactly one of `oci_ref` or
  `checkpoint_id`. `region` and `size` are required with `oci_ref`; both may be
  inherited from the checkpoint when restoring.  
  **Arguments:**

  - `name` (string, required): Human-readable name
  - `source` (object, required): Exactly one of:
    - `oci_ref` (string): OCI reference (public Hub or team DOCR)
    - `checkpoint_id` (string): Checkpoint UUID to restore
  - `region` (string, optional): Region slug (e.g. `nyc1`). Required with `oci_ref`
  - `size` (object, optional): `{ "cpu": <int>, "memory": <int MiB> }`. Required with `oci_ref`
  - `networking` (string, optional): `public` or `vpc`
  - `vpc_uuid` (string, optional): Required when `networking` is `vpc`
  - `auto_pause` (object, optional): `{ "enabled": <bool>, "idle_timeout": "<duration>" }`
  - `auto_resume` (boolean, optional): Auto-resume on HTTP traffic (default true)
  - `http_port` (number, optional): HTTP listen port for ingress / auto-resume
  - `http_protocol` (string, optional): `http` or `http2`
  - `ports` (array of numbers, optional): Guest ports to open (max 5)
  - `environment` (object, optional): Key-value env vars at boot
  - `tags` (array of strings, optional): Resource tags

- **microvm-list**  
  List MicroVMs for the authenticated team.  
  **Arguments:**

  - `page` (number, default: 1)
  - `per_page` (number, default: 25, max: 200)
  - `region` (string, optional): Filter by region slug
  - `name` (string, optional): Filter by name
  - `tag_name` (string, optional): Filter by tag

- **microvm-get**  
  Get a MicroVM by ID.  
  **Arguments:**

  - `id` (string, required): MicroVM UUID

- **microvm-delete**  
  Delete a MicroVM. Irreversible; checkpoints are retained until deleted separately.  
  **Arguments:**

  - `id` (string, required): MicroVM UUID

- **microvm-pause**  
  Pause a running MicroVM (synchronous, idempotent).  
  **Arguments:**

  - `id` (string, required): MicroVM UUID

- **microvm-resume**  
  Resume a paused MicroVM (synchronous, idempotent).  
  **Arguments:**
  - `id` (string, required): MicroVM UUID

### Checkpoints

- **microvm-checkpoint-create**  
  Start an async checkpoint of a running or paused MicroVM. Poll
  `microvm-checkpoint-get` until `CHECKPOINT_AVAILABLE` or `CHECKPOINT_FAILED`.  
  **Arguments:**

  - `id` (string, required): MicroVM UUID
  - `name` (string, optional): Checkpoint name

- **microvm-checkpoint-list**  
  List team checkpoints, newest first.  
  **Arguments:**

  - `page` (number, default: 1)
  - `per_page` (number, default: 25, max: 200)
  - `microvm_id` (string, optional): Filter by source MicroVM UUID

- **microvm-checkpoint-get**  
  Get a checkpoint by ID.  
  **Arguments:**

  - `id` (string, required): Checkpoint UUID

- **microvm-checkpoint-delete**  
  Delete a checkpoint. Irreversible.  
  **Arguments:**
  - `id` (string, required): Checkpoint UUID

When `microvms` is enabled, shared **`region-list`** is also registered via the
common regions helper.

---

## Example usage

- **List MicroVMs:**  
  Tool: `microvm-list`  
  Arguments: `{ "page": 1, "per_page": 10 }`

- **Create from OCI image:**  
  Tool: `microvm-create`  
  Arguments:

  ```json
  {
    "name": "agent-sandbox-1",
    "region": "nyc1",
    "size": { "cpu": 1, "memory": 1024 },
    "source": { "oci_ref": "docker.io/library/nginx:1.27" },
    "tags": ["mcp-demo"]
  }
  ```

- **Pause / resume:**  
  Tools: `microvm-pause` / `microvm-resume`  
  Arguments: `{ "id": "<microvm-uuid>" }`

- **Checkpoint then restore:**
  1. `microvm-checkpoint-create` with `{ "id": "<microvm-uuid>", "name": "snap-1" }`
  2. Poll `microvm-checkpoint-get` until status is `CHECKPOINT_AVAILABLE`
  3. `microvm-create` with `source.checkpoint_id` set to that checkpoint UUID

---

## Agent client setup

### Claude Code (remote, recommended)

```bash
# OAuth (no token in the command)
claude mcp add --transport http digitalocean-microvms \
  https://microvms.mcp.digitalocean.com/mcp

# Or API token
claude mcp add --transport http digitalocean-microvms \
  https://microvms.mcp.digitalocean.com/mcp \
  --header "Authorization: Bearer $DIGITALOCEAN_API_TOKEN"
```

Verify with `claude mcp list`, then ask Claude to list MicroVMs (calls
`microvm-list`) or create one with the example payload above.

### Cursor (remote)

Add to `~/.cursor/config.json` (OAuth — no headers):

```json
{
  "mcpServers": {
    "digitalocean-microvms": {
      "url": "https://microvms.mcp.digitalocean.com/mcp"
    }
  }
}
```

Or with a token (prefer env expansion; never commit the token):

```json
{
  "mcpServers": {
    "digitalocean-microvms": {
      "url": "https://microvms.mcp.digitalocean.com/mcp",
      "headers": {
        "Authorization": "Bearer ${DIGITALOCEAN_API_TOKEN}"
      }
    }
  }
}
```

In Cursor chat: “List my MicroVMs” or “Create a MicroVM named mcp-demo from
`docker.io/library/nginx:1.27` in nyc1 with 1 vCPU and 1024 MiB”.

### Local binary / npx

```bash
npx @digitalocean/mcp --services microvms
# or
mcp-digitalocean --transport http --services microvms
```

Requires `DIGITALOCEAN_API_TOKEN`. Point any MCP client at the local HTTP
`/mcp` endpoint when using `--transport http`.
