## Signals MCP Tools

This directory contains tools for investigating DigitalOcean Signals on Agent Platform conversations: collection consent, sessions, dialogue turns, and exports. All operations are exposed as tools with argument-based input—no resource URIs are used. Cursor pagination and filtering are supported where applicable.

Enable these tools with service name **`signals`**. Hosted URL: `https://signals.mcp.digitalocean.com/mcp`.

---

## Supported Tools

### Collection consent

- **signals-list-consents**  
  List collection consent records for the team.  
  **Arguments:** None

- **signals-get-agent-consent**  
  Get collection consent for one agent.  
  **Arguments:**
    - `AgentID` (string, required): Environment Config ID (agent ID)

- **signals-set-agent-consent**  
  Enable or disable collection for one agent.  
  **Arguments:**
    - `AgentID` (string, required): Environment Config ID (agent ID)
    - `Enabled` (boolean, required): `true` to enable, `false` to disable

### Sessions and dialogues

- **signals-list-agent-sessions**  
  List sessions for an agent.  
  **Arguments:**
    - `AgentID` (string, required): Environment Config ID (agent ID)
    - `Limit` (number, default: 20, max: 100): Page size
    - `After` (string, optional): Cursor from `page_info.end_cursor`
    - `StartTime` (number, optional): Lower bound (Unix epoch seconds)
    - `EndTime` (number, optional): Upper bound (Unix epoch seconds)
    - `SignalType` (array of strings, optional): Filter by signal type

- **signals-list-session-dialogues**  
  List dialogue turns for a session.  
  **Arguments:**
    - `SessionID` (string, required): Session ID
    - `Limit` (number, default: 20, max: 100): Page size
    - `After` (string, optional): Cursor from `page_info.end_cursor`
    - `Before` (string, optional): Upper cursor bound
    - `ContinueSession` (boolean, optional): Continue from the session cursor
    - `StartTime` (number, optional): Lower bound (Unix epoch seconds)
    - `EndTime` (number, optional): Upper bound (Unix epoch seconds)
    - `SignalType` (array of strings, optional): Filter by signal type

### Exports

- **signals-get-export-options**  
  List exportable entity types.  
  **Arguments:** None

- **signals-create-export**  
  Create an export job for an agent.  
  **Arguments:**
    - `AgentID` (string, required): Environment Config ID (agent ID)
    - `SignalType` (array of strings, optional): Filter by signal type
    - `StartTime` (number, optional): Lower bound (Unix epoch seconds)
    - `EndTime` (number, optional): Upper bound (Unix epoch seconds)

- **signals-list-exports**  
  List prior export jobs.  
  **Arguments:**
    - `AgentID` (string, optional): Environment Config ID (agent ID)
    - `Limit` (number, default: 20, max: 100): Page size
    - `After` (string, optional): Cursor from `page_info.end_cursor`

- **signals-get-export**  
  Get one export job.  
  **Arguments:**
    - `ExportID` (string, required): Export job ID

- **signals-get-export-download**  
  Get a short-lived download URL for a completed export.  
  **Arguments:**
    - `ExportID` (string, required): Export job ID

---

## Example Usage

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

- **List sessions for an agent:**  
  Tool: `signals-list-agent-sessions`  
  Arguments:
    - `AgentID`: `"019fb39c-14d9-7080-933e-b9b90e25acda"`
    - `Limit`: `20`

- **List dialogue turns in a session:**  
  Tool: `signals-list-session-dialogues`  
  Arguments:
    - `SessionID`: `"01a10d59-80b2-70cf-8c2e-651ebb200ef8"`
    - `Limit`: `50`

- **Create an export:**  
  Tool: `signals-create-export`  
  Arguments:
    - `AgentID`: `"019fb39c-14d9-7080-933e-b9b90e25acda"`

- **Get an export job:**  
  Tool: `signals-get-export`  
  Arguments:
    - `ExportID`: `"export-1"`

- **Download a completed export:**  
  Tool: `signals-get-export-download`  
  Arguments:
    - `ExportID`: `"export-1"`

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- Pagination uses `Limit` and `After`. Pass `page_info.end_cursor` as `After`.
- `AgentID` is the Harness Runtime **Environment Config ID** (environment ID). Use the same UUID from `doctl harness-runtime config list` or the Control Panel.
- Consent changes can take up to 15 minutes to apply to ingest.
- Export download URLs expire after about 15 minutes.
- All responses are returned in JSON format.
