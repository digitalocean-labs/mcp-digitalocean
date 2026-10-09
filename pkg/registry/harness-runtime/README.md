## Harness Runtime MCP Tools

Tools for discovering and managing Hosted Agents (Harness Runtime), backed by the public `/v2/agents/*` API.

Enable these tools with service name **`harness-runtime`**. Hosted URL: `https://harness-runtime.mcp.digitalocean.com/mcp`.

---

## Supported Tools

- **`harness-runtime-list-agent-configs`** — List the active agent configs for the team (id, name, content hash, creator, timestamps). Supports `PageSize` (1-200, default 50), `PageToken` (cursor from `next_page_token`) and `Search` (case-insensitive name substring, max 64 chars). Manifests are not returned.

Requires the `agent_harness_session:read` scope.

---

## Example queries

- List my hosted agents.
- Find the agent config named "support" so I can run a simulation against it.
