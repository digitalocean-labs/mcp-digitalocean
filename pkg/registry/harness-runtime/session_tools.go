package harnessruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	sessionsPath = "v2/agents/sessions"
)

// Session is the user-facing record of a managed agent + sandbox pair.
type Session struct {
	SessionID             string     `json:"session_id"`
	Name                  string     `json:"name,omitempty"`
	ParentSessionID       string     `json:"parent_session_id,omitempty"`
	ForkID                string     `json:"fork_id,omitempty"`
	ForkFromCheckpointID  string     `json:"fork_from_checkpoint_id,omitempty"`
	ForkCutoffEventID     string     `json:"fork_cutoff_event_id,omitempty"`
	ForkedAt              *time.Time `json:"forked_at,omitempty"`
	AgentKind             string     `json:"agent_kind,omitempty"`
	Status                string     `json:"status,omitempty"`
	SandboxID             string     `json:"sandbox_id,omitempty"`
	CreatedAt             *time.Time `json:"created_at,omitempty"`
	LastEventAt           *time.Time `json:"last_event_at,omitempty"`
	KeepWarm              bool       `json:"keep_warm,omitempty"`
	ResumeOnTopoff        bool       `json:"resume_on_topoff,omitempty"`
	WorkspaceID           string     `json:"workspace_id,omitempty"`
	PauseReason           string     `json:"pause_reason,omitempty"`
	IdleTimeoutSeconds    string     `json:"idle_timeout_seconds,omitempty"`
	RepoHint              string     `json:"repo_hint,omitempty"`
	Warnings              []string   `json:"warnings,omitempty"`
	ConfigID              string     `json:"config_id,omitempty"`
	OpenAISessionID       string     `json:"openai_session_id,omitempty"`
	OpenAIEnvironmentID   string     `json:"openai_environment_id,omitempty"`
}

// CreateSessionResponse is the response of POST /v2/agents/sessions.
type CreateSessionResponse struct {
	Session Session `json:"session"`
}

// GetSessionResponse is the response of GET /v2/agents/sessions/{id}.
type GetSessionResponse struct {
	Session Session `json:"session"`
}

// ListSessionsResponse is the response of GET /v2/agents/sessions.
type ListSessionsResponse struct {
	Sessions      []Session `json:"sessions"`
	NextPageToken string    `json:"next_page_token,omitempty"`
}

// SendInputResponse is the response of POST /v2/agents/sessions/{id}/input.
type SendInputResponse struct {
	RunID string `json:"run_id,omitempty"`
}

// createSession provisions a new session using the JSON config-reference form.
func (t *Tool) createSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	configID, _ := args["ConfigID"].(string)
	if configID == "" {
		return mcp.NewToolResultError("ConfigID is required"), nil
	}
	if _, err := uuid.Parse(configID); err != nil {
		return mcp.NewToolResultError("ConfigID must be a UUID"), nil
	}
	name, _ := args["Name"].(string)
	if name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}

	body := map[string]any{
		"config_id": configID,
		"name":      name,
	}
	if v, ok := args["ResumeOnTopoff"].(bool); ok {
		body["resume_on_topoff"] = v
	}
	if v, _ := args["WorkspaceID"].(string); v != "" {
		body["workspace_id"] = v
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, sessionsPath, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out CreateSessionResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create session", err), nil
	}
	return sessionCreateOut.Result(out)
}

// getSession returns the current snapshot of one session.
func (t *Tool) getSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	sessionID, _ := req.GetArguments()["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, sessionsPath+"/"+sessionID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out GetSessionResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get session", err), nil
	}
	return sessionOut.Result(out)
}

// listSessions enumerates sessions for the caller's team.
func (t *Tool) listSessions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	query := url.Values{}

	if v, ok := args["PageSize"].(float64); ok {
		if v != math.Trunc(v) || v < 1 || v > maxPageSize {
			return mcp.NewToolResultError(fmt.Sprintf("PageSize must be an integer between 1 and %d", maxPageSize)), nil
		}
		query.Set("page_size", strconv.Itoa(int(v)))
	}
	if v, _ := args["PageToken"].(string); v != "" {
		query.Set("page_token", v)
	}
	if v, _ := args["Status"].(string); v != "" {
		query.Set("status", v)
	}
	if v, _ := args["Name"].(string); v != "" {
		query.Set("name", v)
	}
	if v, _ := args["Search"].(string); v != "" {
		query.Set("search", v)
	}

	path := sessionsPath
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListSessionsResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list sessions", err), nil
	}
	if out.Sessions == nil {
		out.Sessions = []Session{}
	}
	return sessionListOut.Result(out)
}

