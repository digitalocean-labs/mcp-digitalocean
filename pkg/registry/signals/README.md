## Signals MCP Tools

Tools for investigating Agent Platform sessions, segments, dialogue, and exports, and for managing Signals collection consent.

Enable these tools with service name **`signals`**. Hosted URL: `https://signals.mcp.digitalocean.com/mcp`.

---

## Supported Tools

- **`signals-list-consents`** — List collection consent records for the team.
- **`signals-get-agent-consent`** — Get collection consent for one agent. Missing consent is treated as deny.
- **`signals-set-agent-consent`** — Enable or disable collection for one agent.
- **`signals-list-agent-sessions`** — List sessions for an agent.
- **`signals-list-session-dialogues`** — List dialogue turns for a session.
- **`signals-create-export`** — Create an export job for an agent.
- **`signals-list-exports`** — List prior export jobs.
- **`signals-get-export`** — Get the status of one export job.
- **`signals-get-export-download`** — Get a short-lived download URL for a completed export.
- **`signals-get-export-options`** — List exportable entity types.

---

## Example queries using Signals MCP Tools

- Show collection consent for this agent.
- Enable Signals collection for agent X.
- List recent sessions for this agent.
- Show the dialogue turns in this session.
- Create an export for this agent, then download it when it completes.
