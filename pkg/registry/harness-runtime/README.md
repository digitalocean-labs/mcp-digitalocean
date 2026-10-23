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
- **`harness-runtime-fork-session`** — Create 1-4 child sessions from a checkpoint of an existing session. `SessionID` required; `FromCheckpointID` (READY checkpoint UUID to fork from — omit to capture an implicit checkpoint of "now") and `Count` (1-4, default 1) are optional. Returns the created child sessions with `parent_session_id` and `fork_id` set.
- **`harness-runtime-exec-in-sandbox`** — Run a one-shot command inside a session's sandbox. `SessionID` and `Argv` (non-empty string array) required. Optional: `Workdir` (default `/workspace`), `Stdin` (base64-encoded), `TimeoutSeconds` (max 240, server-enforced). Returns `exit_code`, `stdout`, `stderr`, and `duration_ms`. A non-zero exit code is still a successful HTTP response — check the field.
- **`harness-runtime-get-ingress-url`** — Get (or provision on first call) the public HTTPS ingress URL for the session's sandbox port 8000 (OpenHands Agent Server). Idempotent while the session runs; revoked on pause/lock/rollback — re-fetch after those operations. Poll until `state` is `INGRESS_URL_STATE_READY`. Only OpenHands sessions support ingress.
- **`harness-runtime-cancel-turn`** — Stop the turn currently running in a session (the Ctrl-C / Esc equivalent) and leave the session alive and idle. `SessionID` required; `RunID` optional (from `send-input`'s response — names the specific turn to cancel; omit to cancel whatever is in flight). The outcome field tells you what happened: `acked`, `no_turn`, or `unsupported`.

### Checkpoints

- **`harness-runtime-list-checkpoints`** — List a session's checkpoints newest first. `SessionID` required. Supports `PageSize` (1-200) and `PageToken` cursor.
- **`harness-runtime-get-checkpoint`** — Get the current snapshot of a checkpoint by `SessionID` and `CheckpointID`. Poll until `status` is `READY` before forking or rolling back.
- **`harness-runtime-create-checkpoint`** — Capture a session's live sandbox state as a durable checkpoint. `SessionID` required; optional `Label`. Synchronous — blocks until the checkpoint is READY and returns it directly. The session must not have an active turn in flight (409 if busy). Returns 501 for agents/substrates that do not support checkpoints.
- **`harness-runtime-delete-checkpoint`** — Delete a checkpoint and its underlying machine-state capture. `SessionID` and `CheckpointID` required. **Destructive.** Idempotent: deleting an already-gone checkpoint still returns success.
- **`harness-runtime-rollback-checkpoint`** — Rewind a session to a READY checkpoint. `SessionID` and `CheckpointID` required. **Destructive.** In-place restore — same `session_id`, new `sandbox_id`. The session must not have a workspace attached. The ingress URL is revoked — re-fetch after the session is READY.

### Sandbox

- **`harness-runtime-list-sandbox-sizes`** — List the available sandbox compute sizes (slug, vCPUs, memory, hourly and monthly pricing). The returned slugs are valid values for sandbox size fields in manifests.
- **`harness-runtime-validate-policy`** — Stateless preflight: check a manifest's tool-permission policy against the agent's capabilities without creating a session. Takes `Manifest` (same JSON object as create-agent-config, must include a `permissions` block). Returns per-rule verdicts (`exact`, `degraded`, `unsupported`, `unverified`); `ok=false` means CreateSession would reject this policy.

### Templates

Custom sandbox templates let you layer an OCI image on a base template for a customized agent environment.

- **`harness-runtime-list-templates`** — List the team's custom sandbox templates (id, name, status, spec, timestamps). Keyset-paginated. Only READY templates can be referenced in agent configs.
- **`harness-runtime-get-template`** — Get a single custom sandbox template by `TemplateID` (UUID), including its spec (base template + OCI image source) and status.
- **`harness-runtime-create-template`** — Register a new custom sandbox template. Requires `Name`, `BaseTemplate` (key like `codex-base`, `opencode-base`, `claude-code-base`, `hermes-base`, `hermes-byoa-base`, `langgraph-base`, `coding-base`, `crewai`), and `SourceOCIRef` (OCI image reference). The platform builds it asynchronously — poll until `TEMPLATE_STATUS_READY`.
- **`harness-runtime-update-template`** — Update a template's `SourceOCIRef` and/or `BaseTemplate` key. Triggers a new build; poll until READY. Returns the updated template.
- **`harness-runtime-delete-template`** — Permanently delete a custom sandbox template by `TemplateID` (UUID). **Destructive.** Agent configs referencing this template can no longer create sessions.
- **`harness-runtime-list-template-builds`** — List builds for a template (newest first). `TemplateID` required. Supports `PageSize` and `PageToken` cursor.
- **`harness-runtime-get-template-build`** — Get a single build by `TemplateID` and `BuildID` (both UUIDs). Includes status (PENDING, BUILDING, SUCCEEDED, FAILED), error message on failure, and the spec.

### Workspaces

Persistent workspaces are virtual disk volumes that survive across sessions.

- **`harness-runtime-create-workspace`** — Provision a new persistent workspace. `SizeGibibytes` (1-100, integer) required. Optional `Name` label. Size is fixed at creation.
- **`harness-runtime-list-workspaces`** — List the team's workspaces. Supports `PageSize`, `PageToken`, and `State` filter (`AVAILABLE`, `ATTACHING`, `ATTACHED`, `RELEASING`, `FAILED`, `DELETING`).
- **`harness-runtime-get-workspace`** — Get a single workspace by `WorkspaceID` (UUID), including state, attached session, size, bytes used, and timestamps.
- **`harness-runtime-delete-workspace`** — Permanently delete a workspace by `WorkspaceID`. **Destructive.** Must not be attached (409 if attached). All data is lost.

### Workspace Transfers

Bulk file transfer to/from a session's attached workspace via multipart upload or async download.

- **`harness-runtime-create-workspace-transfer`** — Initiate a transfer (`SessionID` + `Direction` = `upload` | `download`). Uploads return `transfer_id`, `upload_id`, `part_size`, `expires_at`. Downloads return `transfer_id` — poll until completed.
- **`harness-runtime-get-workspace-transfer`** — Get transfer status. For downloads, poll until `completed` to get `download_url` and `sha256`. For uploads, check `bytes_written`.
- **`harness-runtime-cancel-workspace-transfer`** — Abort an active transfer. Idempotent.
- **`harness-runtime-commit-workspace-upload`** — Finalize a multipart upload after all parts are PUT. Returns `size_bytes`.
- **`harness-runtime-create-workspace-part-urls`** — Get presigned PUT URLs for a set of part numbers. `PartNumbers` (array of positive integers) required. URLs expire at `expires_at`.

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
- Fork session 029ab39c-14d9-7080-933e-b9b90e25acda into 3 child sessions.
- Fork from checkpoint cp_9f2c1a4b of session 029ab39c-14d9-7080-933e-b9b90e25acda.
- Run `ls /workspace` in session 029ab39c-14d9-7080-933e-b9b90e25acda.
- Get the ingress URL for session 029ab39c-14d9-7080-933e-b9b90e25acda.
- Cancel the current turn in session 029ab39c-14d9-7080-933e-b9b90e25acda.
- List all checkpoints for session 029ab39c-14d9-7080-933e-b9b90e25acda.
- Create a checkpoint of session 029ab39c-14d9-7080-933e-b9b90e25acda labeled "before-refactor".
- Roll back session 029ab39c-14d9-7080-933e-b9b90e25acda to checkpoint cp_9f2c1a4b.
- Delete checkpoint cp_9f2c1a4b from session 029ab39c-14d9-7080-933e-b9b90e25acda.
- What sandbox sizes are available?
- Validate this policy against session 029ab39c-14d9-7080-933e-b9b90e25acda.
- List my custom sandbox templates.
- Show me template 049fb39c-14d9-7080-933e-b9b90e25acdc.
- Create a new template "my-python-env" from codex-base with my OCI image.
- Update template 049fb39c-14d9-7080-933e-b9b90e25acdc to use a new image tag.
- Delete template 049fb39c-14d9-7080-933e-b9b90e25acdc.
- List builds for template 049fb39c-14d9-7080-933e-b9b90e25acdc.
- Show build details for template 049fb39c-14d9-7080-933e-b9b90e25acdc build 059fb39c-14d9-7080-933e-b9b90e25acdd.
- Create a 10 GiB workspace named "project-data".
- List my workspaces.
- What's the status of workspace 069fb39c-14d9-7080-933e-b9b90e25acde?
- Delete workspace 069fb39c-14d9-7080-933e-b9b90e25acde.
- Start an upload transfer on session 029ab39c-14d9-7080-933e-b9b90e25acda.
- Check the status of transfer xfer-abc123 on session 029ab39c-14d9-7080-933e-b9b90e25acda.
- Get upload URLs for parts 1, 2, and 3 of transfer xfer-abc123 on session 029ab39c-14d9-7080-933e-b9b90e25acda.
- Commit transfer xfer-abc123 on session 029ab39c-14d9-7080-933e-b9b90e25acda.
- Cancel transfer xfer-abc123 on session 029ab39c-14d9-7080-933e-b9b90e25acda.
