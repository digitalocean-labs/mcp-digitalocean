## Harness Runtime MCP Tools

Tools for discovering and managing Hosted Agents (Harness Runtime), backed by the public `/v2/agents/*` API.

Enable these tools with service name **`harness-runtime`**. Hosted URL: `https://harness-runtime.mcp.digitalocean.com/mcp`.

---

## Supported Tools

### Agent Configs

- **`harness-runtime-list-agent-configs`** — List the active agent configs for the team (id, name, content hash, creator, timestamps). Supports `PageSize` (1-200, default 50), `PageToken` (cursor from `next_page_token`) and `Search` (case-insensitive name substring, max 64 chars). Manifests are not returned.
- **`harness-runtime-get-agent-config`** — Get one agent config by `ConfigID` (UUID), including its sanitized manifest and redacted credential slot metadata. Secret values are never returned.
- **`harness-runtime-create-agent-config`** — Create a new agent config from a `Name` (required, 1-64 chars) and `Manifest` (required, JSON object or YAML string with the agent environment spec — e.g. `agent`, `size`, `env`, `permissions`, `egress`, `secrets`). Optionally set `ManageModelAccessKey` to have the platform provision a serverless-inference key. Configs are immutable after creation. Returns the created config with its assigned UUID. **SECURITY:** any values under the manifest's top-level `secrets` map are write-only — extracted into Secrets Manager and never persisted or echoed back by this or any other tool. Never commit real secret values to files/source control; prefer environment variables or the DO Console over pasting them into a terminal that logs history.
- **`harness-runtime-delete-agent-config`** — Permanently delete an agent config by `ConfigID` (UUID). **Destructive.** Existing sessions that reference this config are retained.
- **`harness-runtime-list-config-sessions`** — List all sessions created from a specific agent config by `ConfigID`. Supports `PageSize`, `PageToken`, and `Status` filter.

### Sessions

- **`harness-runtime-create-session`** — Provision a new session from an existing Agent Config (`ConfigID` + `Name` required). Returns the session row immediately; sandbox boot completes asynchronously. Optional: `ResumeOnTopoff` (auto-resume consent), `WorkspaceID` (persistent workspace).
- **`harness-runtime-get-session`** — Get the current snapshot of a session by `SessionID` (UUID), including status, agent kind, sandbox id, and lineage.
- **`harness-runtime-list-sessions`** — List sessions for the team (newest first). By default omits destroyed and failed sessions; pass `Status=SESSION_STATUS_ALL` to include every session. Supports `PageSize`, `PageToken`, `Status`, `Name` (exact-match filter), and `Search` (substring filter).
- **`harness-runtime-destroy-session`** — Permanently tear down a session's sandbox. **Destructive.** Workspace contents are retained per tenant policy.
- **`harness-runtime-send-input`** — Deliver a user message into a READY or DETACHED session. Requires `SessionID` and at least one of `Text` (string) or `Inputs` (structured dict). Returns the `run_id`.
- **`harness-runtime-pause-session`** — Freeze a READY/DETACHED session's microVM sandbox. microvm-substrate only.
- **`harness-runtime-resume-session`** — Explicitly thaw a PAUSED session's sandbox. Returns synchronously once ready.
- **`harness-runtime-patch-session`** — Update session-scoped settings. Currently supports `ResumeOnTopoff` (auto-resume consent toggle). Returns the updated session snapshot.
- **`harness-runtime-resolve-hitl`** — Record the user's decision on a pending HITL request. `Outcome` must be one of `HITL_OUTCOME_APPROVE`, `HITL_OUTCOME_REJECT`, or `HITL_OUTCOME_DEFER`. Optional: `Reason` (audit trail text), `Content` (structured form answer).

### Sandbox

- **`harness-runtime-list-sandbox-sizes`** — List the available sandbox compute sizes (slug, vCPUs, memory, hourly and monthly pricing). The returned slugs are valid values for sandbox size fields in manifests.
- **`harness-runtime-validate-policy`** — Stateless preflight: check a manifest's tool-permission policy against the agent's capabilities without creating a session. Takes `Manifest` (same JSON object as create-agent-config, must include a `permissions` block). Returns per-rule verdicts (`exact`, `degraded`, `unsupported`, `unverified`); `ok=false` means CreateSession would reject this policy.

Requires the `agent_harness_session:read` scope (read tools) and `agent_harness_session:create` / `agent_harness_session:delete` (write tools).

---

## Example queries

- List my hosted agent configs.
- Show me the manifest of agent config 019fb39c-14d9-7080-933e-b9b90e25acda.
- Find the agent config named "support" so I can run a simulation against it.
- Create a new agent config from this manifest JSON.
- Delete agent config 019fb39c-14d9-7080-933e-b9b90e25acda.
- List all sessions created from config 019fb39c-14d9-7080-933e-b9b90e25acda.
- Create a new session from config 019fb39c-14d9-7080-933e-b9b90e25acda named "test-run-1".
- List all my active sessions.
- What is the status of session 029ab39c-14d9-7080-933e-b9b90e25acda?
- Send a message to my session: "Summarize the latest GitHub issues."
- Pause session 029ab39c-14d9-7080-933e-b9b90e25acda.
- Resume the paused session.
- Approve the HITL request on session 029ab39c-14d9-7080-933e-b9b90e25acda.
- Enable auto-resume on session 029ab39c-14d9-7080-933e-b9b90e25acda so it restarts automatically when the balance is restored.
- Destroy session 029ab39c-14d9-7080-933e-b9b90e25acda.
- What sandbox sizes are available?
- Validate this policy against session 029ab39c-14d9-7080-933e-b9b90e25acda.
