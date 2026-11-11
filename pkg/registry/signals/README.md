## Signals MCP Tools

Tools for managing Signals consent, listing sessions and dialogues, and exporting Signals data. All operations use argument-based input, no resource URIs. Pagination and filtering are supported where applicable.

Enable these tools with service name **`signals`**. Hosted URL: `https://signals.mcp.digitalocean.com/mcp`.

---

## Supported Tools

### Consent

- **signals-list-consents**  
  List Signals consent for the team.  
  **Arguments:** None

- **signals-get-agent-consent**  
  Get Signals consent for one environment. For serverless inference, use `signals-get-inference-consent`.  
  **Arguments:**
    - `EnvironmentID` (string, required*): Environment ID
    - `AgentID` (string, deprecated): use `EnvironmentID`

- **signals-set-agent-consent**  
  Enable or disable Signals for one environment. For serverless inference, use `signals-set-inference-consent`.  
  **Arguments:**
    - `EnvironmentID` (string, required*): Environment ID
    - `AgentID` (string, deprecated): use `EnvironmentID`
    - `Enabled` (boolean, required): `true` to enable, `false` to disable

- **signals-get-inference-consent**  
  Get the team's Signals consent for serverless inference traffic. Returns `enabled: false` when the team has never set it.  
  **Arguments:** None

- **signals-set-inference-consent**  
  Enable or disable Signals for all serverless inference traffic of the team.  
  **Arguments:**
    - `Enabled` (boolean, required): `true` to enable, `false` to disable

### Sessions and dialogues

- **signals-list-agent-sessions**  
  List sessions for an environment, or for serverless inference.  
  **Arguments:**
    - `EnvironmentID` (string, required*): Environment ID. Use `00000000-0000-0000-0000-000000000000` for serverless inference.
    - `AgentID` (string, deprecated): use `EnvironmentID`
    - `Limit` (number, default: 20, max: 100): Page size
    - `After` (string, optional): Cursor from `page_info.end_cursor`
    - `StartTime` (number, optional): Lower bound (Unix epoch seconds)
    - `EndTime` (number, optional): Upper bound (Unix epoch seconds)
    - `SignalType` (array of strings, optional): Filter by signal type

- **signals-list-session-dialogues**  
  List dialogues for a session.  
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
  List export filter options.  
  **Arguments:** None

- **signals-create-export**  
  Create an export for an environment, or for serverless inference.  
  **Arguments:**
    - `EnvironmentID` (string, required*): Environment ID. Use `00000000-0000-0000-0000-000000000000` for serverless inference.
    - `AgentID` (string, deprecated): use `EnvironmentID`
    - `SignalType` (array of strings, optional): Filter by signal type
    - `StartTime` (number, optional): Lower bound (Unix epoch seconds)
    - `EndTime` (number, optional): Upper bound (Unix epoch seconds)

- **signals-list-exports**  
  List exports for the team.  
  **Arguments:**
    - `EnvironmentID` (string, optional): Environment ID. Use `00000000-0000-0000-0000-000000000000` for serverless inference.
    - `AgentID` (string, deprecated): use `EnvironmentID`
    - `Limit` (number, default: 20, max: 100): Page size
    - `After` (string, optional): Cursor from `page_info.end_cursor`

- **signals-get-export**  
  Get an export by ID.  
  **Arguments:**
    - `ExportID` (string, required): Export ID

- **signals-get-export-download**  
  Get a short-lived download URL for a completed export.  
  **Arguments:**
    - `ExportID` (string, required): Export ID

\* Provide `EnvironmentID` or deprecated `AgentID` (or both when equal).

---

## Example Usage

- **List consent for the team:**  
  Tool: `signals-list-consents`  
  Arguments: `{}`

- **Get consent for one environment:**  
  Tool: `signals-get-agent-consent`  
  Arguments:
    - `EnvironmentID`: `"019fb39c-14d9-7080-933e-b9b90e25acda"`

- **Enable Signals for an environment:**  
  Tool: `signals-set-agent-consent`  
  Arguments:
    - `EnvironmentID`: `"019fb39c-14d9-7080-933e-b9b90e25acda"`
    - `Enabled`: `true`

- **Get serverless inference consent:**  
  Tool: `signals-get-inference-consent`  
  Arguments: `{}`

- **Enable Signals for serverless inference:**  
  Tool: `signals-set-inference-consent`  
  Arguments:
    - `Enabled`: `true`

- **List sessions for an environment:**  
  Tool: `signals-list-agent-sessions`  
  Arguments:
    - `EnvironmentID`: `"019fb39c-14d9-7080-933e-b9b90e25acda"`
    - `Limit`: `20`

- **List serverless inference sessions:**  
  Tool: `signals-list-agent-sessions`  
  Arguments:
    - `EnvironmentID`: `"00000000-0000-0000-0000-000000000000"`
    - `Limit`: `20`

- **List dialogues for a session:**  
  Tool: `signals-list-session-dialogues`  
  Arguments:
    - `SessionID`: `"01a10d59-80b2-70cf-8c2e-651ebb200ef8"`
    - `Limit`: `50`

- **Create an export:**  
  Tool: `signals-create-export`  
  Arguments:
    - `EnvironmentID`: `"019fb39c-14d9-7080-933e-b9b90e25acda"`

- **Get an export:**  
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
- The `EnvironmentID` argument is the Environment Config ID. Run `doctl harness-runtime config list` to find it. `AgentID` is a deprecated alias for the same value.
- Serverless inference sessions and exports use the `EnvironmentID` `00000000-0000-0000-0000-000000000000`. Inference consent is a single team-level setting; manage it with `signals-get-inference-consent` and `signals-set-inference-consent`, not the agent consent tools.
- After you change consent, ingest can take up to 15 minutes to apply it.
- Export download URLs expire after about 15 minutes.
- Responses are JSON.
