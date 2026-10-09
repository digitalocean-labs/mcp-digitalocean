## Signals MCP Tools

This directory contains tools for investigating DigitalOcean Signals on Agent Platform conversations: collection consent, sessions, dialogue turns, and exports. All operations are exposed as tools with argument-based input—no resource URIs are used. Cursor pagination (`Limit` / `After`) is supported where applicable.

Enable these tools with service name **`signals`**. Hosted URL: `https://signals.mcp.digitalocean.com/mcp`.

---

## Supported Tools

### Collection consent

- **signals-list-consents**  
  List Signals collection consent records for the authenticated team.  
  **Arguments:** None

- **signals-get-agent-consent**  
  Get Signals collection consent for one agent. If no consent row exists, collection is treated as denied (`enabled=false`, `allowed=false`).  
  **Arguments:**
    - `AgentID` (string, required): Agent ID (Harness Runtime Environment Config UUID / agent config ID)

- **signals-set-agent-consent**  
  Enable or disable Signals collection for one agent. Ingest may take up to 15 minutes to honor a consent change.  
  **Arguments:**
    - `AgentID` (string, required): Agent ID to update
    - `Enabled` (boolean, required): `true` to collect Signals data; `false` to deny collection

### Sessions and dialogues

- **signals-list-agent-sessions**  
  List conversation sessions for an agent as a cursor page of session summaries. Use a returned `session_id` with `signals-list-session-dialogues`.  
  **Arguments:**
    - `AgentID` (string, required): Agent ID whose sessions to list
    - `Limit` (number, optional): Page size (default 20, max 100)
    - `After` (string, optional): Cursor from the previous page's `page_info.end_cursor`
    - `StartTime` (number, optional): Lower bound as Unix epoch seconds
    - `EndTime` (number, optional): Upper bound as Unix epoch seconds
    - `SignalType` (array of strings, optional): Filter to sessions that include any of these signal types (OR)

- **signals-list-session-dialogues**  
  List dialogue turns for a session, including nested signal instances when present.  
  **Arguments:**
    - `SessionID` (string, required): Session ID from `signals-list-agent-sessions`
    - `Limit` (number, optional): Page size (default 20, max 100)
    - `After` (string, optional): Cursor from the previous page's `page_info.end_cursor`
    - `Before` (string, optional): Upper cursor bound for the page
    - `ContinueSession` (boolean, optional): When `true`, continue paging within the current session using the session cursor
    - `StartTime` (number, optional): Lower bound as Unix epoch seconds
    - `EndTime` (number, optional): Upper bound as Unix epoch seconds
    - `SignalType` (array of strings, optional): Filter dialogues that include any of these signal types (OR)

### Exports

- **signals-get-export-options**  
  List exportable entity types and filter options for create-export requests.  
  **Arguments:** None

- **signals-create-export**  
  Create a Signals export job for an agent. Poll with `signals-get-export` until `status` is `completed` or `failed`, then call `signals-get-export-download`.  
  **Arguments:**
    - `AgentID` (string, required): Agent ID whose Signals data to export
    - `SignalType` (array of strings, optional): Limit the export to these signal types
    - `StartTime` (number, optional): Lower bound as Unix epoch seconds
    - `EndTime` (number, optional): Upper bound as Unix epoch seconds

- **signals-list-exports**  
  List prior export jobs for the team.  
  **Arguments:**
    - `AgentID` (string, optional): If set, only exports for this agent
    - `Limit` (number, optional): Page size (default 20, max 100)
    - `After` (string, optional): Cursor from the previous page's `page_info.end_cursor`

- **signals-get-export**  
  Get one export job by ID (status, filters, timestamps). Use this to poll until the job finishes.  
  **Arguments:**
    - `ExportID` (string, required): Export job ID from `signals-create-export` or `signals-list-exports`

- **signals-get-export-download**  
  Get a short-lived (about 15 minutes) presigned Spaces download URL for a completed export. Do not persist the URL.  
  **Arguments:**
    - `ExportID` (string, required): Export job ID whose status is `completed`

---

## Example usage

- **List consent for the team:**  
  Tool: `signals-list-consents`  
  Arguments: `{}`

- **Get consent for one agent:**  
  Tool: `signals-get-agent-consent`  
  Arguments:
    - `AgentID`: `"019fb39c-14d9-7080-933e-b9b90e25acda"`

- **Enable collection for an agent:**  
  Tool: `signals-set-agent-consent`  
  Arguments:
    - `AgentID`: `"019fb39c-14d9-7080-933e-b9b90e25acda"`
    - `Enabled`: `true`

- **List recent sessions:**  
  Tool: `signals-list-agent-sessions`  
  Arguments:
    - `AgentID`: `"019fb39c-14d9-7080-933e-b9b90e25acda"`
    - `Limit`: `20`

- **Show dialogue turns in a session:**  
  Tool: `signals-list-session-dialogues`  
  Arguments:
    - `SessionID`: `"01a10d59-80b2-70cf-8c2e-651ebb200ef8"`
    - `Limit`: `50`

- **Create an export, then download it:**  
  1. Tool: `signals-create-export` with `AgentID` (optional `SignalType`, `StartTime`, `EndTime`)  
  2. Tool: `signals-get-export` with `ExportID` until `status` is `completed`  
  3. Tool: `signals-get-export-download` with `ExportID`

---

## Example queries using Signals MCP Tools

- Show collection consent for this agent.
- Enable Signals collection for agent X.
- List recent sessions for this agent.
- Show the dialogue turns in this session.
- Create an export for this agent, then download it when it completes.

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- List endpoints that support paging use `Limit` and `After`. Pass `page_info.end_cursor` from the previous response as `After`.
- `AgentID` is the Harness Runtime Environment Config UUID (the same ID `doctl harness-runtime config list` returns).
- Consent changes can take up to 15 minutes to apply to ingest.
- Export download URLs expire quickly; fetch a new URL with `signals-get-export-download` if needed.
- All responses are returned as JSON.