// destroySession tears down the sandbox and marks the session DESTROYED.
func (t *Tool) destroySession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	sessionID, _ := req.GetArguments()["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodDelete, sessionsPath+"/"+sessionID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	if _, err := client.Do(ctx, apiReq, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to destroy session", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Session %s destroyed successfully.", sessionID)), nil
}

// sendInput delivers a user chat message into the session.
func (t *Tool) sendInput(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	sessionID, _ := args["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}
	text, _ := args["Text"].(string)
	inputs, hasInputs := args["Inputs"].(map[string]any)
	if text == "" && !hasInputs {
		return mcp.NewToolResultError("At least one of Text or Inputs must be provided"), nil
	}

	body := map[string]any{}
	if text != "" {
		body["text"] = text
	}
	if hasInputs {
		body["inputs"] = inputs
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, sessionsPath+"/"+sessionID+"/input", nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out SendInputResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to send input", err), nil
	}
	return sendInputOut.Result(out)
}

// pauseSession freezes the session's microVM sandbox.
func (t *Tool) pauseSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	sessionID, _ := req.GetArguments()["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, sessionsPath+"/"+sessionID+"/pause", nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	if _, err := client.Do(ctx, apiReq, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to pause session", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Session %s paused successfully.", sessionID)), nil
}

// resumeSession explicitly thaws a PAUSED session's microVM sandbox.
func (t *Tool) resumeSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	sessionID, _ := req.GetArguments()["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, sessionsPath+"/"+sessionID+"/resume", nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	if _, err := client.Do(ctx, apiReq, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to resume session", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Session %s resumed successfully.", sessionID)), nil
}

// patchSession updates session-scoped settings on an existing session.
func (t *Tool) patchSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	sessionID, _ := args["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}

	resumeOnTopoff, hasResumeOnTopoff := args["ResumeOnTopoff"].(bool)
	if !hasResumeOnTopoff {
		return mcp.NewToolResultError("At least one patchable field must be provided (e.g. ResumeOnTopoff)"), nil
	}

	body := map[string]any{
		"resume_on_topoff": resumeOnTopoff,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPatch, sessionsPath+"/"+sessionID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out GetSessionResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to patch session", err), nil
	}
	return sessionOut.Result(out)
}

// resolveHITL records the user's decision on a pending HITL request.
func (t *Tool) resolveHITL(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	sessionID, _ := args["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}
	requestID, _ := args["RequestID"].(string)
	if requestID == "" {
		return mcp.NewToolResultError("RequestID is required"), nil
	}
	outcome, _ := args["Outcome"].(string)
	if outcome == "" {
		return mcp.NewToolResultError("Outcome is required"), nil
	}
	validOutcomes := map[string]bool{
		"HITL_OUTCOME_APPROVE": true,
		"HITL_OUTCOME_REJECT":  true,
		"HITL_OUTCOME_DEFER":   true,
	}
	if !validOutcomes[outcome] {
		return mcp.NewToolResultError("Outcome must be one of: HITL_OUTCOME_APPROVE, HITL_OUTCOME_REJECT, HITL_OUTCOME_DEFER"), nil
	}

	body := map[string]any{
		"outcome": outcome,
	}
	if v, _ := args["Reason"].(string); v != "" {
		body["reason"] = v
	}
	if v, ok := args["Content"].(map[string]any); ok && len(v) > 0 {
		body["content"] = v
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	path := sessionsPath + "/" + sessionID + "/hitl/" + requestID
	apiReq, err := client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	if _, err := client.Do(ctx, apiReq, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to resolve HITL request", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("HITL request %s on session %s resolved with outcome %s.", requestID, sessionID, outcome)), nil
}

// sessionTools returns the session MCP tools to be merged into Tools().
func (t *Tool) sessionTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.createSession,
			Tool: mcp.NewTool("harness-runtime-create-session",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				sessionCreateOut.Schema(),
				mcp.WithDescription("Provision a new hosted agent session from an existing Agent Config. Returns the session row immediately; sandbox boot completes asynchronously. Poll harness-runtime-get-session until status is SESSION_STATUS_READY before sending input."),
				mcp.WithString("ConfigID", mcp.Required(), mcp.Description("Agent Config UUID to create the session from (use harness-runtime-list-agent-configs to find one)")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Session name (1-64 characters; must not be UUID-shaped; must be unique among live sessions for the team)")),
				mcp.WithBoolean("ResumeOnTopoff", mcp.Description("Opt the session in to automatic resumption when the team's prepayment balance is restored (default false)")),
				mcp.WithString("WorkspaceID", mcp.Description("Persistent workspace to attach (1-64 non-whitespace characters; not GA for all teams)")),
			),
		},
		{
			Handler: t.getSession,
			Tool: mcp.NewTool("harness-runtime-get-session",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				sessionOut.Schema(),
				mcp.WithDescription("Get the current snapshot of a hosted agent session by its UUID, including status, agent kind, sandbox id, and lineage fields."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID")),
			),
		},
		{
			Handler: t.listSessions,
			Tool: mcp.NewTool("harness-runtime-list-sessions",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				sessionListOut.Schema(),
				mcp.WithDescription("List hosted agent sessions for the team. By default omits destroyed and failed sessions; pass Status=SESSION_STATUS_ALL to include every session. Keyset-paginated; pass next_page_token back as PageToken for the next page."),
				mcp.WithNumber("PageSize", mcp.Description("Page size (1-200, default 50)")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
				mcp.WithString("Status", mcp.Description("Optional status filter: SESSION_STATUS_UNSPECIFIED, SESSION_STATUS_PROVISIONING, SESSION_STATUS_READY, SESSION_STATUS_DETACHED, SESSION_STATUS_DESTROYING, SESSION_STATUS_DESTROYED, SESSION_STATUS_FAILED, SESSION_STATUS_PAUSED, SESSION_STATUS_ALL")),
				mcp.WithString("Name", mcp.Description("Optional exact-name filter (case-insensitive) — resolves to the single live session currently holding this name")),
				mcp.WithString("Search", mcp.Description("Optional case-insensitive substring filter on session name (max 64 characters); may return multiple results")),
			),
		},
		{
			Handler: t.destroySession,
			Tool: mcp.NewTool("harness-runtime-destroy-session",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Tear down the sandbox and permanently mark the session DESTROYED. Persistent workspace contents are retained per tenant policy. Returns 204 on success."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID to destroy")),
			),
		},
		{
			Handler: t.sendInput,
			Tool: mcp.NewTool("harness-runtime-send-input",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				sendInputOut.Schema(),
				mcp.WithDescription("Deliver a user chat message into a READY or DETACHED session. The agent observes it on its next turn and emits events on the data-plane stream. Returns the run_id the input was attributed to."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID")),
				mcp.WithString("Text", mcp.Description("The user message to deliver (required if Inputs is not provided)")),
				mcp.WithObject("Inputs", mcp.Description("Optional structured kickoff inputs for agents whose entry point takes a dict (e.g. CrewAI); flat object of string/number/boolean values")),
			),
		},
		{
			Handler: t.pauseSession,
			Tool: mcp.NewTool("harness-runtime-pause-session",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Freeze a READY or DETACHED session's microVM sandbox. Supported on microvm-substrate sessions only. Paused sessions still count against quota. Returns 204 on success."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID to pause")),
			),
		},
		{
			Handler: t.resumeSession,
			Tool: mcp.NewTool("harness-runtime-resume-session",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Explicitly thaw a PAUSED session's microVM sandbox. Returns synchronously once the sandbox is ready. Returns 204 on success."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID to resume")),
			),
		},
		{
			Handler: t.patchSession,
			Tool: mcp.NewTool("harness-runtime-patch-session",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskLow),
				sessionOut.Schema(),
				mcp.WithDescription("Update session-scoped settings on an existing (non-terminal) session. Currently supports toggling resume_on_topoff (auto-resume consent after a low-balance pause). Returns the updated session snapshot."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID to patch")),
				mcp.WithBoolean("ResumeOnTopoff", mcp.Required(), mcp.Description("Set whether this session opts in to automatic resumption when the team's prepayment balance is restored")),
			),
		},
		{
			Handler: t.resolveHITL,
			Tool: mcp.NewTool("harness-runtime-resolve-hitl",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Record the user's decision on a pending Human-in-the-Loop (HITL) request. Unblocks the gated agent action and echoes the resolution to all attached clients. Returns 204 on success."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID that owns the HITL request")),
				mcp.WithString("RequestID", mcp.Required(), mcp.Description("The HITL request id to resolve")),
				mcp.WithString("Outcome", mcp.Required(), mcp.Description("Decision: HITL_OUTCOME_APPROVE, HITL_OUTCOME_REJECT, or HITL_OUTCOME_DEFER (back off without rejecting)")),
				mcp.WithString("Reason", mcp.Description("Optional free-text reason captured in the audit trail")),
				mcp.WithObject("Content", mcp.Description("Optional structured answer for a gated request that asks for DATA (e.g. form elicitation); a flat object matching the requestedSchema")),
			),
		},
	}
}
